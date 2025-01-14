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

package tfbridge

import (
	"context"

	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource/plugin"
)

// DiffConfig checks what impacts a hypothetical change to this provider's configuration will have on the provider.
func (p *provider) DiffConfigWithContext(ctx context.Context,
	urn resource.URN, olds, news resource.PropertyMap,
	allowUnknowns bool, ignoredChanges []string) (plugin.DiffResult, error) {

	// ctx = p.initLogging(ctx, p.logSink, urn)

	// TODO[pulumi/pulumi-terraform-bridge#825] implement properly.

	// This needs to include ignoredChanges:
	// checkedInputs, err := applyIgnoreChanges(olds, news, ignoredChanges)
	// if err != nil {
	// 	return plugin.DiffResult{}, fmt.Errorf("failed to apply ignore changes: %w", err)
	// }

	return plugin.DiffResult{}, nil
}
