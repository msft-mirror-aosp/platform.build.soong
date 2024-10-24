// Copyright 2018 Google Inc. All rights reserved.
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

package sdk

import (
	"testing"

	"android/soong/android"
	"android/soong/genrule"
	"android/soong/java"
)

func TestSdkGenrule(t *testing.T) {
	// Test that a genrule can depend on an sdk if using common_os_srcs
	bp := `
				sdk {
					name: "my_sdk",
				}
				genrule {
					name: "my_regular_genrule",
					common_os_srcs: [":my_sdk"],
					out: ["out"],
					cmd: "cp $(in) $(out)",
				}
			`
	android.GroupFixturePreparers(
		// if java components aren't registered, the sdk module doesn't create a snapshot for some reason.
		java.PrepareForTestWithJavaBuildComponents,
		genrule.PrepareForTestWithGenRuleBuildComponents,
		PrepareForTestWithSdkBuildComponents,
		android.FixtureRegisterWithContext(registerGenRuleBuildComponents),
	).RunTestWithBp(t, bp)
}
