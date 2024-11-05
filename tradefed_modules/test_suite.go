// Copyright 2024 Google Inc. All rights reserved.
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

package tradefed_modules

import (
	"android/soong/android"
)

func init() {
	RegisterTestSuiteBuildComponents(android.InitRegistrationContext)
}

func RegisterTestSuiteBuildComponents(ctx android.RegistrationContext) {
	ctx.RegisterModuleType("test_suite", TestSuiteFactory)
}

var PrepareForTestWithTestSuiteBuildComponents = android.GroupFixturePreparers(
	android.FixtureRegisterWithContext(RegisterTestSuiteBuildComponents),
)

type testSuiteProperties struct {
	Description string
	Tests []string `android:"path,arch_variant"`
}

type testSuiteModule struct {
	android.ModuleBase
	android.DefaultableModuleBase
	testSuiteProperties
}

func (t *testSuiteModule) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	// TODO(hwj): Implement this.
}

func TestSuiteFactory() android.Module {
	module := &testSuiteModule{}
	module.AddProperties(&module.testSuiteProperties)

	android.InitAndroidModule(module)
	android.InitDefaultableModule(module)

	return module
}
