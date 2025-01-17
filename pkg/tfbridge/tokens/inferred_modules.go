// Copyright 2016-2023, Pulumi Corporation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tokens

import (
	"fmt"
	"strings"

	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/common/util/contract"

	b "github.com/pulumi/pulumi-terraform-bridge/v3/pkg/tfbridge"
	shim "github.com/pulumi/pulumi-terraform-bridge/v3/pkg/tfshim"
)

type InferredModulesOpts struct {
	// The TF prefix of the package.
	TfPkgPrefix string
	// The name of the main module. Defaults to "index".
	MainModule string
	// The minimum number of shared items for a prefix before it becomes a module.
	//
	// < 0 -> don't bin into modules.
	// = 0 -> apply the default value.
	// > 0 -> set the value.
	MinimumModuleSize int
	// The number of items in a longer prefix needed to break out into it's own prefix.
	//
	// For example, with the tokens `pkg_mod_sub1_a`, `pkg_mod_sub2_b`, `pkg_mod_sub2_c`,
	// `pkg_mod_sub3_d`:
	//
	// MinimumSubmoduleSize = 3 will result in:
	//
	//	pkg:mod:Sub1A, pkg:mod:Sub2B, pkg:mod:Sub2C, pkg:mod:Sub3D
	//
	// MinimumSubmoduleSize = 2 will result in:
	//
	//	pkg:mod:Sub1A, pkg:modSub2:B, pkg:modSub2C, pkg:mod:Sub3D
	//
	// < 0 -> don't bin into submodules. Only the most common prefix will be used.
	// = 0 -> apply the default value.
	// > 0 -> set the value.
	MimimumSubmoduleSize int
}

// A strategy to infer module placement from global analysis of all items (Resources & DataSources).
func InferredModules(
	info *b.ProviderInfo, finalize Make, opts *InferredModulesOpts,
) (b.Strategy, error) {
	if opts == nil {
		opts = &InferredModulesOpts{}
	}
	err := opts.ensurePrefix(info)
	if err != nil {
		return b.Strategy{}, fmt.Errorf("inferring pkg prefix: %w", err)
	}
	contract.Assertf(opts.MinimumModuleSize >= 0, "Cannot have a minimum modules size less then zero")
	if opts.MinimumModuleSize == 0 {
		opts.MinimumModuleSize = defaultMinimumModuleSize
	}
	if opts.MimimumSubmoduleSize == 0 {
		opts.MimimumSubmoduleSize = defaultMinimumSubmoduleSize
	}
	if opts.MainModule == "" {
		opts.MainModule = "index"
	}

	tokenMap := opts.computeTokens(info)

	rIsEmpty := func(r *b.ResourceInfo) bool { return r.Tok == "" }
	dIsEmpty := func(r *b.DataSourceInfo) bool { return r.Tok == "" }

	return b.Strategy{
		Resource: tokenFromMap(tokenMap, rIsEmpty, finalize, func(tk string, resource *b.ResourceInfo) {
			checkedApply(&resource.Tok, tokens.Type(tk))
		}),
		DataSource: tokenFromMap(tokenMap, dIsEmpty, finalize, func(tk string, datasource *b.DataSourceInfo) {
			checkedApply(&datasource.Tok, tokens.ModuleMember(tk))
		}),
	}, nil
}

func (opts *InferredModulesOpts) ensurePrefix(info *b.ProviderInfo) error {
	prefix := opts.TfPkgPrefix
	var noCommonality bool
	findPrefix := func(key string, _ shim.Resource) bool {
		if noCommonality {
			return false
		}
		if prefix == "" {
			prefix = key
			return true
		}

		prefix = sharedPrefix(key, prefix)
		if prefix == "" {
			noCommonality = true
		}

		return true
	}
	mapProviderItems(info, findPrefix)
	if noCommonality {
		return fmt.Errorf("no common prefix detected")
	}
	if prefix == "" {
		return fmt.Errorf("no items found")
	}
	opts.TfPkgPrefix = prefix
	return nil
}

