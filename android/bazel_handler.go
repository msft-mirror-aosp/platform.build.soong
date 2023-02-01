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
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"android/soong/android/allowlists"
	"android/soong/bazel/cquery"
	"android/soong/shared"

	"github.com/google/blueprint"

	"android/soong/bazel"
)

var (
	_                 = pctx.HostBinToolVariable("bazelBuildRunfilesTool", "build-runfiles")
	buildRunfilesRule = pctx.AndroidStaticRule("bazelBuildRunfiles", blueprint.RuleParams{
		Command:     "${bazelBuildRunfilesTool} ${in} ${outDir}",
		Depfile:     "",
		Description: "",
		CommandDeps: []string{"${bazelBuildRunfilesTool}"},
	}, "outDir")
)

func init() {
	RegisterMixedBuildsMutator(InitRegistrationContext)
}

func RegisterMixedBuildsMutator(ctx RegistrationContext) {
	ctx.FinalDepsMutators(func(ctx RegisterMutatorsContext) {
		ctx.BottomUp("mixed_builds_prep", mixedBuildsPrepareMutator).Parallel()
	})
}

func mixedBuildsPrepareMutator(ctx BottomUpMutatorContext) {
	if m := ctx.Module(); m.Enabled() {
		if mixedBuildMod, ok := m.(MixedBuildBuildable); ok {
			if mixedBuildMod.IsMixedBuildSupported(ctx) && MixedBuildsEnabled(ctx) {
				mixedBuildMod.QueueBazelCall(ctx)
			}
		}
	}
}

type cqueryRequest interface {
	// Name returns a string name for this request type. Such request type names must be unique,
	// and must only consist of alphanumeric characters.
	Name() string

	// StarlarkFunctionBody returns a starlark function body to process this request type.
	// The returned string is the body of a Starlark function which obtains
	// all request-relevant information about a target and returns a string containing
	// this information.
	// The function should have the following properties:
	//   - The arguments are `target` (a configured target) and `id_string` (the label + configuration).
	//   - The return value must be a string.
	//   - The function body should not be indented outside of its own scope.
	StarlarkFunctionBody() string
}

// Portion of cquery map key to describe target configuration.
type configKey struct {
	arch   string
	osType OsType
}

func (c configKey) String() string {
	return fmt.Sprintf("%s::%s", c.arch, c.osType)
}

// Map key to describe bazel cquery requests.
type cqueryKey struct {
	label       string
	requestType cqueryRequest
	configKey   configKey
}

func makeCqueryKey(label string, cqueryRequest cqueryRequest, cfgKey configKey) cqueryKey {
	if strings.HasPrefix(label, "//") {
		// Normalize Bazel labels to specify main repository explicitly.
		label = "@" + label
	}
	return cqueryKey{label, cqueryRequest, cfgKey}
}

func (c cqueryKey) String() string {
	return fmt.Sprintf("cquery(%s,%s,%s)", c.label, c.requestType.Name(), c.configKey)
}

// BazelContext is a context object useful for interacting with Bazel during
// the course of a build. Use of Bazel to evaluate part of the build graph
// is referred to as a "mixed build". (Some modules are managed by Soong,
// some are managed by Bazel). To facilitate interop between these build
// subgraphs, Soong may make requests to Bazel and evaluate their responses
// so that Soong modules may accurately depend on Bazel targets.
type BazelContext interface {
	// Add a cquery request to the bazel request queue. All queued requests
	// will be sent to Bazel on a subsequent invocation of InvokeBazel.
	QueueBazelRequest(label string, requestType cqueryRequest, cfgKey configKey)

	// ** Cquery Results Retrieval Functions
	// The below functions pertain to retrieving cquery results from a prior
	// InvokeBazel function call and parsing the results.

	// Returns result files built by building the given bazel target label.
	GetOutputFiles(label string, cfgKey configKey) ([]string, error)

	// Returns the results of GetOutputFiles and GetCcObjectFiles in a single query (in that order).
	GetCcInfo(label string, cfgKey configKey) (cquery.CcInfo, error)

	// Returns the executable binary resultant from building together the python sources
	// TODO(b/232976601): Remove.
	GetPythonBinary(label string, cfgKey configKey) (string, error)

	// Returns the results of the GetApexInfo query (including output files)
	GetApexInfo(label string, cfgkey configKey) (cquery.ApexInfo, error)

	// Returns the results of the GetCcUnstrippedInfo query
	GetCcUnstrippedInfo(label string, cfgkey configKey) (cquery.CcUnstrippedInfo, error)

	// ** end Cquery Results Retrieval Functions

	// Issues commands to Bazel to receive results for all cquery requests
	// queued in the BazelContext. The ctx argument is optional and is only
	// used for performance data collection
	InvokeBazel(config Config, ctx *Context) error

	// Returns true if Bazel handling is enabled for the module with the given name.
	// Note that this only implies "bazel mixed build" allowlisting. The caller
	// should independently verify the module is eligible for Bazel handling
	// (for example, that it is MixedBuildBuildable).
	IsModuleNameAllowed(moduleName string) bool

	// Returns the bazel output base (the root directory for all bazel intermediate outputs).
	OutputBase() string

	// Returns build statements which should get registered to reflect Bazel's outputs.
	BuildStatementsToRegister() []bazel.BuildStatement

	// Returns the depsets defined in Bazel's aquery response.
	AqueryDepsets() []bazel.AqueryDepset
}

type bazelRunner interface {
	createBazelCommand(paths *bazelPaths, runName bazel.RunName, command bazelCommand, extraFlags ...string) *exec.Cmd
	issueBazelCommand(bazelCmd *exec.Cmd) (output string, errorMessage string, error error)
}

type bazelPaths struct {
	homeDir       string
	bazelPath     string
	outputBase    string
	workspaceDir  string
	soongOutDir   string
	metricsDir    string
	bazelDepsFile string
}

