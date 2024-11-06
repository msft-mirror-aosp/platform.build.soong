// Copyright 2024 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package tradefed_modules

import (
	"android/soong/android"
	"android/soong/java"
	"testing"
)

func TestTestSuites(t *testing.T) {
	t.Parallel()
	ctx := android.GroupFixturePreparers(
		java.PrepareForTestWithJavaDefaultModules,
		android.FixtureRegisterWithContext(RegisterTestSuiteBuildComponents),
	).RunTestWithBp(t, `
		android_test {
			name: "TestModule1",
			sdk_version: "current",
		}

		android_test {
			name: "TestModule2",
			sdk_version: "current",
		}

		test_suite {
			name: "my-suite",
			description: "a test suite",
			tests: [
				"TestModule1",
				"TestModule2",
			]
		}
	`)
	manifestPath := ctx.ModuleForTests("my-suite", "").Output("out/soong/test_suites/my-suite/my-suite.json")
	got := android.ContentFromFileRuleForTests(t, ctx.TestContext, manifestPath)
	want := `{"name": "my-suite"}` + "\n"
	if got != want {
		t.Errorf("my-suite.json content was %q, want %q", got, want)
	}
}
