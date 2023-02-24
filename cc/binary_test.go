// Copyright 2022 Google Inc. All rights reserved.
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

package cc

import (
	"android/soong/bazel/cquery"
	"testing"

	"android/soong/android"
)

func TestCcBinaryWithBazel(t *testing.T) {
	t.Parallel()
	bp := `
cc_binary {
	name: "foo",
	srcs: ["foo.cc"],
	bazel_module: { label: "//foo/bar:bar" },
}`
	config := TestConfig(t.TempDir(), android.Android, nil, bp, nil)
	config.BazelContext = android.MockBazelContext{
		OutputBaseDir: "outputbase",
		LabelToCcBinary: map[string]cquery.CcUnstrippedInfo{
			"//foo/bar:bar": cquery.CcUnstrippedInfo{
				OutputFile:       "foo",
				UnstrippedOutput: "foo.unstripped",
			},
		},
	}
	ctx := testCcWithConfig(t, config)

	binMod := ctx.ModuleForTests("foo", "android_arm64_armv8-a").Module()
	producer := binMod.(android.OutputFileProducer)
	outputFiles, err := producer.OutputFiles("")
	if err != nil {
		t.Errorf("Unexpected error getting cc_binary outputfiles %s", err)
	}
	expectedOutputFiles := []string{"outputbase/execroot/__main__/foo"}
	android.AssertDeepEquals(t, "output files", expectedOutputFiles, outputFiles.Strings())

	unStrippedFilePath := binMod.(*Module).UnstrippedOutputFile()
	expectedUnStrippedFile := "outputbase/execroot/__main__/foo.unstripped"
	android.AssertStringEquals(t, "Unstripped output file", expectedUnStrippedFile, unStrippedFilePath.String())
}

func TestCcBinaryWithBazelValidations(t *testing.T) {
	t.Parallel()
	bp := `
cc_binary {
	name: "foo",
	srcs: ["foo.cc"],
	bazel_module: { label: "//foo/bar:bar" },
	tidy: true,
}`
	config := TestConfig(t.TempDir(), android.Android, nil, bp, nil)
	config.BazelContext = android.MockBazelContext{
		OutputBaseDir: "outputbase",
		LabelToCcBinary: map[string]cquery.CcUnstrippedInfo{
			"//foo/bar:bar": cquery.CcUnstrippedInfo{
				OutputFile:       "foo",
				UnstrippedOutput: "foo.unstripped",
				TidyFiles:        []string{"foo.c.tidy"},
			},
		},
	}
	ctx := android.GroupFixturePreparers(
		prepareForCcTest,
		android.FixtureMergeEnv(map[string]string{
			"ALLOW_LOCAL_TIDY_TRUE": "1",
		}),
	).RunTestWithConfig(t, config).TestContext

	binMod := ctx.ModuleForTests("foo", "android_arm64_armv8-a").Module()
	producer := binMod.(android.OutputFileProducer)
	outputFiles, err := producer.OutputFiles("")
	if err != nil {
		t.Errorf("Unexpected error getting cc_binary outputfiles %s", err)
	}
	expectedOutputFiles := []string{"out/soong/.intermediates/foo/android_arm64_armv8-a/validated/foo"}
	android.AssertPathsRelativeToTopEquals(t, "output files", expectedOutputFiles, outputFiles)

	unStrippedFilePath := binMod.(*Module).UnstrippedOutputFile()
	expectedUnStrippedFile := "outputbase/execroot/__main__/foo.unstripped"
	android.AssertStringEquals(t, "Unstripped output file", expectedUnStrippedFile, unStrippedFilePath.String())
}

func TestBinaryLinkerScripts(t *testing.T) {
	t.Parallel()
	result := PrepareForIntegrationTestWithCc.RunTestWithBp(t, `
		cc_binary {
			name: "foo",
			srcs: ["foo.cc"],
			linker_scripts: ["foo.ld", "bar.ld"],
		}`)

	binFoo := result.ModuleForTests("foo", "android_arm64_armv8-a").Rule("ld")

	android.AssertStringListContains(t, "missing dependency on linker_scripts",
		binFoo.Implicits.Strings(), "foo.ld")
	android.AssertStringListContains(t, "missing dependency on linker_scripts",
		binFoo.Implicits.Strings(), "bar.ld")
	android.AssertStringDoesContain(t, "missing flag for linker_scripts",
		binFoo.Args["ldFlags"], "-Wl,--script,foo.ld")
	android.AssertStringDoesContain(t, "missing flag for linker_scripts",
		binFoo.Args["ldFlags"], "-Wl,--script,bar.ld")
}
