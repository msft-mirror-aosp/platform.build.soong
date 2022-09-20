// Copyright 2021 Google Inc. All rights reserved.
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

package bp2build

import (
	"testing"

	"android/soong/android"
	"android/soong/cc"
)

const (
	// See cc/testing.go for more context
	soongCcLibraryHeadersPreamble = `
cc_defaults {
    name: "linux_bionic_supported",
}`
)

func TestCcLibraryHeadersLoadStatement(t *testing.T) {
	testCases := []struct {
		bazelTargets           BazelTargets
		expectedLoadStatements string
	}{
		{
			bazelTargets: BazelTargets{
				BazelTarget{
					name:      "cc_library_headers_target",
					ruleClass: "cc_library_headers",
					// Note: no bzlLoadLocation for native rules
				},
			},
			expectedLoadStatements: ``,
		},
	}

	for _, testCase := range testCases {
		actual := testCase.bazelTargets.LoadStatements()
		expected := testCase.expectedLoadStatements
		if actual != expected {
			t.Fatalf("Expected load statements to be %s, got %s", expected, actual)
		}
	}
}

func registerCcLibraryHeadersModuleTypes(ctx android.RegistrationContext) {
	cc.RegisterCCBuildComponents(ctx)
}

func runCcLibraryHeadersTestCase(t *testing.T, tc Bp2buildTestCase) {
	t.Helper()
	RunBp2BuildTestCase(t, registerCcLibraryHeadersModuleTypes, tc)
}

func TestCcLibraryHeadersSimple(t *testing.T) {
	runCcLibraryHeadersTestCase(t, Bp2buildTestCase{
		Description:                "cc_library_headers test",
		ModuleTypeUnderTest:        "cc_library_headers",
		ModuleTypeUnderTestFactory: cc.LibraryHeaderFactory,
		Filesystem: map[string]string{
			"lib-1/lib1a.h":                        "",
			"lib-1/lib1b.h":                        "",
			"lib-2/lib2a.h":                        "",
			"lib-2/lib2b.h":                        "",
			"dir-1/dir1a.h":                        "",
			"dir-1/dir1b.h":                        "",
			"dir-2/dir2a.h":                        "",
			"dir-2/dir2b.h":                        "",
			"arch_arm64_exported_include_dir/a.h":  "",
			"arch_x86_exported_include_dir/b.h":    "",
			"arch_x86_64_exported_include_dir/c.h": "",
		},
		Blueprint: soongCcLibraryHeadersPreamble + `
cc_library_headers {
    name: "foo_headers",
    export_include_dirs: ["dir-1", "dir-2"],
    header_libs: ["lib-1", "lib-2"],

    arch: {
        arm64: {
      // We expect dir-1 headers to be dropped, because dir-1 is already in export_include_dirs
            export_include_dirs: ["arch_arm64_exported_include_dir", "dir-1"],
        },
        x86: {
            export_include_dirs: ["arch_x86_exported_include_dir"],
        },
        x86_64: {
            export_include_dirs: ["arch_x86_64_exported_include_dir"],
        },
    },
    sdk_version: "current",
    min_sdk_version: "29",

    // TODO: Also support export_header_lib_headers
}`,
		ExpectedBazelTargets: []string{
			MakeBazelTarget("cc_library_headers", "foo_headers", AttrNameToString{
				"export_includes": `[
        "dir-1",
        "dir-2",
    ] + select({
        "//build/bazel/platforms/arch:arm64": ["arch_arm64_exported_include_dir"],
        "//build/bazel/platforms/arch:x86": ["arch_x86_exported_include_dir"],
        "//build/bazel/platforms/arch:x86_64": ["arch_x86_64_exported_include_dir"],
        "//conditions:default": [],
    })`,
				"sdk_version":     `"current"`,
				"min_sdk_version": `"29"`,
			}),
		},
	})
}

