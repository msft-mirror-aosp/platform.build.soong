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

package etc

import (
	"android/soong/android"
)

func init() {
	android.RegisterModuleType("adb_keys", AdbKeysModuleFactory)
}

type AdbKeysModule struct {
	android.ModuleBase
	outputPath  android.OutputPath
	installPath android.InstallPath
}

func AdbKeysModuleFactory() android.Module {
	module := &AdbKeysModule{}
	android.InitAndroidArchModule(module, android.DeviceSupported, android.MultilibFirst)
	return module
}

func (m *AdbKeysModule) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	productVariables := ctx.Config().ProductVariables()
	if !(android.Bool(productVariables.Debuggable) && len(android.String(productVariables.AdbKeys)) > 0) {
		m.Disable()
		m.SkipInstall()
		return
	}

	m.outputPath = android.PathForModuleOut(ctx, "adb_keys").OutputPath
	input := android.ExistentPathForSource(ctx, android.String(productVariables.AdbKeys))
	ctx.Build(pctx, android.BuildParams{
		Rule:   android.Cp,
		Output: m.outputPath,
		Input:  input.Path(),
	})
	m.installPath = android.PathForModuleInPartitionInstall(ctx, ctx.DeviceConfig().ProductPath(), "etc/security")
	ctx.InstallFile(m.installPath, "adb_keys", m.outputPath)
}

func (m *AdbKeysModule) AndroidMkEntries() []android.AndroidMkEntries {
	if m.IsSkipInstall() {
		return []android.AndroidMkEntries{}
	}

	return []android.AndroidMkEntries{
		{
			Class:      "ETC",
			OutputFile: android.OptionalPathForPath(m.outputPath),
		}}
}
