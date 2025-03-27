// Copyright 2020 Google Inc. All rights reserved.
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

package android

import (
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/google/blueprint"
)

func init() {
	RegisterParallelSingletonType("testsuites", testSuiteFilesFactory)
}

func testSuiteFilesFactory() Singleton {
	return &testSuiteFiles{}
}

type testSuiteFiles struct{}

type TestSuiteModule interface {
	Module
	TestSuites() []string
}

type TestSuiteInfo struct {
	// A suffix to append to the name of the test.
	// Useful because historically different variants of soong modules became differently-named
	// make modules, like "my_test.vendor" for the vendor variant.
	NameSuffix string

	TestSuites []string

	NeedsArchFolder bool

	MainFile Path

	MainFileStem string

	MainFileExt string

	ConfigFile Path

	ConfigFileSuffix string

	ExtraConfigs Paths

	PerTestcaseDirectory bool

	Data []DataPath

	NonArchData []DataPath

	CompatibilitySupportFiles []Path
}

var TestSuiteInfoProvider = blueprint.NewProvider[TestSuiteInfo]()

type SupportFilesInfo struct {
	SupportFiles InstallPaths
}

var SupportFilesInfoProvider = blueprint.NewProvider[SupportFilesInfo]()

type filePair struct {
	src Path
	dst WritablePath
}

type testSuiteInstallsInfo struct {
	Files []filePair
}

var testSuiteInstallsInfoProvider = blueprint.NewProvider[testSuiteInstallsInfo]()

func (t *testSuiteFiles) GenerateBuildActions(ctx SingletonContext) {
	files := make(map[string]map[string]InstallPaths)
	var toInstall []filePair

	ctx.VisitAllModuleProxies(func(m ModuleProxy) {
		if tsm, ok := OtherModuleProvider(ctx, m, TestSuiteInfoProvider); ok {
			for _, testSuite := range tsm.TestSuites {
				if files[testSuite] == nil {
					files[testSuite] = make(map[string]InstallPaths)
				}
				name := ctx.ModuleName(m)
				files[testSuite][name] = append(files[testSuite][name],
					OtherModuleProviderOrDefault(ctx, m, InstallFilesProvider).InstallFiles...)
			}
		}
		if testSuiteInstalls, ok := OtherModuleProvider(ctx, m, testSuiteInstallsInfoProvider); ok {
			installs := OtherModuleProviderOrDefault(ctx, m, InstallFilesProvider).InstallFiles
			for _, f := range testSuiteInstalls.Files {
				alreadyInstalled := false
				for _, install := range installs {
					if install.String() == f.dst.String() {
						alreadyInstalled = true
						break
					}
				}
				if !alreadyInstalled {
					toInstall = append(toInstall, f)
				}
			}
		}
	})

	sort.Slice(toInstall, func(i, j int) bool {
		c := strings.Compare(toInstall[i].src.String(), toInstall[j].src.String())
		if c < 0 {
			return true
		} else if c > 0 {
			return false
		}
		return toInstall[i].dst.String() < toInstall[j].dst.String()
	})
	// Dedup, as multiple tests may install the same test data to the same folder
	toInstall = slices.Compact(toInstall)

	for _, install := range toInstall {
		ctx.Build(pctx, BuildParams{
			Rule:   Cp,
			Input:  install.src,
			Output: install.dst,
		})
	}

	robolectricZip, robolectrictListZip := buildTestSuite(ctx, "robolectric-tests", files["robolectric-tests"])
	ctx.Phony("robolectric-tests", robolectricZip, robolectrictListZip)
	ctx.DistForGoal("robolectric-tests", robolectricZip, robolectrictListZip)

	ravenwoodZip, ravenwoodListZip := buildTestSuite(ctx, "ravenwood-tests", files["ravenwood-tests"])
	ctx.Phony("ravenwood-tests", ravenwoodZip, ravenwoodListZip)
	ctx.DistForGoal("ravenwood-tests", ravenwoodZip, ravenwoodListZip)
}

func buildTestSuite(ctx SingletonContext, suiteName string, files map[string]InstallPaths) (Path, Path) {
	var installedPaths InstallPaths
	for _, module := range SortedKeys(files) {
		installedPaths = append(installedPaths, files[module]...)
	}

	outputFile := pathForPackaging(ctx, suiteName+".zip")
	rule := NewRuleBuilder(pctx, ctx)
	rule.Command().BuiltTool("soong_zip").
		FlagWithOutput("-o ", outputFile).
		FlagWithArg("-P ", "host/testcases").
		FlagWithArg("-C ", pathForTestCases(ctx).String()).
		FlagWithRspFileInputList("-r ", outputFile.ReplaceExtension(ctx, "rsp"), installedPaths.Paths()).
		Flag("-sha256") // necessary to save cas_uploader's time

	testList := buildTestList(ctx, suiteName+"_list", installedPaths)
	testListZipOutputFile := pathForPackaging(ctx, suiteName+"_list.zip")

	rule.Command().BuiltTool("soong_zip").
		FlagWithOutput("-o ", testListZipOutputFile).
		FlagWithArg("-C ", pathForPackaging(ctx).String()).
		FlagWithInput("-f ", testList).
		Flag("-sha256")

	rule.Build(strings.ReplaceAll(suiteName, "-", "_")+"_zip", suiteName+".zip")

	return outputFile, testListZipOutputFile
}

func buildTestList(ctx SingletonContext, listFile string, installedPaths InstallPaths) Path {
	buf := &strings.Builder{}
	for _, p := range installedPaths {
		if p.Ext() != ".config" {
			continue
		}
		pc, err := toTestListPath(p.String(), pathForTestCases(ctx).String(), "host/testcases")
		if err != nil {
			ctx.Errorf("Failed to convert path: %s, %v", p.String(), err)
			continue
		}
		buf.WriteString(pc)
		buf.WriteString("\n")
	}
	outputFile := pathForPackaging(ctx, listFile)
	WriteFileRuleVerbatim(ctx, outputFile, buf.String())
	return outputFile
}

func toTestListPath(path, relativeRoot, prefix string) (string, error) {
	dest, err := filepath.Rel(relativeRoot, path)
	if err != nil {
		return "", err
	}
	return filepath.Join(prefix, dest), nil
}

func pathForPackaging(ctx PathContext, pathComponents ...string) OutputPath {
	pathComponents = append([]string{"packaging"}, pathComponents...)
	return PathForOutput(ctx, pathComponents...)
}

func pathForTestCases(ctx PathContext) InstallPath {
	return pathForInstall(ctx, ctx.Config().BuildOS, X86, "testcases")
}