// A context object which tracks queued requests that need to be made to Bazel,
// and their results after the requests have been made.
type mixedBuildBazelContext struct {
	bazelRunner
	paths        *bazelPaths
	requests     map[cqueryKey]bool // cquery requests that have not yet been issued to Bazel
	requestMutex sync.Mutex         // requests can be written in parallel

	results map[cqueryKey]string // Results of cquery requests after Bazel invocations

	// Build statements which should get registered to reflect Bazel's outputs.
	buildStatements []bazel.BuildStatement

	// Depsets which should be used for Bazel's build statements.
	depsets []bazel.AqueryDepset

	// Per-module allowlist/denylist functionality to control whether analysis of
	// modules are handled by Bazel. For modules which do not have a Bazel definition
	// (or do not sufficiently support bazel handling via MixedBuildBuildable),
	// this allowlist will have no effect, even if the module is explicitly allowlisted here.
	// Per-module denylist to opt modules out of bazel handling.
	bazelDisabledModules map[string]bool
	// Per-module allowlist to opt modules in to bazel handling.
	bazelEnabledModules map[string]bool
	// If true, modules are bazel-enabled by default, unless present in bazelDisabledModules.
	modulesDefaultToBazel bool

	targetProduct      string
	targetBuildVariant string
}

var _ BazelContext = &mixedBuildBazelContext{}

// A bazel context to use when Bazel is disabled.
type noopBazelContext struct{}

var _ BazelContext = noopBazelContext{}

// A bazel context to use for tests.
type MockBazelContext struct {
	OutputBaseDir string

	LabelToOutputFiles  map[string][]string
	LabelToCcInfo       map[string]cquery.CcInfo
	LabelToPythonBinary map[string]string
	LabelToApexInfo     map[string]cquery.ApexInfo
	LabelToCcBinary     map[string]cquery.CcUnstrippedInfo
}

func (m MockBazelContext) QueueBazelRequest(_ string, _ cqueryRequest, _ configKey) {
	panic("unimplemented")
}

func (m MockBazelContext) GetOutputFiles(label string, _ configKey) ([]string, error) {
	result, ok := m.LabelToOutputFiles[label]
	if !ok {
		return []string{}, fmt.Errorf("no target with label %q in LabelToOutputFiles", label)
	}
	return result, nil
}

func (m MockBazelContext) GetCcInfo(label string, _ configKey) (cquery.CcInfo, error) {
	result, ok := m.LabelToCcInfo[label]
	if !ok {
		return cquery.CcInfo{}, fmt.Errorf("no target with label %q in LabelToCcInfo", label)
	}
	return result, nil
}

func (m MockBazelContext) GetPythonBinary(label string, _ configKey) (string, error) {
	result, ok := m.LabelToPythonBinary[label]
	if !ok {
		return "", fmt.Errorf("no target with label %q in LabelToPythonBinary", label)
	}
	return result, nil
}

func (m MockBazelContext) GetApexInfo(label string, _ configKey) (cquery.ApexInfo, error) {
	result, ok := m.LabelToApexInfo[label]
	if !ok {
		return cquery.ApexInfo{}, fmt.Errorf("no target with label %q in LabelToApexInfo", label)
	}
	return result, nil
}

func (m MockBazelContext) GetCcUnstrippedInfo(label string, _ configKey) (cquery.CcUnstrippedInfo, error) {
	result, ok := m.LabelToCcBinary[label]
	if !ok {
		return cquery.CcUnstrippedInfo{}, fmt.Errorf("no target with label %q in LabelToCcBinary", label)
	}
	return result, nil
}

func (m MockBazelContext) InvokeBazel(_ Config, _ *Context) error {
	panic("unimplemented")
}

func (m MockBazelContext) IsModuleNameAllowed(_ string) bool {
	return true
}

func (m MockBazelContext) OutputBase() string { return m.OutputBaseDir }

func (m MockBazelContext) BuildStatementsToRegister() []bazel.BuildStatement {
	return []bazel.BuildStatement{}
}

func (m MockBazelContext) AqueryDepsets() []bazel.AqueryDepset {
	return []bazel.AqueryDepset{}
}

var _ BazelContext = MockBazelContext{}

func (bazelCtx *mixedBuildBazelContext) QueueBazelRequest(label string, requestType cqueryRequest, cfgKey configKey) {
	key := makeCqueryKey(label, requestType, cfgKey)
	bazelCtx.requestMutex.Lock()
	defer bazelCtx.requestMutex.Unlock()
	bazelCtx.requests[key] = true
}

func (bazelCtx *mixedBuildBazelContext) GetOutputFiles(label string, cfgKey configKey) ([]string, error) {
	key := makeCqueryKey(label, cquery.GetOutputFiles, cfgKey)
	if rawString, ok := bazelCtx.results[key]; ok {
		bazelOutput := strings.TrimSpace(rawString)

		return cquery.GetOutputFiles.ParseResult(bazelOutput), nil
	}
	return nil, fmt.Errorf("no bazel response found for %v", key)
}

func (bazelCtx *mixedBuildBazelContext) GetCcInfo(label string, cfgKey configKey) (cquery.CcInfo, error) {
	key := makeCqueryKey(label, cquery.GetCcInfo, cfgKey)
	if rawString, ok := bazelCtx.results[key]; ok {
		bazelOutput := strings.TrimSpace(rawString)
		return cquery.GetCcInfo.ParseResult(bazelOutput)
	}
	return cquery.CcInfo{}, fmt.Errorf("no bazel response found for %v", key)
}

func (bazelCtx *mixedBuildBazelContext) GetPythonBinary(label string, cfgKey configKey) (string, error) {
	key := makeCqueryKey(label, cquery.GetPythonBinary, cfgKey)
	if rawString, ok := bazelCtx.results[key]; ok {
		bazelOutput := strings.TrimSpace(rawString)
		return cquery.GetPythonBinary.ParseResult(bazelOutput), nil
	}
	return "", fmt.Errorf("no bazel response found for %v", key)
}

func (bazelCtx *mixedBuildBazelContext) GetApexInfo(label string, cfgKey configKey) (cquery.ApexInfo, error) {
	key := makeCqueryKey(label, cquery.GetApexInfo, cfgKey)
	if rawString, ok := bazelCtx.results[key]; ok {
		return cquery.GetApexInfo.ParseResult(strings.TrimSpace(rawString))
	}
	return cquery.ApexInfo{}, fmt.Errorf("no bazel response found for %v", key)
}

