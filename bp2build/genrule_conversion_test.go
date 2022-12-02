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
	"fmt"
	"testing"

	"android/soong/android"
	"android/soong/cc"
	"android/soong/genrule"
	"android/soong/java"
)

func registerGenruleModuleTypes(ctx android.RegistrationContext) {
	ctx.RegisterModuleType("genrule_defaults", func() android.Module { return genrule.DefaultsFactory() })
}

func runGenruleTestCase(t *testing.T, tc Bp2buildTestCase) {
	t.Helper()
	(&tc).ModuleTypeUnderTest = "genrule"
	(&tc).ModuleTypeUnderTestFactory = genrule.GenRuleFactory
	RunBp2BuildTestCase(t, registerGenruleModuleTypes, tc)
}

func otherGenruleBp(genruleTarget string) map[string]string {
	return map[string]string{
		"other/Android.bp": fmt.Sprintf(`%s {
    name: "foo.tool",
    out: ["foo_tool.out"],
    srcs: ["foo_tool.in"],
    cmd: "cp $(in) $(out)",
}
%s {
    name: "other.tool",
    out: ["other_tool.out"],
    srcs: ["other_tool.in"],
    cmd: "cp $(in) $(out)",
}`, genruleTarget, genruleTarget),
	}
}

func TestGenruleCliVariableReplacement(t *testing.T) {
	testCases := []struct {
		moduleType string
		factory    android.ModuleFactory
		genDir     string
		hod        android.HostOrDeviceSupported
	}{
		{
			moduleType: "genrule",
			factory:    genrule.GenRuleFactory,
			genDir:     "$(RULEDIR)",
		},
		{
			moduleType: "cc_genrule",
			factory:    cc.GenRuleFactory,
			genDir:     "$(RULEDIR)",
			hod:        android.DeviceSupported,
		},
		{
			moduleType: "java_genrule",
			factory:    java.GenRuleFactory,
			genDir:     "$(RULEDIR)",
			hod:        android.DeviceSupported,
		},
		{
			moduleType: "java_genrule_host",
			factory:    java.GenRuleFactoryHost,
			genDir:     "$(RULEDIR)",
			hod:        android.HostSupported,
		},
	}

	bp := `%s {
    name: "foo.tool",
    out: ["foo_tool.out"],
    srcs: ["foo_tool.in"],
    cmd: "cp $(in) $(out)",
    bazel_module: { bp2build_available: false },
}

%s {
    name: "foo",
    out: ["foo.out"],
    srcs: ["foo.in"],
    tools: [":foo.tool"],
    cmd: "$(location :foo.tool) --genDir=$(genDir) arg $(in) $(out)",
    bazel_module: { bp2build_available: true },
}`

	for _, tc := range testCases {
		moduleAttrs := AttrNameToString{
			"cmd":   fmt.Sprintf(`"$(location :foo.tool) --genDir=%s arg $(SRCS) $(OUTS)"`, tc.genDir),
			"outs":  `["foo.out"]`,
			"srcs":  `["foo.in"]`,
			"tools": `[":foo.tool"]`,
		}

		expectedBazelTargets := []string{
			makeBazelTargetHostOrDevice("genrule", "foo", moduleAttrs, tc.hod),
		}

		t.Run(tc.moduleType, func(t *testing.T) {
			RunBp2BuildTestCase(t, func(ctx android.RegistrationContext) {},
				Bp2buildTestCase{
					ModuleTypeUnderTest:        tc.moduleType,
					ModuleTypeUnderTestFactory: tc.factory,
					Blueprint:                  fmt.Sprintf(bp, tc.moduleType, tc.moduleType),
					ExpectedBazelTargets:       expectedBazelTargets,
				})
		})
	}
}

