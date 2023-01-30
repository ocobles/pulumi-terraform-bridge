// Copyright 2016-2022, Pulumi Corporation.
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

package testprovider

import (
	"fmt"
	"path/filepath"
	"unicode"

	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"

	"github.com/terraform-providers/terraform-provider-random/randomshim"

	tfpf "github.com/pulumi/pulumi-terraform-bridge/pf/tfbridge"
	"github.com/pulumi/pulumi-terraform-bridge/v3/pkg/tfbridge"
)

// Adapts Random provider to tfbridge for testing tfbridge against a
// realistic provider.
func RandomProvider() tfpf.ProviderInfo {
	randomPkg := "random"
	randomMod := "index"

	// randomMember manufactures a type token for the random package and the given module and type.
	randomMember := func(mod string, mem string) tokens.ModuleMember {
		return tokens.ModuleMember(randomPkg + ":" + mod + ":" + mem)
	}

	// randomType manufactures a type token for the random package and the given module and type.
	randomType := func(mod string, typ string) tokens.Type {
		return tokens.Type(randomMember(mod, typ))
	}

	// randomResource manufactures a standard resource token given a module and resource name.  It automatically uses the
	// random package and names the file by simply lower casing the resource's first character.
	randomResource := func(mod string, res string) tokens.Type {
		fn := string(unicode.ToLower(rune(res[0]))) + res[1:]
		return randomType(mod+"/"+fn, res)
	}

	info := tfbridge.ProviderInfo{
		Name:        "random",
		Description: "A Pulumi package to safely use randomness in Pulumi programs.",
		Keywords:    []string{"pulumi", "random"},
		License:     "Apache-2.0",
		Homepage:    "https://pulumi.io",
		Repository:  "https://github.com/pulumi/pulumi-random",
		Version:     "4.8.2",
		Resources: map[string]*tfbridge.ResourceInfo{
			"random_id":       {Tok: randomResource(randomMod, "RandomId")},
			"random_password": {Tok: randomResource(randomMod, "RandomPassword")},
			"random_pet":      {Tok: randomResource(randomMod, "RandomPet")},
			"random_shuffle":  {Tok: randomResource(randomMod, "RandomShuffle")},
			"random_string":   {Tok: randomResource(randomMod, "RandomString")},
			"random_integer":  {Tok: randomResource(randomMod, "RandomInteger")},
			"random_uuid":     {Tok: randomResource(randomMod, "RandomUuid")},
		},
		JavaScript: &tfbridge.JavaScriptInfo{
			Dependencies: map[string]string{
				"@pulumi/pulumi": "^3.0.0",
			},
			DevDependencies: map[string]string{
				"@types/node": "^10.0.0", // so we can access strongly typed node definitions.
			},
		},
		Python: &tfbridge.PythonInfo{
			Requires: map[string]string{
				"pulumi": ">=3.0.0,<4.0.0",
			},
		},
		Golang: &tfbridge.GolangInfo{
			ImportBasePath: filepath.Join(
				fmt.Sprintf("github.com/pulumi/pulumi-%[1]s/sdk/", randomPkg),
				tfbridge.GetModuleMajorVersion("0.0.1"),
				"go",
				randomPkg,
			),
			GenerateResourceContainerTypes: true,
		},
		CSharp: &tfbridge.CSharpInfo{
			PackageReferences: map[string]string{
				"Pulumi": "3.*",
			},
			Namespaces: map[string]string{
				"random": "Random",
			},
		},
	}

	return tfpf.ProviderInfo{
		ProviderInfo: info,
		NewProvider:  randomshim.NewProvider,
	}
}