func (bazelCtx *mixedBuildBazelContext) GetCcUnstrippedInfo(label string, cfgKey configKey) (cquery.CcUnstrippedInfo, error) {
	key := makeCqueryKey(label, cquery.GetCcUnstrippedInfo, cfgKey)
	if rawString, ok := bazelCtx.results[key]; ok {
		return cquery.GetCcUnstrippedInfo.ParseResult(strings.TrimSpace(rawString))
	}
	return cquery.CcUnstrippedInfo{}, fmt.Errorf("no bazel response for %s", key)
}

func (n noopBazelContext) QueueBazelRequest(_ string, _ cqueryRequest, _ configKey) {
	panic("unimplemented")
}

func (n noopBazelContext) GetOutputFiles(_ string, _ configKey) ([]string, error) {
	panic("unimplemented")
}

func (n noopBazelContext) GetCcInfo(_ string, _ configKey) (cquery.CcInfo, error) {
	panic("unimplemented")
}

func (n noopBazelContext) GetPythonBinary(_ string, _ configKey) (string, error) {
	panic("unimplemented")
}

func (n noopBazelContext) GetApexInfo(_ string, _ configKey) (cquery.ApexInfo, error) {
	panic("unimplemented")
}

func (n noopBazelContext) GetCcUnstrippedInfo(_ string, _ configKey) (cquery.CcUnstrippedInfo, error) {
	//TODO implement me
	panic("implement me")
}

func (n noopBazelContext) InvokeBazel(_ Config, _ *Context) error {
	panic("unimplemented")
}

func (m noopBazelContext) OutputBase() string {
	return ""
}

func (n noopBazelContext) IsModuleNameAllowed(_ string) bool {
	return false
}

func (m noopBazelContext) BuildStatementsToRegister() []bazel.BuildStatement {
	return []bazel.BuildStatement{}
}

func (m noopBazelContext) AqueryDepsets() []bazel.AqueryDepset {
	return []bazel.AqueryDepset{}
}

func GetBazelEnabledAndDisabledModules(buildMode SoongBuildMode, forceEnabled map[string]struct{}) (map[string]bool, map[string]bool) {
	disabledModules := map[string]bool{}
	enabledModules := map[string]bool{}
	addToStringSet := func(set map[string]bool, items []string) {
		for _, item := range items {
			set[item] = true
		}
	}

	switch buildMode {
	case BazelProdMode:
		addToStringSet(enabledModules, allowlists.ProdMixedBuildsEnabledList)
		for enabledAdHocModule := range forceEnabled {
			enabledModules[enabledAdHocModule] = true
		}
	case BazelStagingMode:
		// Staging mode includes all prod modules plus all staging modules.
		addToStringSet(enabledModules, allowlists.ProdMixedBuildsEnabledList)
		addToStringSet(enabledModules, allowlists.StagingMixedBuildsEnabledList)
		for enabledAdHocModule := range forceEnabled {
			enabledModules[enabledAdHocModule] = true
		}
	case BazelDevMode:
		addToStringSet(disabledModules, allowlists.MixedBuildsDisabledList)
	default:
		panic("Expected BazelProdMode, BazelStagingMode, or BazelDevMode")
	}
	return enabledModules, disabledModules
}

func GetBazelEnabledModules(buildMode SoongBuildMode) []string {
	enabledModules, disabledModules := GetBazelEnabledAndDisabledModules(buildMode, nil)
	enabledList := make([]string, 0, len(enabledModules))
	for module := range enabledModules {
		if !disabledModules[module] {
			enabledList = append(enabledList, module)
		}
	}
	sort.Strings(enabledList)
	return enabledList
}

