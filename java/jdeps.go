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

package java

import (
	"encoding/json"
	"fmt"

	"android/soong/android"
)

// This singleton generates android java dependency into to a json file. It does so for each
// blueprint Android.bp resulting in a java.Module when either make, mm, mma, mmm or mmma is
// called. Dependency info file is generated in $OUT/module_bp_java_depend.json.

func init() {
	android.RegisterParallelSingletonType("jdeps_generator", jDepsGeneratorSingleton)
}

func jDepsGeneratorSingleton() android.Singleton {
	return &jdepsGeneratorSingleton{}
}

type jdepsGeneratorSingleton struct {
	outputPath android.Path
}

const (
	jdepsJsonFileName = "module_bp_java_deps.json"
)

func (j *jdepsGeneratorSingleton) GenerateBuildActions(ctx android.SingletonContext) {
	// (b/204397180) Generate module_bp_java_deps.json by default.
	moduleInfos := make(map[string]android.IdeInfo)

	ctx.VisitAllModuleProxies(func(module android.ModuleProxy) {
		if !android.OtherModuleProviderOrDefault(ctx, module, android.CommonModuleInfoKey).Enabled {
			return
		}

		// Prevent including both prebuilts and matching source modules when one replaces the other.
		if !android.IsModulePreferredProxy(ctx, module) {
			return
		}

		ideInfoProvider, ok := android.OtherModuleProvider(ctx, module, android.IdeInfoProviderKey)
		if !ok {
			return
		}
		name := ideInfoProvider.BaseModuleName
		if info, ok := android.OtherModuleProvider(ctx, module, JavaLibraryInfoProvider); ok && info.Prebuilt {
			// TODO(b/113562217): Extract the base module name from the Import name, often the Import name
			// has a prefix "prebuilt_". Remove the prefix explicitly if needed until we find a better
			// solution to get the Import name.
			name = android.RemoveOptionalPrebuiltPrefix(module.Name())
		}

		dpInfo := moduleInfos[name]
		dpInfo = dpInfo.Merge(ideInfoProvider)
		dpInfo.Paths = []string{ctx.ModuleDir(module)}
		moduleInfos[name] = dpInfo

		mkProvider, ok := android.OtherModuleProvider(ctx, module, android.AndroidMkDataInfoProvider)
		if !ok {
			return
		}
		if mkProvider.Class != "" {
			dpInfo.Classes = append(dpInfo.Classes, mkProvider.Class)
		}

		if dep, ok := android.OtherModuleProvider(ctx, module, JavaInfoProvider); ok {
			dpInfo.Installed_paths = append(dpInfo.Installed_paths, dep.ImplementationJars.Strings()...)
		}
		dpInfo.Classes = android.FirstUniqueStrings(dpInfo.Classes)
		dpInfo.Installed_paths = android.FirstUniqueStrings(dpInfo.Installed_paths)
		moduleInfos[name] = dpInfo
	})

	jfpath := android.PathForOutput(ctx, jdepsJsonFileName)
	err := createJsonFile(moduleInfos, jfpath)
	if err != nil {
		ctx.Errorf(err.Error())
	}
	j.outputPath = jfpath

	// This is necessary to satisfy the dangling rules check as this file is written by Soong rather than a rule.
	ctx.Build(pctx, android.BuildParams{
		Rule:   android.Touch,
		Output: jfpath,
	})
	ctx.DistForGoals([]string{"general-tests", "dist_files"}, j.outputPath)
}

func createJsonFile(moduleInfos map[string]android.IdeInfo, jfpath android.WritablePath) error {
	buf, err := json.MarshalIndent(moduleInfos, "", "\t")
	if err != nil {
		return fmt.Errorf("JSON marshal of java deps failed: %s", err)
	}
	err = android.WriteFileToOutputDir(jfpath, buf, 0666)
	if err != nil {
		return fmt.Errorf("Writing java deps to %s failed: %s", jfpath.String(), err)
	}
	return nil
}