type node struct {
	segment  string
	children map[string]*node
	// tfToken is only non-empty if the node represents a literal tf token
	tfToken string
}

func (n *node) child(segment string) *node {
	if n.children == nil {
		n.children = map[string]*node{}
	}
	v, ok := n.children[segment]
	if ok {
		return v
	}
	child := &node{segment: segment}
	n.children[segment] = child
	return child
}

func (n *node) insert(child *node) {
	if n.children == nil {
		n.children = map[string]*node{}
	}
	_, ok := n.children[child.segment]
	contract.Assertf(!ok, "duplicate segment in child: %q", child.segment)
	n.children[child.segment] = child
}

func (n *node) len() int {
	i := 0
	if n.tfToken != "" {
		i++
	}
	for _, child := range n.children {
		i += child.len()
	}
	return i
}

// A depth first search of child nodes.
//
// parent is a function that returns parent nodes, with the immediate parent starting at 0
// and each increment increasing the indirection. 1 yields the grandparent, 2 the
// great-grandparent, etc. parent panics when no node is available.
//
// dfs will pick up nodes inserted up the hierarchy during traversal, but only if they
// were inserted with unique names.
func (n *node) dfs(iter func(parent func(int) *node, node *node)) {
	parentStack := []*node{n}
	fullIter(n.children, func(_ string, child *node) {
		child.dfsInner(&parentStack, iter)
	})
}

// Iterate over a map in any order, ensuring that all keys in the map are iterated over,
// even if they were added during the iteration.
//
// There is no guarantee of the order of the iteration.
func fullIter[K comparable, V any](m map[K]V, f func(K, V)) {
	seen := map[K]bool{}
	for done := false; !done; {
		done = true
		for k, v := range m {
			if seen[k] {
				continue
			}
			seen[k] = true
			done = false

			f(k, v)
		}
	}
}

func (n *node) dfsInner(parentStack *[]*node, iter func(parent func(int) *node, node *node)) {
	// Pop this node onto the parent stack so children can access it
	*parentStack = append(*parentStack, n)
	// Iterate over children by key, making sure that newly added keys are iterated over
	fullIter(n.children, func(k string, v *node) {
		v.dfsInner(parentStack, iter)
	})

	// Pop the node off afterwards
	*parentStack = (*parentStack)[:len(*parentStack)-1]

	iter(func(i int) *node { return (*parentStack)[len(*parentStack)-1-i] }, n)
}