func TestGenruleLocationsLabel(t *testing.T) {
	testCases := []struct {
		moduleType string
		factory    android.ModuleFactory
		hod        android.HostOrDeviceSupported
	}{
		{
			moduleType: "genrule",
			factory:    genrule.GenRuleFactory,
		},
		{
			moduleType: "cc_genrule",
			factory:    cc.GenRuleFactory,
			hod:        android.DeviceSupported,
		},
		{
			moduleType: "java_genrule",
			factory:    java.GenRuleFactory,
			hod:        android.DeviceSupported,
		},
		{
			moduleType: "java_genrule_host",
			factory:    java.GenRuleFactoryHost,
			hod:        android.HostSupported,
		},
	}

	bp := `%s {
    name: "foo.tools",
    out: ["foo_tool.out", "foo_tool2.out"],
    srcs: ["foo_tool.in"],
    cmd: "cp $(in) $(out)",
    bazel_module: { bp2build_available: true },
}

%s {
    name: "foo",
    out: ["foo.out"],
    srcs: ["foo.in"],
    tools: [":foo.tools"],
    cmd: "$(locations :foo.tools) -s $(out) $(in)",
    bazel_module: { bp2build_available: true },
}`

	for _, tc := range testCases {
		fooAttrs := AttrNameToString{
			"cmd":   `"$(locations :foo.tools) -s $(OUTS) $(SRCS)"`,
			"outs":  `["foo.out"]`,
			"srcs":  `["foo.in"]`,
			"tools": `[":foo.tools"]`,
		}
		fooToolsAttrs := AttrNameToString{
			"cmd": `"cp $(SRCS) $(OUTS)"`,
			"outs": `[
        "foo_tool.out",
        "foo_tool2.out",
    ]`,
			"srcs": `["foo_tool.in"]`,
		}

		expectedBazelTargets := []string{
			makeBazelTargetHostOrDevice("genrule", "foo", fooAttrs, tc.hod),
			makeBazelTargetHostOrDevice("genrule", "foo.tools", fooToolsAttrs, tc.hod),
		}

		t.Run(tc.moduleType, func(t *testing.T) {
			RunBp2BuildTestCase(t, func(ctx android.RegistrationContext) {},
				Bp2buildTestCase{
					ModuleTypeUnderTest:        tc.moduleType,
					ModuleTypeUnderTestFactory: tc.factory,
					Blueprint:                  fmt.Sprintf(bp, tc.moduleType, tc.moduleType),
					ExpectedBazelTargets:       expectedBazelTargets,
				})
		})
	}
}

func TestGenruleLocationsAbsoluteLabel(t *testing.T) {
	testCases := []struct {
		moduleType string
		factory    android.ModuleFactory
		hod        android.HostOrDeviceSupported
	}{
		{
			moduleType: "genrule",
			factory:    genrule.GenRuleFactory,
		},
		{
			moduleType: "cc_genrule",
			factory:    cc.GenRuleFactory,
			hod:        android.DeviceSupported,
		},
		{
			moduleType: "java_genrule",
			factory:    java.GenRuleFactory,
			hod:        android.DeviceSupported,
		},
		{
			moduleType: "java_genrule_host",
			factory:    java.GenRuleFactoryHost,
			hod:        android.HostSupported,
		},
	}

	bp := `%s {
    name: "foo",
    out: ["foo.out"],
    srcs: ["foo.in"],
    tool_files: [":foo.tool"],
    cmd: "$(locations :foo.tool) -s $(out) $(in)",
    bazel_module: { bp2build_available: true },
}`

	for _, tc := range testCases {
		moduleAttrs := AttrNameToString{
			"cmd":   `"$(locations //other:foo.tool) -s $(OUTS) $(SRCS)"`,
			"outs":  `["foo.out"]`,
			"srcs":  `["foo.in"]`,
			"tools": `["//other:foo.tool"]`,
		}

		expectedBazelTargets := []string{
			makeBazelTargetHostOrDevice("genrule", "foo", moduleAttrs, tc.hod),
		}

		t.Run(tc.moduleType, func(t *testing.T) {
			RunBp2BuildTestCase(t, func(ctx android.RegistrationContext) {},
				Bp2buildTestCase{
					ModuleTypeUnderTest:        tc.moduleType,
					ModuleTypeUnderTestFactory: tc.factory,
					Blueprint:                  fmt.Sprintf(bp, tc.moduleType),
					ExpectedBazelTargets:       expectedBazelTargets,
					Filesystem:                 otherGenruleBp(tc.moduleType),
				})
		})
	}
}

