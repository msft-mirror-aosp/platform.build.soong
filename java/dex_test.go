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

package java

import (
	"testing"

	"android/soong/android"

	"github.com/google/blueprint/proptools"
)

func TestR8(t *testing.T) {
	result := PrepareForTestWithJavaDefaultModulesWithoutFakeDex2oatd.RunTestWithBp(t, `
		android_app {
			name: "app",
			srcs: ["foo.java"],
			libs: ["lib"],
			static_libs: ["static_lib"],
			platform_apis: true,
		}

		android_app {
			name: "stable_app",
			srcs: ["foo.java"],
			sdk_version: "current",
			min_sdk_version: "31",
		}

		android_app {
			name: "core_platform_app",
			srcs: ["foo.java"],
			sdk_version: "core_platform",
			min_sdk_version: "31",
		}

		java_library {
			name: "lib",
			srcs: ["foo.java"],
		}

		java_library {
			name: "static_lib",
			srcs: ["foo.java"],
		}
	`)

	app := result.ModuleForTests("app", "android_common")
	stableApp := result.ModuleForTests("stable_app", "android_common")
	corePlatformApp := result.ModuleForTests("core_platform_app", "android_common")
	lib := result.ModuleForTests("lib", "android_common")
	staticLib := result.ModuleForTests("static_lib", "android_common")

	appJavac := app.Rule("javac")
	appR8 := app.Rule("r8")
	stableAppR8 := stableApp.Rule("r8")
	corePlatformAppR8 := corePlatformApp.Rule("r8")
	libHeader := lib.Output("turbine-combined/lib.jar").Output
	staticLibHeader := staticLib.Output("turbine-combined/static_lib.jar").Output

	android.AssertStringDoesContain(t, "expected lib header jar in app javac classpath",
		appJavac.Args["classpath"], libHeader.String())
	android.AssertStringDoesContain(t, "expected static_lib header jar in app javac classpath",
		appJavac.Args["classpath"], staticLibHeader.String())

	android.AssertStringDoesContain(t, "expected lib header jar in app r8 classpath",
		appR8.Args["r8Flags"], libHeader.String())
	android.AssertStringDoesNotContain(t, "expected no static_lib header jar in app r8 classpath",
		appR8.Args["r8Flags"], staticLibHeader.String())
	android.AssertStringDoesContain(t, "expected -ignorewarnings in app r8 flags",
		appR8.Args["r8Flags"], "-ignorewarnings")
	android.AssertStringDoesContain(t, "expected --android-platform-build in app r8 flags",
		appR8.Args["r8Flags"], "--android-platform-build")
	android.AssertStringDoesNotContain(t, "expected no --android-platform-build in stable_app r8 flags",
		stableAppR8.Args["r8Flags"], "--android-platform-build")
	android.AssertStringDoesContain(t, "expected --android-platform-build in core_platform_app r8 flags",
		corePlatformAppR8.Args["r8Flags"], "--android-platform-build")
}