func TestCcLibraryHeadersOsSpecificHeader(t *testing.T) {
	runCcLibraryHeadersTestCase(t, Bp2buildTestCase{
		Description:                "cc_library_headers test with os-specific header_libs props",
		ModuleTypeUnderTest:        "cc_library_headers",
		ModuleTypeUnderTestFactory: cc.LibraryHeaderFactory,
		Filesystem:                 map[string]string{},
		Blueprint: soongCcLibraryPreamble + `
cc_library_headers {
    name: "android-lib",
    bazel_module: { bp2build_available: false },
}
cc_library_headers {
    name: "base-lib",
    bazel_module: { bp2build_available: false },
}
cc_library_headers {
    name: "darwin-lib",
    bazel_module: { bp2build_available: false },
}
cc_library_headers {
    name: "linux-lib",
    bazel_module: { bp2build_available: false },
}
cc_library_headers {
    name: "linux_bionic-lib",
    bazel_module: { bp2build_available: false },
}
cc_library_headers {
    name: "windows-lib",
    bazel_module: { bp2build_available: false },
}
cc_library_headers {
    name: "foo_headers",
    header_libs: ["base-lib"],
		export_header_lib_headers: ["base-lib"],
    target: {
        android: {
						header_libs: ["android-lib"],
						export_header_lib_headers: ["android-lib"],
				},
        darwin: {
						header_libs: ["darwin-lib"],
						export_header_lib_headers: ["darwin-lib"],
				},
        linux_bionic: {
						header_libs: ["linux_bionic-lib"],
						export_header_lib_headers: ["linux_bionic-lib"],
				},
        linux_glibc: {
						header_libs: ["linux-lib"],
						export_header_lib_headers: ["linux-lib"],
				},
        windows: {
						header_libs: ["windows-lib"],
						export_header_lib_headers: ["windows-lib"],
				},
    },
    include_build_directory: false,
}`,
		ExpectedBazelTargets: []string{
			MakeBazelTarget("cc_library_headers", "foo_headers", AttrNameToString{
				"deps": `[":base-lib"] + select({
        "//build/bazel/platforms/os:android": [":android-lib"],
        "//build/bazel/platforms/os:darwin": [":darwin-lib"],
        "//build/bazel/platforms/os:linux": [":linux-lib"],
        "//build/bazel/platforms/os:linux_bionic": [":linux_bionic-lib"],
        "//build/bazel/platforms/os:windows": [":windows-lib"],
        "//conditions:default": [],
    })`,
			}),
		},
	})
}

func TestCcLibraryHeadersOsSpecficHeaderLibsExportHeaderLibHeaders(t *testing.T) {
	runCcLibraryHeadersTestCase(t, Bp2buildTestCase{
		Description:                "cc_library_headers test with os-specific header_libs and export_header_lib_headers props",
		ModuleTypeUnderTest:        "cc_library_headers",
		ModuleTypeUnderTestFactory: cc.LibraryHeaderFactory,
		Filesystem:                 map[string]string{},
		Blueprint: soongCcLibraryPreamble + `
cc_library_headers {
    name: "android-lib",
    bazel_module: { bp2build_available: false },
  }
cc_library_headers {
    name: "exported-lib",
    bazel_module: { bp2build_available: false },
}
cc_library_headers {
    name: "foo_headers",
    target: {
        android: {
            header_libs: ["android-lib", "exported-lib"],
            export_header_lib_headers: ["exported-lib"]
        },
    },
    include_build_directory: false,
}`,
		ExpectedBazelTargets: []string{
			MakeBazelTarget("cc_library_headers", "foo_headers", AttrNameToString{
				"deps": `select({
        "//build/bazel/platforms/os:android": [":exported-lib"],
        "//conditions:default": [],
    })`,
			}),
		},
	})
}

func TestCcLibraryHeadersArchAndTargetExportSystemIncludes(t *testing.T) {
	runCcLibraryHeadersTestCase(t, Bp2buildTestCase{
		Description:                "cc_library_headers test with arch-specific and target-specific export_system_include_dirs props",
		ModuleTypeUnderTest:        "cc_library_headers",
		ModuleTypeUnderTestFactory: cc.LibraryHeaderFactory,
		Filesystem:                 map[string]string{},
		Blueprint: soongCcLibraryPreamble + `cc_library_headers {
    name: "foo_headers",
    export_system_include_dirs: [
        "shared_include_dir",
    ],
    target: {
        android: {
            export_system_include_dirs: [
                "android_include_dir",
            ],
        },
        linux_glibc: {
            export_system_include_dirs: [
                "linux_include_dir",
            ],
        },
        darwin: {
            export_system_include_dirs: [
                "darwin_include_dir",
            ],
        },
    },
    arch: {
        arm: {
            export_system_include_dirs: [
                "arm_include_dir",
            ],
        },
        x86_64: {
            export_system_include_dirs: [
                "x86_64_include_dir",
            ],
        },
    },
    include_build_directory: false,
}`,
		ExpectedBazelTargets: []string{
			MakeBazelTarget("cc_library_headers", "foo_headers", AttrNameToString{
				"export_system_includes": `["shared_include_dir"] + select({
        "//build/bazel/platforms/arch:arm": ["arm_include_dir"],
        "//build/bazel/platforms/arch:x86_64": ["x86_64_include_dir"],
        "//conditions:default": [],
    }) + select({
        "//build/bazel/platforms/os:android": ["android_include_dir"],
        "//build/bazel/platforms/os:darwin": ["darwin_include_dir"],
        "//build/bazel/platforms/os:linux": ["linux_include_dir"],
        "//conditions:default": [],
    })`,
			}),
		},
	})
}