func TestGenruleSrcsLocationsAbsoluteLabel(t *testing.T) {
	testCases := []struct {
		moduleType string
		factory    android.ModuleFactory
		hod        android.HostOrDeviceSupported
	}{
		{
			moduleType: "genrule",
			factory:    genrule.GenRuleFactory,
		},
		{
			moduleType: "cc_genrule",
			factory:    cc.GenRuleFactory,
			hod:        android.DeviceSupported,
		},
		{
			moduleType: "java_genrule",
			factory:    java.GenRuleFactory,
			hod:        android.DeviceSupported,
		},
		{
			moduleType: "java_genrule_host",
			factory:    java.GenRuleFactoryHost,
			hod:        android.HostSupported,
		},
	}

	bp := `%s {
    name: "foo",
    out: ["foo.out"],
    srcs: [":other.tool"],
    tool_files: [":foo.tool"],
    cmd: "$(locations :foo.tool) -s $(out) $(location :other.tool)",
    bazel_module: { bp2build_available: true },
}`

	for _, tc := range testCases {
		moduleAttrs := AttrNameToString{
			"cmd":   `"$(locations //other:foo.tool) -s $(OUTS) $(location //other:other.tool)"`,
			"outs":  `["foo.out"]`,
			"srcs":  `["//other:other.tool"]`,
			"tools": `["//other:foo.tool"]`,
		}

		expectedBazelTargets := []string{
			makeBazelTargetHostOrDevice("genrule", "foo", moduleAttrs, tc.hod),
		}

		t.Run(tc.moduleType, func(t *testing.T) {
			RunBp2BuildTestCase(t, func(ctx android.RegistrationContext) {},
				Bp2buildTestCase{
					ModuleTypeUnderTest:        tc.moduleType,
					ModuleTypeUnderTestFactory: tc.factory,
					Blueprint:                  fmt.Sprintf(bp, tc.moduleType),
					ExpectedBazelTargets:       expectedBazelTargets,
					Filesystem:                 otherGenruleBp(tc.moduleType),
				})
		})
	}
}

func TestGenruleLocationLabelShouldSubstituteFirstToolLabel(t *testing.T) {
	testCases := []struct {
		moduleType string
		factory    android.ModuleFactory
		hod        android.HostOrDeviceSupported
	}{
		{
			moduleType: "genrule",
			factory:    genrule.GenRuleFactory,
		},
		{
			moduleType: "cc_genrule",
			factory:    cc.GenRuleFactory,
			hod:        android.DeviceSupported,
		},
		{
			moduleType: "java_genrule",
			factory:    java.GenRuleFactory,
			hod:        android.DeviceSupported,
		},
		{
			moduleType: "java_genrule_host",
			factory:    java.GenRuleFactoryHost,
			hod:        android.HostSupported,
		},
	}

	bp := `%s {
    name: "foo",
    out: ["foo.out"],
    srcs: ["foo.in"],
    tool_files: [":foo.tool", ":other.tool"],
    cmd: "$(location) -s $(out) $(in)",
    bazel_module: { bp2build_available: true },
}`

	for _, tc := range testCases {
		moduleAttrs := AttrNameToString{
			"cmd":  `"$(location //other:foo.tool) -s $(OUTS) $(SRCS)"`,
			"outs": `["foo.out"]`,
			"srcs": `["foo.in"]`,
			"tools": `[
        "//other:foo.tool",
        "//other:other.tool",
    ]`,
		}

		expectedBazelTargets := []string{
			makeBazelTargetHostOrDevice("genrule", "foo", moduleAttrs, tc.hod),
		}

		t.Run(tc.moduleType, func(t *testing.T) {
			RunBp2BuildTestCase(t, func(ctx android.RegistrationContext) {},
				Bp2buildTestCase{
					ModuleTypeUnderTest:        tc.moduleType,
					ModuleTypeUnderTestFactory: tc.factory,
					Blueprint:                  fmt.Sprintf(bp, tc.moduleType),
					ExpectedBazelTargets:       expectedBazelTargets,
					Filesystem:                 otherGenruleBp(tc.moduleType),
				})
		})
	}
}