func TestR8TransitiveDeps(t *testing.T) {
	bp := `
		override_android_app {
			name: "override_app",
			base: "app",
		}

		android_app {
			name: "app",
			srcs: ["foo.java"],
			libs: [
				"lib",
				"uses_libs_dep_import",
			],
			static_libs: [
				"static_lib",
				"repeated_dep",
			],
			platform_apis: true,
		}

		java_library {
			name: "static_lib",
			srcs: ["foo.java"],
		}

		java_library {
			name: "lib",
			libs: [
				"transitive_lib",
				"repeated_dep",
				"prebuilt_lib",
			],
			static_libs: ["transitive_static_lib"],
			srcs: ["foo.java"],
		}

		java_library {
			name: "repeated_dep",
			srcs: ["foo.java"],
		}

		java_library {
			name: "transitive_static_lib",
			srcs: ["foo.java"],
		}

		java_library {
			name: "transitive_lib",
			srcs: ["foo.java"],
			libs: ["transitive_lib_2"],
		}

		java_library {
			name: "transitive_lib_2",
			srcs: ["foo.java"],
		}

		java_import {
			name: "lib",
			jars: ["lib.jar"],
		}

		java_library {
			name: "uses_lib",
			srcs: ["foo.java"],
		}

		java_library {
			name: "optional_uses_lib",
			srcs: ["foo.java"],
		}

		android_library {
			name: "uses_libs_dep",
			uses_libs: ["uses_lib"],
			optional_uses_libs: ["optional_uses_lib"],
		}

		android_library_import {
			name: "uses_libs_dep_import",
			aars: ["aar.aar"],
			static_libs: ["uses_libs_dep"],
		}
	`

	testcases := []struct {
		name      string
		unbundled bool
	}{
		{
			name:      "non-unbundled build",
			unbundled: false,
		},
		{
			name:      "unbundled build",
			unbundled: true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			fixturePreparer := PrepareForTestWithJavaDefaultModulesWithoutFakeDex2oatd
			if tc.unbundled {
				fixturePreparer = android.GroupFixturePreparers(
					fixturePreparer,
					android.FixtureModifyProductVariables(
						func(variables android.FixtureProductVariables) {
							variables.Unbundled_build = proptools.BoolPtr(true)
						},
					),
				)
			}
			result := fixturePreparer.RunTestWithBp(t, bp)

			getHeaderJar := func(name string) android.Path {
				mod := result.ModuleForTests(name, "android_common")
				return mod.Output("turbine-combined/" + name + ".jar").Output
			}

			appR8 := result.ModuleForTests("app", "android_common").Rule("r8")
			overrideAppR8 := result.ModuleForTests("app", "android_common_override_app").Rule("r8")
			appHeader := getHeaderJar("app")
			overrideAppHeader := result.ModuleForTests("app", "android_common_override_app").Output("turbine-combined/app.jar").Output
			libHeader := getHeaderJar("lib")
			transitiveLibHeader := getHeaderJar("transitive_lib")
			transitiveLib2Header := getHeaderJar("transitive_lib_2")
			staticLibHeader := getHeaderJar("static_lib")
			transitiveStaticLibHeader := getHeaderJar("transitive_static_lib")
			repeatedDepHeader := getHeaderJar("repeated_dep")
			usesLibHeader := getHeaderJar("uses_lib")
			optionalUsesLibHeader := getHeaderJar("optional_uses_lib")
			prebuiltLibHeader := result.ModuleForTests("prebuilt_lib", "android_common").Output("combined/lib.jar").Output

			for _, rule := range []android.TestingBuildParams{appR8, overrideAppR8} {
				android.AssertStringDoesNotContain(t, "expected no app header jar in app r8 classpath",
					rule.Args["r8Flags"], appHeader.String())
				android.AssertStringDoesNotContain(t, "expected no override_app header jar in app r8 classpath",
					rule.Args["r8Flags"], overrideAppHeader.String())
				android.AssertStringDoesContain(t, "expected transitive lib header jar in app r8 classpath",
					rule.Args["r8Flags"], transitiveLibHeader.String())
				android.AssertStringDoesContain(t, "expected transitive lib ^2 header jar in app r8 classpath",
					rule.Args["r8Flags"], transitiveLib2Header.String())
				android.AssertStringDoesContain(t, "expected lib header jar in app r8 classpath",
					rule.Args["r8Flags"], libHeader.String())
				android.AssertStringDoesContain(t, "expected uses_lib header jar in app r8 classpath",
					rule.Args["r8Flags"], usesLibHeader.String())
				android.AssertStringDoesContain(t, "expected optional_uses_lib header jar in app r8 classpath",
					rule.Args["r8Flags"], optionalUsesLibHeader.String())
				android.AssertStringDoesNotContain(t, "expected no static_lib header jar in app r8 classpath",
					rule.Args["r8Flags"], staticLibHeader.String())
				android.AssertStringDoesNotContain(t, "expected no transitive static_lib header jar in app r8 classpath",
					rule.Args["r8Flags"], transitiveStaticLibHeader.String())
				// we shouldn't list this dep because it is already included as static_libs in the app
				android.AssertStringDoesNotContain(t, "expected no repeated_dep header jar in app r8 classpath",
					rule.Args["r8Flags"], repeatedDepHeader.String())
				// skip a prebuilt transitive dep if the source is also a transitive dep
				android.AssertStringDoesNotContain(t, "expected no prebuilt header jar in app r8 classpath",
					rule.Args["r8Flags"], prebuiltLibHeader.String())
				android.AssertStringDoesContain(t, "expected -ignorewarnings in app r8 flags",
					rule.Args["r8Flags"], "-ignorewarnings")
				android.AssertStringDoesContain(t, "expected --android-platform-build in app r8 flags",
					rule.Args["r8Flags"], "--android-platform-build")
			}
		})
	}
}