// Precompute the mapping from tf tokens to pulumi modules.
//
// The resulting map is complete for all TF resources and datasources in info.P.
func (opts *InferredModulesOpts) computeTokens(info *b.ProviderInfo) map[string]tokenInfo {
	contract.Assertf(opts.TfPkgPrefix != "", "TF package prefix not provided or computed")
	tree := &node{segment: opts.TfPkgPrefix}

	// Build segment tree:
	//
	// Expand each item (resource | datasource) into it's segments (divided by "_"), then
	// insert each token into the tree structure. The tree is defined by segments, where
	// each node represents a segment and each path a token.
	mapProviderItems(info, func(s string, _ shim.Resource) bool {
		segments := strings.Split(strings.TrimPrefix(s, opts.TfPkgPrefix), "_")
		contract.Assertf(len(segments) > 0, "No segments found")
		contract.Assertf(segments[0] != "", "Empty segment from splitting %q with prefix %q", s, opts.TfPkgPrefix)
		node := tree
		for _, segment := range segments {
			node = node.child(segment)
		}
		node.tfToken = s
		return true
	})

	contract.Assertf(tree.tfToken == "", "We don't expect a resource called '%s'", opts.TfPkgPrefix)
	output := map[string]tokenInfo{}
	// Collapse the segment tree via a depth first traversal.
	tree.dfs(func(parent func(int) *node, n *node) {
		if parent(0) == tree {
			// Inject each path as a node
			if n.len() < opts.MinimumModuleSize {
				// Node segment is not big enough for its own module, so inject each token
				// into the main module
				for _, child := range n.children {
					output[child.tfToken] = tokenInfo{
						mod:  opts.MainModule,
						name: n.segment + "_" + child.segment,
					}
				}
				if n.tfToken != "" {
					output[n.tfToken] = tokenInfo{
						mod:  opts.MainModule,
						name: n.segment,
					}
				}
			} else {
				// Node segment will form its own modules, so inject each token as a
				// module member of `n.segment`.
				for _, child := range n.children {
					contract.Assertf(child.tfToken != "", "child of %q: %#v", n.segment, child)
					output[child.tfToken] = tokenInfo{
						mod:  n.segment,
						name: child.segment,
					}
				}
				// If the node is both a module and a item, put the item in the module
				if n.tfToken != "" {
					output[n.tfToken] = tokenInfo{
						mod:  n.segment,
						name: n.segment,
					}
				}
			}
		} else {
			// flatten the tree by injecting children into the parent node.
			if n.len() < opts.MimimumSubmoduleSize {
				for _, child := range n.children {
					contract.Assertf(child.children == nil, "module already flattened")
					parent(0).insert(&node{
						segment: n.segment + "_" + child.segment,
						tfToken: child.tfToken,
					})
				}
				// Clear the children, since they have been moved to the parent
				n.children = nil
				if n.tfToken == "" {
					// If this is only a leaf node, we can cut it
					delete(parent(0).children, n.segment)
				}
			} else {
				// Inject the node into the grand-parent, putting it next to the parent
				// and remove it as a child of parent.
				delete(parent(0).children, n.segment)
				parent(1).insert(&node{
					segment:  parent(0).segment + "_" + n.segment,
					tfToken:  n.tfToken,
					children: n.children,
				})
			}
		}
	})

	return output
}

func ignoredTokens(info *b.ProviderInfo) map[string]bool {
	ignored := map[string]bool{}
	if info == nil {
		return ignored
	}
	for _, tk := range info.IgnoreMappings {
		ignored[tk] = true
	}
	return ignored
}

func mapProviderItems(info *b.ProviderInfo, each func(string, shim.Resource) bool) {
	ignored := ignoredTokens(info)
	info.P.ResourcesMap().Range(func(key string, value shim.Resource) bool {
		if ignored[key] {
			return true
		}
		return each(key, value)
	})
	info.P.DataSourcesMap().Range(func(key string, value shim.Resource) bool {
		if ignored[key] {
			return true
		}
		return each(key, value)
	})
}

func sharedPrefix(s1, s2 string) string {
	// Shorten the longer string so it is only as long as the shortest string
	if len(s1) < len(s2) {
		s2 = s2[:len(s1)]
	} else if len(s1) > len(s2) {
		s1 = s1[:len(s2)]
	}

	for i := range s1 {
		if s1[i] != s2[i] {
			return s1[:i]
		}
	}
	return s1
}

type tokenInfo struct{ mod, name string }

func tokenFromMap[T b.ResourceInfo | b.DataSourceInfo](
	tokenMap map[string]tokenInfo, isEmpty func(*T) bool,
	finalize Make, apply func(tk string, elem *T),
) b.ElementStrategy[T] {
	return func(tfToken string, elem *T) error {
		if !isEmpty(elem) {
			return nil
		}
		info, ok := tokenMap[tfToken]
		if !ok {
			existing := []string{}
			for k := range tokenMap {
				existing = append(existing, k)
			}
			return fmt.Errorf("TF token '%s' not present when prefix computed, found %#v", tfToken, existing)
		}
		tk, err := finalize(camelCase(info.mod), upperCamelCase(info.name))
		if err != nil {
			return err
		}
		apply(tk, elem)
		return nil
	}
}