func TestGenruleLocationsLabelShouldSubstituteFirstToolLabel(t *testing.T) {
	testCases := []struct {
		moduleType string
		factory    android.ModuleFactory
		hod        android.HostOrDeviceSupported
	}{
		{
			moduleType: "genrule",
			factory:    genrule.GenRuleFactory,
		},
		{
			moduleType: "cc_genrule",
			factory:    cc.GenRuleFactory,
			hod:        android.DeviceSupported,
		},
		{
			moduleType: "java_genrule",
			factory:    java.GenRuleFactory,
			hod:        android.DeviceSupported,
		},
		{
			moduleType: "java_genrule_host",
			factory:    java.GenRuleFactoryHost,
			hod:        android.HostSupported,
		},
	}

	bp := `%s {
    name: "foo",
    out: ["foo.out"],
    srcs: ["foo.in"],
    tool_files: [":foo.tool", ":other.tool"],
    cmd: "$(locations) -s $(out) $(in)",
    bazel_module: { bp2build_available: true },
}`

	for _, tc := range testCases {
		moduleAttrs := AttrNameToString{
			"cmd":  `"$(locations //other:foo.tool) -s $(OUTS) $(SRCS)"`,
			"outs": `["foo.out"]`,
			"srcs": `["foo.in"]`,
			"tools": `[
        "//other:foo.tool",
        "//other:other.tool",
    ]`,
		}

		expectedBazelTargets := []string{
			makeBazelTargetHostOrDevice("genrule", "foo", moduleAttrs, tc.hod),
		}

		t.Run(tc.moduleType, func(t *testing.T) {
			RunBp2BuildTestCase(t, func(ctx android.RegistrationContext) {},
				Bp2buildTestCase{
					ModuleTypeUnderTest:        tc.moduleType,
					ModuleTypeUnderTestFactory: tc.factory,
					Blueprint:                  fmt.Sprintf(bp, tc.moduleType),
					ExpectedBazelTargets:       expectedBazelTargets,
					Filesystem:                 otherGenruleBp(tc.moduleType),
				})
		})
	}
}

func TestGenruleWithoutToolsOrToolFiles(t *testing.T) {
	testCases := []struct {
		moduleType string
		factory    android.ModuleFactory
		hod        android.HostOrDeviceSupported
	}{
		{
			moduleType: "genrule",
			factory:    genrule.GenRuleFactory,
		},
		{
			moduleType: "cc_genrule",
			factory:    cc.GenRuleFactory,
			hod:        android.DeviceSupported,
		},
		{
			moduleType: "java_genrule",
			factory:    java.GenRuleFactory,
			hod:        android.DeviceSupported,
		},
		{
			moduleType: "java_genrule_host",
			factory:    java.GenRuleFactoryHost,
			hod:        android.HostSupported,
		},
	}

	bp := `%s {
    name: "foo",
    out: ["foo.out"],
    srcs: ["foo.in"],
    cmd: "cp $(in) $(out)",
    bazel_module: { bp2build_available: true },
}`

	for _, tc := range testCases {
		moduleAttrs := AttrNameToString{
			"cmd":  `"cp $(SRCS) $(OUTS)"`,
			"outs": `["foo.out"]`,
			"srcs": `["foo.in"]`,
		}

		expectedBazelTargets := []string{
			makeBazelTargetHostOrDevice("genrule", "foo", moduleAttrs, tc.hod),
		}

		t.Run(tc.moduleType, func(t *testing.T) {
			RunBp2BuildTestCase(t, func(ctx android.RegistrationContext) {},
				Bp2buildTestCase{
					ModuleTypeUnderTest:        tc.moduleType,
					ModuleTypeUnderTestFactory: tc.factory,
					Blueprint:                  fmt.Sprintf(bp, tc.moduleType),
					ExpectedBazelTargets:       expectedBazelTargets,
				})
		})
	}
}