func TestR8Flags(t *testing.T) {
	result := PrepareForTestWithJavaDefaultModulesWithoutFakeDex2oatd.RunTestWithBp(t, `
		android_app {
			name: "app",
			srcs: ["foo.java"],
			platform_apis: true,
			optimize: {
				shrink: false,
				optimize: false,
				obfuscate: false,
				ignore_warnings: false,
			},
		}
	`)

	app := result.ModuleForTests("app", "android_common")
	appR8 := app.Rule("r8")
	android.AssertStringDoesContain(t, "expected -dontshrink in app r8 flags",
		appR8.Args["r8Flags"], "-dontshrink")
	android.AssertStringDoesContain(t, "expected -dontoptimize in app r8 flags",
		appR8.Args["r8Flags"], "-dontoptimize")
	android.AssertStringDoesContain(t, "expected -dontobfuscate in app r8 flags",
		appR8.Args["r8Flags"], "-dontobfuscate")
	android.AssertStringDoesNotContain(t, "expected no -ignorewarnings in app r8 flags",
		appR8.Args["r8Flags"], "-ignorewarnings")
	android.AssertStringDoesContain(t, "expected --android-platform-build in app r8 flags",
		appR8.Args["r8Flags"], "--android-platform-build")
}

func TestD8(t *testing.T) {
	result := PrepareForTestWithJavaDefaultModulesWithoutFakeDex2oatd.RunTestWithBp(t, `
		java_library {
			name: "foo",
			srcs: ["foo.java"],
			libs: ["lib"],
			static_libs: ["static_lib"],
			installable: true,
		}

		java_library {
			name: "lib",
			srcs: ["foo.java"],
		}

		java_library {
			name: "static_lib",
			srcs: ["foo.java"],
		}
	`)

	foo := result.ModuleForTests("foo", "android_common")
	lib := result.ModuleForTests("lib", "android_common")
	staticLib := result.ModuleForTests("static_lib", "android_common")

	fooJavac := foo.Rule("javac")
	fooD8 := foo.Rule("d8")
	libHeader := lib.Output("turbine-combined/lib.jar").Output
	staticLibHeader := staticLib.Output("turbine-combined/static_lib.jar").Output

	android.AssertStringDoesContain(t, "expected lib header jar in foo javac classpath",
		fooJavac.Args["classpath"], libHeader.String())
	android.AssertStringDoesContain(t, "expected static_lib header jar in foo javac classpath",
		fooJavac.Args["classpath"], staticLibHeader.String())

	android.AssertStringDoesContain(t, "expected lib header jar in foo d8 classpath",
		fooD8.Args["d8Flags"], libHeader.String())
	android.AssertStringDoesNotContain(t, "expected no  static_lib header jar in foo javac classpath",
		fooD8.Args["d8Flags"], staticLibHeader.String())
}

func TestProguardFlagsInheritance(t *testing.T) {
	result := PrepareForTestWithJavaDefaultModulesWithoutFakeDex2oatd.RunTestWithBp(t, `
		android_app {
			name: "app",
			static_libs: [
				"primary_android_lib",
				"primary_lib",
			],
			platform_apis: true,
		}

		java_library {
			name: "primary_lib",
			optimize: {
				proguard_flags_files: ["primary.flags"],
			},
		}

		android_library {
			name: "primary_android_lib",
			static_libs: ["secondary_lib"],
			optimize: {
				proguard_flags_files: ["primary_android.flags"],
			},
		}

		java_library {
			name: "secondary_lib",
			static_libs: ["tertiary_lib"],
			optimize: {
				proguard_flags_files: ["secondary.flags"],
			},
		}

		java_library {
			name: "tertiary_lib",
			optimize: {
				proguard_flags_files: ["tertiary.flags"],
			},
		}
	`)

	app := result.ModuleForTests("app", "android_common")
	appR8 := app.Rule("r8")
	android.AssertStringDoesContain(t, "expected primary_lib's proguard flags from direct dep",
		appR8.Args["r8Flags"], "primary.flags")
	android.AssertStringDoesContain(t, "expected primary_android_lib's proguard flags from direct dep",
		appR8.Args["r8Flags"], "primary_android.flags")
	android.AssertStringDoesContain(t, "expected secondary_lib's proguard flags from inherited dep",
		appR8.Args["r8Flags"], "secondary.flags")
	android.AssertStringDoesContain(t, "expected tertiary_lib's proguard flags from inherited dep",
		appR8.Args["r8Flags"], "tertiary.flags")
}