func NewBazelContext(c *config) (BazelContext, error) {
	if c.BuildMode != BazelProdMode && c.BuildMode != BazelStagingMode && c.BuildMode != BazelDevMode {
		return noopBazelContext{}, nil
	}

	enabledModules, disabledModules := GetBazelEnabledAndDisabledModules(c.BuildMode, c.BazelModulesForceEnabledByFlag())

	paths := bazelPaths{
		soongOutDir: c.soongOutDir,
	}
	var missing []string
	vars := []struct {
		name string
		ptr  *string

		// True if the environment variable needs to be tracked so that changes to the variable
		// cause the ninja file to be regenerated, false otherwise. False should only be set for
		// environment variables that have no effect on the generated ninja file.
		track bool
	}{
		{"BAZEL_HOME", &paths.homeDir, true},
		{"BAZEL_PATH", &paths.bazelPath, true},
		{"BAZEL_OUTPUT_BASE", &paths.outputBase, true},
		{"BAZEL_WORKSPACE", &paths.workspaceDir, true},
		{"BAZEL_METRICS_DIR", &paths.metricsDir, false},
		{"BAZEL_DEPS_FILE", &paths.bazelDepsFile, true},
	}
	for _, v := range vars {
		if v.track {
			if s := c.Getenv(v.name); len(s) > 1 {
				*v.ptr = s
				continue
			}
		} else if s, ok := c.env[v.name]; ok {
			*v.ptr = s
		} else {
			missing = append(missing, v.name)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required env vars to use bazel: %s", missing)
	}

	targetBuildVariant := "user"
	if c.Eng() {
		targetBuildVariant = "eng"
	} else if c.Debuggable() {
		targetBuildVariant = "userdebug"
	}
	targetProduct := "unknown"
	if c.HasDeviceProduct() {
		targetProduct = c.DeviceProduct()
	}

	return &mixedBuildBazelContext{
		bazelRunner:           &builtinBazelRunner{},
		paths:                 &paths,
		requests:              make(map[cqueryKey]bool),
		modulesDefaultToBazel: c.BuildMode == BazelDevMode,
		bazelEnabledModules:   enabledModules,
		bazelDisabledModules:  disabledModules,
		targetProduct:         targetProduct,
		targetBuildVariant:    targetBuildVariant,
	}, nil
}

func (p *bazelPaths) BazelMetricsDir() string {
	return p.metricsDir
}

func (context *mixedBuildBazelContext) IsModuleNameAllowed(moduleName string) bool {
	if context.bazelDisabledModules[moduleName] {
		return false
	}
	if context.bazelEnabledModules[moduleName] {
		return true
	}
	return context.modulesDefaultToBazel
}

func pwdPrefix() string {
	// Darwin doesn't have /proc
	if runtime.GOOS != "darwin" {
		return "PWD=/proc/self/cwd"
	}
	return ""
}

type bazelCommand struct {
	command string
	// query or label
	expression string
}

type mockBazelRunner struct {
	bazelCommandResults map[bazelCommand]string
	// use *exec.Cmd as a key to get the bazelCommand, the map will be used in issueBazelCommand()
	// Register createBazelCommand() invocations. Later, an
	// issueBazelCommand() invocation can be mapped to the *exec.Cmd instance
	// and then to the expected result via bazelCommandResults
	tokens     map[*exec.Cmd]bazelCommand
	commands   []bazelCommand
	extraFlags []string
}

func (r *mockBazelRunner) createBazelCommand(_ *bazelPaths, _ bazel.RunName,
	command bazelCommand, extraFlags ...string) *exec.Cmd {
	r.commands = append(r.commands, command)
	r.extraFlags = append(r.extraFlags, strings.Join(extraFlags, " "))
	cmd := &exec.Cmd{}
	if r.tokens == nil {
		r.tokens = make(map[*exec.Cmd]bazelCommand)
	}
	r.tokens[cmd] = command
	return cmd
}

func (r *mockBazelRunner) issueBazelCommand(bazelCmd *exec.Cmd) (string, string, error) {
	if command, ok := r.tokens[bazelCmd]; ok {
		return r.bazelCommandResults[command], "", nil
	}
	return "", "", nil
}

type builtinBazelRunner struct{}

// Issues the given bazel command with given build label and additional flags.
// Returns (stdout, stderr, error). The first and second return values are strings
// containing the stdout and stderr of the run command, and an error is returned if
// the invocation returned an error code.
func (r *builtinBazelRunner) issueBazelCommand(bazelCmd *exec.Cmd) (string, string, error) {
	stderr := &bytes.Buffer{}
	bazelCmd.Stderr = stderr
	if output, err := bazelCmd.Output(); err != nil {
		return "", string(stderr.Bytes()),
			fmt.Errorf("bazel command failed: %s\n---command---\n%s\n---env---\n%s\n---stderr---\n%s---",
				err, bazelCmd, strings.Join(bazelCmd.Env, "\n"), stderr)
	} else {
		return string(output), string(stderr.Bytes()), nil
	}
}

func (r *builtinBazelRunner) createBazelCommand(paths *bazelPaths, runName bazel.RunName, command bazelCommand,
	extraFlags ...string) *exec.Cmd {
	cmdFlags := []string{
		"--output_base=" + absolutePath(paths.outputBase),
		command.command,
		command.expression,
		// TODO(asmundak): is it needed in every build?
		"--profile=" + shared.BazelMetricsFilename(paths, runName),

		// Set default platforms to canonicalized values for mixed builds requests.
		// If these are set in the bazelrc, they will have values that are
		// non-canonicalized to @sourceroot labels, and thus be invalid when
		// referenced from the buildroot.
		//
		// The actual platform values here may be overridden by configuration
		// transitions from the buildroot.
		fmt.Sprintf("--extra_toolchains=%s", "//prebuilts/clang/host/linux-x86:all"),

		// We don't need to set --host_platforms because it's set in bazelrc files
		// that the bazel shell script wrapper passes

		// Explicitly disable downloading rules (such as canonical C++ and Java rules) from the network.
		"--experimental_repository_disable_download",

		// Suppress noise
		"--ui_event_filters=-INFO",
		"--noshow_progress",
		"--norun_validations",
	}
	cmdFlags = append(cmdFlags, extraFlags...)

	bazelCmd := exec.Command(paths.bazelPath, cmdFlags...)
	bazelCmd.Dir = absolutePath(paths.syntheticWorkspaceDir())
	extraEnv := []string{
		"HOME=" + paths.homeDir,
		pwdPrefix(),
		"BUILD_DIR=" + absolutePath(paths.soongOutDir),
		// Make OUT_DIR absolute here so build/bazel/bin/bazel uses the correct
		// OUT_DIR at <root>/out, instead of <root>/out/soong/workspace/out.
		"OUT_DIR=" + absolutePath(paths.outDir()),
		// Disables local host detection of gcc; toolchain information is defined
		// explicitly in BUILD files.
		"BAZEL_DO_NOT_DETECT_CPP_TOOLCHAIN=1",
	}
	bazelCmd.Env = append(os.Environ(), extraEnv...)

	return bazelCmd
}

func printableCqueryCommand(bazelCmd *exec.Cmd) string {
	outputString := strings.Join(bazelCmd.Env, " ") + " \"" + strings.Join(bazelCmd.Args, "\" \"") + "\""
	return outputString

}

func (context *mixedBuildBazelContext) mainBzlFileContents() []byte {
	// TODO(cparsons): Define configuration transitions programmatically based
	// on available archs.
	contents := `
#####################################################
# This file is generated by soong_build. Do not edit.
#####################################################

def _config_node_transition_impl(settings, attr):
    if attr.os == "android" and attr.arch == "target":
        target = "{PRODUCT}-{VARIANT}"
    else:
        target = "{PRODUCT}-{VARIANT}_%s_%s" % (attr.os, attr.arch)
    return {
        "//command_line_option:platforms": "@soong_injection//product_config_platforms/products/{PRODUCT}-{VARIANT}:%s" % target,
    }

_config_node_transition = transition(
    implementation = _config_node_transition_impl,
    inputs = [],
    outputs = [
        "//command_line_option:platforms",
    ],
)

def _passthrough_rule_impl(ctx):
    return [DefaultInfo(files = depset(ctx.files.deps))]

config_node = rule(
    implementation = _passthrough_rule_impl,
    attrs = {
        "arch" : attr.string(mandatory = True),
        "os"   : attr.string(mandatory = True),
        "deps" : attr.label_list(cfg = _config_node_transition, allow_files = True),
        "_allowlist_function_transition": attr.label(default = "@bazel_tools//tools/allowlists/function_transition_allowlist"),
    },
)


# Rule representing the root of the build, to depend on all Bazel targets that
# are required for the build. Building this target will build the entire Bazel
# build tree.
mixed_build_root = rule(
    implementation = _passthrough_rule_impl,
    attrs = {
        "deps" : attr.label_list(),
    },
)

def _phony_root_impl(ctx):
    return []

# Rule to depend on other targets but build nothing.
# This is useful as follows: building a target of this rule will generate
# symlink forests for all dependencies of the target, without executing any
# actions of the build.
phony_root = rule(
    implementation = _phony_root_impl,
    attrs = {"deps" : attr.label_list()},
)
`

	productReplacer := strings.NewReplacer(
		"{PRODUCT}", context.targetProduct,
		"{VARIANT}", context.targetBuildVariant)

	return []byte(productReplacer.Replace(contents))
}

func (context *mixedBuildBazelContext) mainBuildFileContents() []byte {
	// TODO(cparsons): Map label to attribute programmatically; don't use hard-coded
	// architecture mapping.
	formatString := `
# This file is generated by soong_build. Do not edit.
load(":main.bzl", "config_node", "mixed_build_root", "phony_root")

%s

mixed_build_root(name = "buildroot",
    deps = [%s],
    testonly = True, # Unblocks testonly deps.
)

phony_root(name = "phonyroot",
    deps = [":buildroot"],
    testonly = True, # Unblocks testonly deps.
)
`
	configNodeFormatString := `
config_node(name = "%s",
    arch = "%s",
    os = "%s",
    deps = [%s],
    testonly = True, # Unblocks testonly deps.
)
`

	configNodesSection := ""

	labelsByConfig := map[string][]string{}
	for val := range context.requests {
		labelString := fmt.Sprintf("\"@%s\"", val.label)
		configString := getConfigString(val)
		labelsByConfig[configString] = append(labelsByConfig[configString], labelString)
	}

	allLabels := []string{}
	for configString, labels := range labelsByConfig {
		configTokens := strings.Split(configString, "|")
		if len(configTokens) != 2 {
			panic(fmt.Errorf("Unexpected config string format: %s", configString))
		}
		archString := configTokens[0]
		osString := configTokens[1]
		targetString := fmt.Sprintf("%s_%s", osString, archString)
		allLabels = append(allLabels, fmt.Sprintf("\":%s\"", targetString))
		labelsString := strings.Join(labels, ",\n            ")
		configNodesSection += fmt.Sprintf(configNodeFormatString, targetString, archString, osString, labelsString)
	}

	return []byte(fmt.Sprintf(formatString, configNodesSection, strings.Join(allLabels, ",\n            ")))
}

func indent(original string) string {
	result := ""
	for _, line := range strings.Split(original, "\n") {
		result += "  " + line + "\n"
	}
	return result
}

// Returns the file contents of the buildroot.cquery file that should be used for the cquery
// expression in order to obtain information about buildroot and its dependencies.
// The contents of this file depend on the mixedBuildBazelContext's requests; requests are enumerated
// and grouped by their request type. The data retrieved for each label depends on its
// request type.
func (context *mixedBuildBazelContext) cqueryStarlarkFileContents() []byte {
	requestTypeToCqueryIdEntries := map[cqueryRequest][]string{}
	for val := range context.requests {
		cqueryId := getCqueryId(val)
		mapEntryString := fmt.Sprintf("%q : True", cqueryId)
		requestTypeToCqueryIdEntries[val.requestType] =
			append(requestTypeToCqueryIdEntries[val.requestType], mapEntryString)
	}
	labelRegistrationMapSection := ""
	functionDefSection := ""
	mainSwitchSection := ""

	mapDeclarationFormatString := `
%s = {
  %s
}
`
	functionDefFormatString := `
def %s(target, id_string):
%s
`
	mainSwitchSectionFormatString := `
  if id_string in %s:
    return id_string + ">>" + %s(target, id_string)
`

	for requestType := range requestTypeToCqueryIdEntries {
		labelMapName := requestType.Name() + "_Labels"
		functionName := requestType.Name() + "_Fn"
		labelRegistrationMapSection += fmt.Sprintf(mapDeclarationFormatString,
			labelMapName,
			strings.Join(requestTypeToCqueryIdEntries[requestType], ",\n  "))
		functionDefSection += fmt.Sprintf(functionDefFormatString,
			functionName,
			indent(requestType.StarlarkFunctionBody()))
		mainSwitchSection += fmt.Sprintf(mainSwitchSectionFormatString,
			labelMapName, functionName)
	}

	formatString := `
# This file is generated by soong_build. Do not edit.

# a drop-in replacement for json.encode(), not available in cquery environment
# TODO(cparsons): bring json module in and remove this function
def json_encode(input):
  # Avoiding recursion by limiting
  #  - a dict to contain anything except a dict
  #  - a list to contain only primitives
  def encode_primitive(p):
    t = type(p)
    if t == "string" or t == "int":
      return repr(p)
    fail("unsupported value '%s' of type '%s'" % (p, type(p)))

  def encode_list(list):
    return "[%s]" % ", ".join([encode_primitive(item) for item in list])

  def encode_list_or_primitive(v):
    return encode_list(v) if type(v) == "list" else encode_primitive(v)

  if type(input) == "dict":
    # TODO(juu): the result is read line by line so can't use '\n' yet
    kv_pairs = [("%s: %s" % (encode_primitive(k), encode_list_or_primitive(v))) for (k, v) in input.items()]
    return "{ %s }" % ", ".join(kv_pairs)
  else:
    return encode_list_or_primitive(input)

{LABEL_REGISTRATION_MAP_SECTION}

{FUNCTION_DEF_SECTION}

def get_arch(target):
  # TODO(b/199363072): filegroups and file targets aren't associated with any
  # specific platform architecture in mixed builds. This is consistent with how
  # Soong treats filegroups, but it may not be the case with manually-written
  # filegroup BUILD targets.
  buildoptions = build_options(target)
  if buildoptions == None:
    # File targets do not have buildoptions. File targets aren't associated with
    #  any specific platform architecture in mixed builds, so use the host.
    return "x86_64|linux"
  platforms = buildoptions["//command_line_option:platforms"]
  if len(platforms) != 1:
    # An individual configured target should have only one platform architecture.
    # Note that it's fine for there to be multiple architectures for the same label,
    # but each is its own configured target.
    fail("expected exactly 1 platform for " + str(target.label) + " but got " + str(platforms))
  platform_name = platforms[0].name
  if platform_name == "host":
    return "HOST"
  if not platform_name.startswith("{TARGET_PRODUCT}-{TARGET_BUILD_VARIANT}"):
    fail("expected platform name of the form '{TARGET_PRODUCT}-{TARGET_BUILD_VARIANT}_android_<arch>' or '{TARGET_PRODUCT}-{TARGET_BUILD_VARIANT}_linux_<arch>', but was " + str(platforms))
  platform_name = platform_name.removeprefix("{TARGET_PRODUCT}-{TARGET_BUILD_VARIANT}").removeprefix("_")
  if not platform_name:
    return "target|android"
  elif platform_name.startswith("android_"):
    return platform_name.removeprefix("android_") + "|android"
  elif platform_name.startswith("linux_"):
    return platform_name.removeprefix("linux_") + "|linux"
  else:
    fail("expected platform name of the form '{TARGET_PRODUCT}-{TARGET_BUILD_VARIANT}_android_<arch>' or '{TARGET_PRODUCT}-{TARGET_BUILD_VARIANT}_linux_<arch>', but was " + str(platforms))

def format(target):
  id_string = str(target.label) + "|" + get_arch(target)

  # TODO(b/248106697): Remove once Bazel is updated to always normalize labels.
  if id_string.startswith("//"):
    id_string = "@" + id_string

  {MAIN_SWITCH_SECTION}

  # This target was not requested via cquery, and thus must be a dependency
  # of a requested target.
  return id_string + ">>NONE"
`
	replacer := strings.NewReplacer(
		"{TARGET_PRODUCT}", context.targetProduct,
		"{TARGET_BUILD_VARIANT}", context.targetBuildVariant,
		"{LABEL_REGISTRATION_MAP_SECTION}", labelRegistrationMapSection,
		"{FUNCTION_DEF_SECTION}", functionDefSection,
		"{MAIN_SWITCH_SECTION}", mainSwitchSection)

	return []byte(replacer.Replace(formatString))
}

// Returns a path containing build-related metadata required for interfacing
// with Bazel. Example: out/soong/bazel.
func (p *bazelPaths) intermediatesDir() string {
	return filepath.Join(p.soongOutDir, "bazel")
}

// Returns the path where the contents of the @soong_injection repository live.
// It is used by Soong to tell Bazel things it cannot over the command line.
func (p *bazelPaths) injectedFilesDir() string {
	return filepath.Join(p.soongOutDir, bazel.SoongInjectionDirName)
}

// Returns the path of the synthetic Bazel workspace that contains a symlink
// forest composed the whole source tree and BUILD files generated by bp2build.
func (p *bazelPaths) syntheticWorkspaceDir() string {
	return filepath.Join(p.soongOutDir, "workspace")
}

// Returns the path to the top level out dir ($OUT_DIR).
func (p *bazelPaths) outDir() string {
	return filepath.Dir(p.soongOutDir)
}

const buildrootLabel = "@soong_injection//mixed_builds:buildroot"

var (
	cqueryCmd = bazelCommand{"cquery", fmt.Sprintf("deps(%s, 2)", buildrootLabel)}
	aqueryCmd = bazelCommand{"aquery", fmt.Sprintf("deps(%s)", buildrootLabel)}
	buildCmd  = bazelCommand{"build", "@soong_injection//mixed_builds:phonyroot"}
)

// Issues commands to Bazel to receive results for all cquery requests
// queued in the BazelContext.
func (context *mixedBuildBazelContext) InvokeBazel(config Config, ctx *Context) error {
	if ctx != nil {
		ctx.EventHandler.Begin("bazel")
		defer ctx.EventHandler.End("bazel")
	}

	if metricsDir := context.paths.BazelMetricsDir(); metricsDir != "" {
		if err := os.MkdirAll(metricsDir, 0777); err != nil {
			return err
		}
	}
	context.results = make(map[cqueryKey]string)
	if err := context.runCquery(ctx); err != nil {
		return err
	}
	if err := context.runAquery(config, ctx); err != nil {
		return err
	}
	if err := context.generateBazelSymlinks(ctx); err != nil {
		return err
	}

	// Clear requests.
	context.requests = map[cqueryKey]bool{}
	return nil
}

func (context *mixedBuildBazelContext) runCquery(ctx *Context) error {
	if ctx != nil {
		ctx.EventHandler.Begin("cquery")
		defer ctx.EventHandler.End("cquery")
	}
	soongInjectionPath := absolutePath(context.paths.injectedFilesDir())
	mixedBuildsPath := filepath.Join(soongInjectionPath, "mixed_builds")
	if _, err := os.Stat(mixedBuildsPath); os.IsNotExist(err) {
		err = os.MkdirAll(mixedBuildsPath, 0777)
		if err != nil {
			return err
		}
	}
	if err := os.WriteFile(filepath.Join(soongInjectionPath, "WORKSPACE.bazel"), []byte{}, 0666); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(mixedBuildsPath, "main.bzl"), context.mainBzlFileContents(), 0666); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(mixedBuildsPath, "BUILD.bazel"), context.mainBuildFileContents(), 0666); err != nil {
		return err
	}
	cqueryFileRelpath := filepath.Join(context.paths.injectedFilesDir(), "buildroot.cquery")
	if err := os.WriteFile(absolutePath(cqueryFileRelpath), context.cqueryStarlarkFileContents(), 0666); err != nil {
		return err
	}

	cqueryCommandWithFlag := context.createBazelCommand(context.paths, bazel.CqueryBuildRootRunName, cqueryCmd,
		"--output=starlark", "--starlark:file="+absolutePath(cqueryFileRelpath))
	cqueryOutput, cqueryErrorMessage, cqueryErr := context.issueBazelCommand(cqueryCommandWithFlag)
	if cqueryErr != nil {
		return cqueryErr
	}
	cqueryCommandPrint := fmt.Sprintf("cquery command line:\n  %s \n\n\n", printableCqueryCommand(cqueryCommandWithFlag))
	if err := os.WriteFile(filepath.Join(soongInjectionPath, "cquery.out"), []byte(cqueryCommandPrint+cqueryOutput), 0666); err != nil {
		return err
	}
	cqueryResults := map[string]string{}
	for _, outputLine := range strings.Split(cqueryOutput, "\n") {
		if strings.Contains(outputLine, ">>") {
			splitLine := strings.SplitN(outputLine, ">>", 2)
			cqueryResults[splitLine[0]] = splitLine[1]
		}
	}
	for val := range context.requests {
		if cqueryResult, ok := cqueryResults[getCqueryId(val)]; ok {
			context.results[val] = cqueryResult
		} else {
			return fmt.Errorf("missing result for bazel target %s. query output: [%s], cquery err: [%s]",
				getCqueryId(val), cqueryOutput, cqueryErrorMessage)
		}
	}
	return nil
}