func TestGenruleBp2BuildInlinesDefaults(t *testing.T) {
	testCases := []Bp2buildTestCase{
		{
			Description: "genrule applies properties from a genrule_defaults dependency if not specified",
			Blueprint: `genrule_defaults {
    name: "gen_defaults",
    cmd: "do-something $(in) $(out)",
}
genrule {
    name: "gen",
    out: ["out"],
    srcs: ["in1"],
    defaults: ["gen_defaults"],
    bazel_module: { bp2build_available: true },
}
`,
			ExpectedBazelTargets: []string{
				MakeBazelTargetNoRestrictions("genrule", "gen", AttrNameToString{
					"cmd":  `"do-something $(SRCS) $(OUTS)"`,
					"outs": `["out"]`,
					"srcs": `["in1"]`,
				}),
			},
		},
		{
			Description: "genrule does merges properties from a genrule_defaults dependency, latest-first",
			Blueprint: `genrule_defaults {
    name: "gen_defaults",
    out: ["out-from-defaults"],
    srcs: ["in-from-defaults"],
    cmd: "cmd-from-defaults",
}
genrule {
    name: "gen",
    out: ["out"],
    srcs: ["in1"],
    defaults: ["gen_defaults"],
    cmd: "do-something $(in) $(out)",
    bazel_module: { bp2build_available: true },
}
`,
			ExpectedBazelTargets: []string{
				MakeBazelTargetNoRestrictions("genrule", "gen", AttrNameToString{
					"cmd": `"do-something $(SRCS) $(OUTS)"`,
					"outs": `[
        "out-from-defaults",
        "out",
    ]`,
					"srcs": `[
        "in-from-defaults",
        "in1",
    ]`,
				}),
			},
		},
		{
			Description: "genrule applies properties from list of genrule_defaults",
			Blueprint: `genrule_defaults {
    name: "gen_defaults1",
    cmd: "cp $(in) $(out)",
}

genrule_defaults {
    name: "gen_defaults2",
    srcs: ["in1"],
}

genrule {
    name: "gen",
    out: ["out"],
    defaults: ["gen_defaults1", "gen_defaults2"],
    bazel_module: { bp2build_available: true },
}
`,
			ExpectedBazelTargets: []string{
				MakeBazelTargetNoRestrictions("genrule", "gen", AttrNameToString{
					"cmd":  `"cp $(SRCS) $(OUTS)"`,
					"outs": `["out"]`,
					"srcs": `["in1"]`,
				}),
			},
		},
		{
			Description: "genrule applies properties from genrule_defaults transitively",
			Blueprint: `genrule_defaults {
    name: "gen_defaults1",
    defaults: ["gen_defaults2"],
    cmd: "cmd1 $(in) $(out)", // overrides gen_defaults2's cmd property value.
}

genrule_defaults {
    name: "gen_defaults2",
    defaults: ["gen_defaults3"],
    cmd: "cmd2 $(in) $(out)",
    out: ["out-from-2"],
    srcs: ["in1"],
}

genrule_defaults {
    name: "gen_defaults3",
    out: ["out-from-3"],
    srcs: ["srcs-from-3"],
}

genrule {
    name: "gen",
    out: ["out"],
    defaults: ["gen_defaults1"],
    bazel_module: { bp2build_available: true },
}
`,
			ExpectedBazelTargets: []string{
				MakeBazelTargetNoRestrictions("genrule", "gen", AttrNameToString{
					"cmd": `"cmd1 $(SRCS) $(OUTS)"`,
					"outs": `[
        "out-from-3",
        "out-from-2",
        "out",
    ]`,
					"srcs": `[
        "srcs-from-3",
        "in1",
    ]`,
				}),
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Description, func(t *testing.T) {
			runGenruleTestCase(t, testCase)
		})
	}
}

func TestCcGenruleArchAndExcludeSrcs(t *testing.T) {
	name := "cc_genrule with arch"
	bp := `
	cc_genrule {
		name: "foo",
		srcs: [
			"foo1.in",
			"foo2.in",
		],
		exclude_srcs: ["foo2.in"],
		arch: {
			arm: {
				srcs: [
					"foo1_arch.in",
					"foo2_arch.in",
				],
				exclude_srcs: ["foo2_arch.in"],
			},
		},
		cmd: "cat $(in) > $(out)",
		bazel_module: { bp2build_available: true },
	}`

	expectedBazelAttrs := AttrNameToString{
		"srcs": `["foo1.in"] + select({
        "//build/bazel/platforms/arch:arm": ["foo1_arch.in"],
        "//conditions:default": [],
    })`,
		"cmd":                    `"cat $(SRCS) > $(OUTS)"`,
		"target_compatible_with": `["//build/bazel/platforms/os:android"]`,
	}

	expectedBazelTargets := []string{
		MakeBazelTargetNoRestrictions("genrule", "foo", expectedBazelAttrs),
	}

	t.Run(name, func(t *testing.T) {
		RunBp2BuildTestCase(t, func(ctx android.RegistrationContext) {},
			Bp2buildTestCase{
				ModuleTypeUnderTest:        "cc_genrule",
				ModuleTypeUnderTestFactory: cc.GenRuleFactory,
				Blueprint:                  bp,
				ExpectedBazelTargets:       expectedBazelTargets,
			})
	})
}