func TestCcLibraryHeadersNoCrtIgnored(t *testing.T) {
	runCcLibraryHeadersTestCase(t, Bp2buildTestCase{
		Description:                "cc_library_headers test",
		ModuleTypeUnderTest:        "cc_library_headers",
		ModuleTypeUnderTestFactory: cc.LibraryHeaderFactory,
		Filesystem: map[string]string{
			"lib-1/lib1a.h":                        "",
			"lib-1/lib1b.h":                        "",
			"lib-2/lib2a.h":                        "",
			"lib-2/lib2b.h":                        "",
			"dir-1/dir1a.h":                        "",
			"dir-1/dir1b.h":                        "",
			"dir-2/dir2a.h":                        "",
			"dir-2/dir2b.h":                        "",
			"arch_arm64_exported_include_dir/a.h":  "",
			"arch_x86_exported_include_dir/b.h":    "",
			"arch_x86_64_exported_include_dir/c.h": "",
		},
		Blueprint: soongCcLibraryHeadersPreamble + `
cc_library_headers {
    name: "lib-1",
    export_include_dirs: ["lib-1"],
    no_libcrt: true,
    include_build_directory: false,
}`,
		ExpectedBazelTargets: []string{
			MakeBazelTarget("cc_library_headers", "lib-1", AttrNameToString{
				"export_includes": `["lib-1"]`,
			}),
		},
	})
}

func TestCcLibraryHeadersExportedStaticLibHeadersReexported(t *testing.T) {
	runCcLibraryHeadersTestCase(t, Bp2buildTestCase{
		Description:                "cc_library_headers exported_static_lib_headers is reexported",
		ModuleTypeUnderTest:        "cc_library_headers",
		ModuleTypeUnderTestFactory: cc.LibraryHeaderFactory,
		Filesystem:                 map[string]string{},
		Blueprint: soongCcLibraryHeadersPreamble + `
cc_library_headers {
		name: "foo_headers",
		export_static_lib_headers: ["foo_export"],
		static_libs: ["foo_export", "foo_no_reexport"],
    bazel_module: { bp2build_available: true },
}
` + simpleModuleDoNotConvertBp2build("cc_library_headers", "foo_export"),
		ExpectedBazelTargets: []string{
			MakeBazelTarget("cc_library_headers", "foo_headers", AttrNameToString{
				"deps": `[":foo_export"]`,
			}),
		},
	})
}

func TestCcLibraryHeadersExportedSharedLibHeadersReexported(t *testing.T) {
	runCcLibraryHeadersTestCase(t, Bp2buildTestCase{
		Description:                "cc_library_headers exported_shared_lib_headers is reexported",
		ModuleTypeUnderTest:        "cc_library_headers",
		ModuleTypeUnderTestFactory: cc.LibraryHeaderFactory,
		Filesystem:                 map[string]string{},
		Blueprint: soongCcLibraryHeadersPreamble + `
cc_library_headers {
		name: "foo_headers",
		export_shared_lib_headers: ["foo_export"],
		shared_libs: ["foo_export", "foo_no_reexport"],
    bazel_module: { bp2build_available: true },
}
` + simpleModuleDoNotConvertBp2build("cc_library_headers", "foo_export"),
		ExpectedBazelTargets: []string{
			MakeBazelTarget("cc_library_headers", "foo_headers", AttrNameToString{
				"deps": `[":foo_export"]`,
			}),
		},
	})
}

func TestCcLibraryHeadersExportedHeaderLibHeadersReexported(t *testing.T) {
	runCcLibraryHeadersTestCase(t, Bp2buildTestCase{
		Description:                "cc_library_headers exported_header_lib_headers is reexported",
		ModuleTypeUnderTest:        "cc_library_headers",
		ModuleTypeUnderTestFactory: cc.LibraryHeaderFactory,
		Filesystem:                 map[string]string{},
		Blueprint: soongCcLibraryHeadersPreamble + `
cc_library_headers {
		name: "foo_headers",
		export_header_lib_headers: ["foo_export"],
		header_libs: ["foo_export", "foo_no_reexport"],
    bazel_module: { bp2build_available: true },
}
` + simpleModuleDoNotConvertBp2build("cc_library_headers", "foo_export"),
		ExpectedBazelTargets: []string{
			MakeBazelTarget("cc_library_headers", "foo_headers", AttrNameToString{
				"deps": `[":foo_export"]`,
			}),
		},
	})
}

func TestCcLibraryHeadersWholeStaticLibsReexported(t *testing.T) {
	runCcLibraryHeadersTestCase(t, Bp2buildTestCase{
		Description:                "cc_library_headers whole_static_libs is reexported",
		ModuleTypeUnderTest:        "cc_library_headers",
		ModuleTypeUnderTestFactory: cc.LibraryHeaderFactory,
		Filesystem:                 map[string]string{},
		Blueprint: soongCcLibraryHeadersPreamble + `
cc_library_headers {
		name: "foo_headers",
		whole_static_libs: ["foo_export"],
    bazel_module: { bp2build_available: true },
}
` + simpleModuleDoNotConvertBp2build("cc_library_headers", "foo_export"),
		ExpectedBazelTargets: []string{
			MakeBazelTarget("cc_library_headers", "foo_headers", AttrNameToString{
				"deps": `[":foo_export"]`,
			}),
		},
	})
}
