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

package tfbridgetests

import (
	"testing"

	"github.com/pulumi/pulumi-terraform-bridge/pf/tests/internal/testprovider"
	testutils "github.com/pulumi/pulumi-terraform-bridge/testing/x"
	tfbridge "github.com/pulumi/pulumi-terraform-bridge/v3/pkg/tfbridge"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
)

func TestPreStateUpgradeHook(t *testing.T) {
	info := testprovider.RandomProvider()
	info.Resources["random_string"].PreStateUpgradeHook = func(args tfbridge.PreStateUpgradeHookArgs) (int64, resource.PropertyMap, error) {
		// Assume that if prior state is missing a schema version marker, it really is a corrupt state at version 2.
		if args.PriorStateSchemaVersion == 0 {
			return 2, args.PriorState, nil
		}
		// Otherwise proceed as ususal without modification.
		return args.PriorStateSchemaVersion, args.PriorState, nil
	}

	server := newProviderServer(t, info)
	testutils.Replay(t, server, `
	{
	  "method": "/pulumirpc.ResourceProvider/Update",
	  "request": {
	    "id": "0",
	    "urn": "urn:pulumi:dev::repro-pulumi-random::random:index/randomString:RandomString::s",
	    "olds": {
	      "length": 1,
	      "result": "x"
	    },
	    "news": {
	      "length": 2
	    }
	  },
	  "response": {
	    "properties": {
	      "__meta": "{\"schema_version\":\"2\"}",
	      "id": "*",
	      "length": 2,
	      "lower": true,
	      "minLower": 0,
	      "minNumeric": 0,
	      "minSpecial": 0,
	      "minUpper": 0,
	      "number": true,
	      "numeric": true,
	      "result": "*",
	      "special": true,
	      "upper": true
	    }
	  }
	}
        `)

}