func (context *mixedBuildBazelContext) runAquery(config Config, ctx *Context) error {
	if ctx != nil {
		ctx.EventHandler.Begin("aquery")
		defer ctx.EventHandler.End("aquery")
	}
	// Issue an aquery command to retrieve action information about the bazel build tree.
	//
	// Use jsonproto instead of proto; actual proto parsing would require a dependency on Bazel's
	// proto sources, which would add a number of unnecessary dependencies.
	extraFlags := []string{"--output=proto", "--include_file_write_contents"}
	if Bool(config.productVariables.ClangCoverage) {
		extraFlags = append(extraFlags, "--collect_code_coverage")
		paths := make([]string, 0, 2)
		if p := config.productVariables.NativeCoveragePaths; len(p) > 0 {
			for i := range p {
				// TODO(b/259404593) convert path wildcard to regex values
				if p[i] == "*" {
					p[i] = ".*"
				}
			}
			paths = append(paths, JoinWithPrefixAndSeparator(p, "+", ","))
		}
		if p := config.productVariables.NativeCoverageExcludePaths; len(p) > 0 {
			paths = append(paths, JoinWithPrefixAndSeparator(p, "-", ","))
		}
		if len(paths) > 0 {
			extraFlags = append(extraFlags, "--instrumentation_filter="+strings.Join(paths, ","))
		}
	}
	aqueryOutput, _, err := context.issueBazelCommand(context.createBazelCommand(context.paths, bazel.AqueryBuildRootRunName, aqueryCmd,
		extraFlags...))
	if err != nil {
		return err
	}
	context.buildStatements, context.depsets, err = bazel.AqueryBuildStatements([]byte(aqueryOutput))
	return err
}

func (context *mixedBuildBazelContext) generateBazelSymlinks(ctx *Context) error {
	if ctx != nil {
		ctx.EventHandler.Begin("symlinks")
		defer ctx.EventHandler.End("symlinks")
	}
	// Issue a build command of the phony root to generate symlink forests for dependencies of the
	// Bazel build. This is necessary because aquery invocations do not generate this symlink forest,
	// but some of symlinks may be required to resolve source dependencies of the build.
	_, _, err := context.issueBazelCommand(context.createBazelCommand(context.paths, bazel.BazelBuildPhonyRootRunName, buildCmd))
	return err
}

func (context *mixedBuildBazelContext) BuildStatementsToRegister() []bazel.BuildStatement {
	return context.buildStatements
}

func (context *mixedBuildBazelContext) AqueryDepsets() []bazel.AqueryDepset {
	return context.depsets
}

func (context *mixedBuildBazelContext) OutputBase() string {
	return context.paths.outputBase
}

// Singleton used for registering BUILD file ninja dependencies (needed
// for correctness of builds which use Bazel.
func BazelSingleton() Singleton {
	return &bazelSingleton{}
}

type bazelSingleton struct{}

func (c *bazelSingleton) GenerateBuildActions(ctx SingletonContext) {
	// bazelSingleton is a no-op if mixed-soong-bazel-builds are disabled.
	if !ctx.Config().IsMixedBuildsEnabled() {
		return
	}

	// Add ninja file dependencies for files which all bazel invocations require.
	bazelBuildList := absolutePath(filepath.Join(
		filepath.Dir(ctx.Config().moduleListFile), "bazel.list"))
	ctx.AddNinjaFileDeps(bazelBuildList)

	data, err := os.ReadFile(bazelBuildList)
	if err != nil {
		ctx.Errorf(err.Error())
	}
	files := strings.Split(strings.TrimSpace(string(data)), "\n")
	for _, file := range files {
		ctx.AddNinjaFileDeps(file)
	}

	for _, depset := range ctx.Config().BazelContext.AqueryDepsets() {
		var outputs []Path
		var orderOnlies []Path
		for _, depsetDepHash := range depset.TransitiveDepSetHashes {
			otherDepsetName := bazelDepsetName(depsetDepHash)
			outputs = append(outputs, PathForPhony(ctx, otherDepsetName))
		}
		for _, artifactPath := range depset.DirectArtifacts {
			pathInBazelOut := PathForBazelOut(ctx, artifactPath)
			if artifactPath == "bazel-out/volatile-status.txt" {
				// See https://bazel.build/docs/user-manual#workspace-status
				orderOnlies = append(orderOnlies, pathInBazelOut)
			} else {
				outputs = append(outputs, pathInBazelOut)
			}
		}
		thisDepsetName := bazelDepsetName(depset.ContentHash)
		ctx.Build(pctx, BuildParams{
			Rule:      blueprint.Phony,
			Outputs:   []WritablePath{PathForPhony(ctx, thisDepsetName)},
			Implicits: outputs,
			OrderOnly: orderOnlies,
		})
	}

	executionRoot := path.Join(ctx.Config().BazelContext.OutputBase(), "execroot", "__main__")
	bazelOutDir := path.Join(executionRoot, "bazel-out")
	for index, buildStatement := range ctx.Config().BazelContext.BuildStatementsToRegister() {
		if len(buildStatement.Command) > 0 {
			rule := NewRuleBuilder(pctx, ctx)
			createCommand(rule.Command(), buildStatement, executionRoot, bazelOutDir, ctx)
			desc := fmt.Sprintf("%s: %s", buildStatement.Mnemonic, buildStatement.OutputPaths)
			rule.Build(fmt.Sprintf("bazel %d", index), desc)
			continue
		}
		// Certain actions returned by aquery (for instance FileWrite) do not contain a command
		// and thus require special treatment. If BuildStatement were an interface implementing
		// buildRule(ctx) function, the code here would just call it.
		// Unfortunately, the BuildStatement is defined in
		// the 'bazel' package, which cannot depend on 'android' package where ctx is defined,
		// because this would cause circular dependency. So, until we move aquery processing
		// to the 'android' package, we need to handle special cases here.
		switch buildStatement.Mnemonic {
		case "FileWrite", "SourceSymlinkManifest":
			out := PathForBazelOut(ctx, buildStatement.OutputPaths[0])
			WriteFileRuleVerbatim(ctx, out, buildStatement.FileContents)
		case "SymlinkTree":
			// build-runfiles arguments are the manifest file and the target directory
			// where it creates the symlink tree according to this manifest (and then
			// writes the MANIFEST file to it).
			outManifest := PathForBazelOut(ctx, buildStatement.OutputPaths[0])
			outManifestPath := outManifest.String()
			if !strings.HasSuffix(outManifestPath, "MANIFEST") {
				panic("the base name of the symlink tree action should be MANIFEST, got " + outManifestPath)
			}
			outDir := filepath.Dir(outManifestPath)
			ctx.Build(pctx, BuildParams{
				Rule:        buildRunfilesRule,
				Output:      outManifest,
				Inputs:      []Path{PathForBazelOut(ctx, buildStatement.InputPaths[0])},
				Description: "symlink tree for " + outDir,
				Args: map[string]string{
					"outDir": outDir,
				},
			})
		default:
			panic(fmt.Sprintf("unhandled build statement: %v", buildStatement))
		}
	}
}

// Register bazel-owned build statements (obtained from the aquery invocation).
func createCommand(cmd *RuleBuilderCommand, buildStatement bazel.BuildStatement, executionRoot string, bazelOutDir string, ctx BuilderContext) {
	// executionRoot is the action cwd.
	cmd.Text(fmt.Sprintf("cd '%s' &&", executionRoot))

	// Remove old outputs, as some actions might not rerun if the outputs are detected.
	if len(buildStatement.OutputPaths) > 0 {
		cmd.Text("rm -rf") // -r because outputs can be Bazel dir/tree artifacts.
		for _, outputPath := range buildStatement.OutputPaths {
			cmd.Text(fmt.Sprintf("'%s'", outputPath))
		}
		cmd.Text("&&")
	}

	for _, pair := range buildStatement.Env {
		// Set per-action env variables, if any.
		cmd.Flag(pair.Key + "=" + pair.Value)
	}

	// The actual Bazel action.
	if len(buildStatement.Command) > 16*1024 {
		commandFile := PathForBazelOut(ctx, buildStatement.OutputPaths[0]+".sh")
		WriteFileRule(ctx, commandFile, buildStatement.Command)

		cmd.Text("bash").Text(buildStatement.OutputPaths[0] + ".sh").Implicit(commandFile)
	} else {
		cmd.Text(buildStatement.Command)
	}

	for _, outputPath := range buildStatement.OutputPaths {
		cmd.ImplicitOutput(PathForBazelOut(ctx, outputPath))
	}
	for _, inputPath := range buildStatement.InputPaths {
		cmd.Implicit(PathForBazelOut(ctx, inputPath))
	}
	for _, inputDepsetHash := range buildStatement.InputDepsetHashes {
		otherDepsetName := bazelDepsetName(inputDepsetHash)
		cmd.Implicit(PathForPhony(ctx, otherDepsetName))
	}

	if depfile := buildStatement.Depfile; depfile != nil {
		// The paths in depfile are relative to `executionRoot`.
		// Hence, they need to be corrected by replacing "bazel-out"
		// with the full `bazelOutDir`.
		// Otherwise, implicit outputs and implicit inputs under "bazel-out/"
		// would be deemed missing.
		// (Note: The regexp uses a capture group because the version of sed
		//  does not support a look-behind pattern.)
		replacement := fmt.Sprintf(`&& sed -i'' -E 's@(^|\s|")bazel-out/@\1%s/@g' '%s'`,
			bazelOutDir, *depfile)
		cmd.Text(replacement)
		cmd.ImplicitDepFile(PathForBazelOut(ctx, *depfile))
	}

	for _, symlinkPath := range buildStatement.SymlinkPaths {
		cmd.ImplicitSymlinkOutput(PathForBazelOut(ctx, symlinkPath))
	}
}

func getCqueryId(key cqueryKey) string {
	return key.label + "|" + getConfigString(key)
}

func getConfigString(key cqueryKey) string {
	arch := key.configKey.arch
	if len(arch) == 0 || arch == "common" {
		if key.configKey.osType.Class == Device {
			// For the generic Android, the expected result is "target|android", which
			// corresponds to the product_variable_config named "android_target" in
			// build/bazel/platforms/BUILD.bazel.
			arch = "target"
		} else {
			// Use host platform, which is currently hardcoded to be x86_64.
			arch = "x86_64"
		}
	}
	osName := key.configKey.osType.Name
	if len(osName) == 0 || osName == "common_os" || osName == "linux_glibc" || osName == "linux_musl" {
		// Use host OS, which is currently hardcoded to be linux.
		osName = "linux"
	}
	return arch + "|" + osName
}

func GetConfigKey(ctx BaseModuleContext) configKey {
	return configKey{
		// use string because Arch is not a valid key in go
		arch:   ctx.Arch().String(),
		osType: ctx.Os(),
	}
}

func bazelDepsetName(contentHash string) string {
	return fmt.Sprintf("bazel_depset_%s", contentHash)
}
