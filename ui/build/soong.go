// Copyright 2017 Google Inc. All rights reserved.
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

package build

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"android/soong/bazel"
	"android/soong/ui/metrics"
	"android/soong/ui/status"

	"android/soong/shared"

	"github.com/google/blueprint"
	"github.com/google/blueprint/bootstrap"
	"github.com/google/blueprint/microfactory"
)

const (
	availableEnvFile = "soong.environment.available"
	usedEnvFile      = "soong.environment.used"

	soongBuildTag        = "build"
	bp2buildFilesTag     = "bp2build_files"
	bp2buildWorkspaceTag = "bp2build_workspace"
	jsonModuleGraphTag   = "modulegraph"
	queryviewTag         = "queryview"
	apiBp2buildTag       = "api_bp2build"
	soongDocsTag         = "soong_docs"

	// bootstrapEpoch is used to determine if an incremental build is incompatible with the current
	// version of bootstrap and needs cleaning before continuing the build.  Increment this for
	// incompatible changes, for example when moving the location of the bpglob binary that is
	// executed during bootstrap before the primary builder has had a chance to update the path.
	bootstrapEpoch = 1
)

func writeEnvironmentFile(_ Context, envFile string, envDeps map[string]string) error {
	data, err := shared.EnvFileContents(envDeps)
	if err != nil {
		return err
	}

	return os.WriteFile(envFile, data, 0644)
}

// This uses Android.bp files and various tools to generate <builddir>/build.ninja.
//
// However, the execution of <builddir>/build.ninja happens later in
// build/soong/ui/build/build.go#Build()
//
// We want to rely on as few prebuilts as possible, so we need to bootstrap
// Soong. The process is as follows:
//
// 1. We use "Microfactory", a simple tool to compile Go code, to build
//    first itself, then soong_ui from soong_ui.bash. This binary contains
//    parts of soong_build that are needed to build itself.
// 2. This simplified version of soong_build then reads the Blueprint files
//    that describe itself and emits .bootstrap/build.ninja that describes
//    how to build its full version and use that to produce the final Ninja
//    file Soong emits.
// 3. soong_ui executes .bootstrap/build.ninja
//
// (After this, Kati is executed to parse the Makefiles, but that's not part of
// bootstrapping Soong)

// A tiny struct used to tell Blueprint that it's in bootstrap mode. It would
// probably be nicer to use a flag in bootstrap.Args instead.
type BlueprintConfig struct {
	toolDir                   string
	soongOutDir               string
	outDir                    string
	runGoTests                bool
	debugCompilation          bool
	subninjas                 []string
	primaryBuilderInvocations []bootstrap.PrimaryBuilderInvocation
}

func (c BlueprintConfig) HostToolDir() string {
	return c.toolDir
}

func (c BlueprintConfig) SoongOutDir() string {
	return c.soongOutDir
}

func (c BlueprintConfig) OutDir() string {
	return c.outDir
}

func (c BlueprintConfig) RunGoTests() bool {
	return c.runGoTests
}

func (c BlueprintConfig) DebugCompilation() bool {
	return c.debugCompilation
}

func (c BlueprintConfig) Subninjas() []string {
	return c.subninjas
}

func (c BlueprintConfig) PrimaryBuilderInvocations() []bootstrap.PrimaryBuilderInvocation {
	return c.primaryBuilderInvocations
}

func environmentArgs(config Config, tag string) []string {
	return []string{
		"--available_env", shared.JoinPath(config.SoongOutDir(), availableEnvFile),
		"--used_env", config.UsedEnvFile(tag),
	}
}

func writeEmptyFile(ctx Context, path string) {
	err := os.MkdirAll(filepath.Dir(path), 0777)
	if err != nil {
		ctx.Fatalf("Failed to create parent directories of empty file '%s': %s", path, err)
	}

	if exists, err := fileExists(path); err != nil {
		ctx.Fatalf("Failed to check if file '%s' exists: %s", path, err)
	} else if !exists {
		err = os.WriteFile(path, nil, 0666)
		if err != nil {
			ctx.Fatalf("Failed to create empty file '%s': %s", path, err)
		}
	}
}

func fileExists(path string) (bool, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

type PrimaryBuilderFactory struct {
	name         string
	description  string
	config       Config
	output       string
	specificArgs []string
	debugPort    string
}

func (pb PrimaryBuilderFactory) primaryBuilderInvocation() bootstrap.PrimaryBuilderInvocation {
	commonArgs := make([]string, 0, 0)

	if !pb.config.skipSoongTests {
		commonArgs = append(commonArgs, "-t")
	}

	commonArgs = append(commonArgs, "-l", filepath.Join(pb.config.FileListDir(), "Android.bp.list"))
	invocationEnv := make(map[string]string)
	if pb.debugPort != "" {
		//debug mode
		commonArgs = append(commonArgs, "--delve_listen", pb.debugPort,
			"--delve_path", shared.ResolveDelveBinary())
		// GODEBUG=asyncpreemptoff=1 disables the preemption of goroutines. This
		// is useful because the preemption happens by sending SIGURG to the OS
		// thread hosting the goroutine in question and each signal results in
		// work that needs to be done by Delve; it uses ptrace to debug the Go
		// process and the tracer process must deal with every signal (it is not
		// possible to selectively ignore SIGURG). This makes debugging slower,
		// sometimes by an order of magnitude depending on luck.
		// The original reason for adding async preemption to Go is here:
		// https://github.com/golang/proposal/blob/master/design/24543-non-cooperative-preemption.md
		invocationEnv["GODEBUG"] = "asyncpreemptoff=1"
	}

	var allArgs []string
	allArgs = append(allArgs, pb.specificArgs...)
	allArgs = append(allArgs,
		"--globListDir", pb.name,
		"--globFile", pb.config.NamedGlobFile(pb.name))

	allArgs = append(allArgs, commonArgs...)
	allArgs = append(allArgs, environmentArgs(pb.config, pb.name)...)
	if profileCpu := os.Getenv("SOONG_PROFILE_CPU"); profileCpu != "" {
		allArgs = append(allArgs, "--cpuprofile", profileCpu+"."+pb.name)
	}
	if profileMem := os.Getenv("SOONG_PROFILE_MEM"); profileMem != "" {
		allArgs = append(allArgs, "--memprofile", profileMem+"."+pb.name)
	}
	allArgs = append(allArgs, "Android.bp")

	return bootstrap.PrimaryBuilderInvocation{
		Inputs:      []string{"Android.bp"},
		Outputs:     []string{pb.output},
		Args:        allArgs,
		Description: pb.description,
		// NB: Changing the value of this environment variable will not result in a
		// rebuild. The bootstrap Ninja file will change, but apparently Ninja does
		// not consider changing the pool specified in a statement a change that's
		// worth rebuilding for.
		Console: os.Getenv("SOONG_UNBUFFERED_OUTPUT") == "1",
		Env:     invocationEnv,
	}
}

// bootstrapEpochCleanup deletes files used by bootstrap during incremental builds across
// incompatible changes.  Incompatible changes are marked by incrementing the bootstrapEpoch
// constant.  A tree is considered out of date for the current epoch of the
// .soong.bootstrap.epoch.<epoch> file doesn't exist.
func bootstrapEpochCleanup(ctx Context, config Config) {
	epochFile := fmt.Sprintf(".soong.bootstrap.epoch.%d", bootstrapEpoch)
	epochPath := filepath.Join(config.SoongOutDir(), epochFile)
	if exists, err := fileExists(epochPath); err != nil {
		ctx.Fatalf("failed to check if bootstrap epoch file %q exists: %q", epochPath, err)
	} else if !exists {
		// The tree is out of date for the current epoch, delete files used by bootstrap
		// and force the primary builder to rerun.
		os.Remove(filepath.Join(config.SoongOutDir(), "build.ninja"))
		for _, globFile := range bootstrapGlobFileList(config) {
			os.Remove(globFile)
		}

		// Mark the tree as up to date with the current epoch by writing the epoch marker file.
		writeEmptyFile(ctx, epochPath)
	}
}

func bootstrapGlobFileList(config Config) []string {
	return []string{
		config.NamedGlobFile(soongBuildTag),
		config.NamedGlobFile(bp2buildFilesTag),
		config.NamedGlobFile(jsonModuleGraphTag),
		config.NamedGlobFile(queryviewTag),
		config.NamedGlobFile(apiBp2buildTag),
		config.NamedGlobFile(soongDocsTag),
	}
}

func bootstrapBlueprint(ctx Context, config Config) {
	ctx.BeginTrace(metrics.RunSoong, "blueprint bootstrap")
	defer ctx.EndTrace()

	// Clean up some files for incremental builds across incompatible changes.
	bootstrapEpochCleanup(ctx, config)

	mainSoongBuildExtraArgs := []string{"-o", config.SoongNinjaFile()}
	if config.EmptyNinjaFile() {
		mainSoongBuildExtraArgs = append(mainSoongBuildExtraArgs, "--empty-ninja-file")
	}
	if config.bazelProdMode {
		mainSoongBuildExtraArgs = append(mainSoongBuildExtraArgs, "--bazel-mode")
	}
	if config.bazelDevMode {
		mainSoongBuildExtraArgs = append(mainSoongBuildExtraArgs, "--bazel-mode-dev")
	}
	if config.bazelStagingMode {
		mainSoongBuildExtraArgs = append(mainSoongBuildExtraArgs, "--bazel-mode-staging")
	}
	if config.IsPersistentBazelEnabled() {
		mainSoongBuildExtraArgs = append(mainSoongBuildExtraArgs, "--use-bazel-proxy")
	}
	if len(config.bazelForceEnabledModules) > 0 {
		mainSoongBuildExtraArgs = append(mainSoongBuildExtraArgs, "--bazel-force-enabled-modules="+config.bazelForceEnabledModules)
	}

	queryviewDir := filepath.Join(config.SoongOutDir(), "queryview")
	// The BUILD files will be generated in out/soong/.api_bp2build (no symlinks to src files)
	// The final workspace will be generated in out/soong/api_bp2build
	apiBp2buildDir := filepath.Join(config.SoongOutDir(), ".api_bp2build")

	pbfs := []PrimaryBuilderFactory{
		{
			name:         soongBuildTag,
			description:  fmt.Sprintf("analyzing Android.bp files and generating ninja file at %s", config.SoongNinjaFile()),
			config:       config,
			output:       config.SoongNinjaFile(),
			specificArgs: mainSoongBuildExtraArgs,
		},
		{
			name:         bp2buildFilesTag,
			description:  fmt.Sprintf("converting Android.bp files to BUILD files at %s/bp2build", config.SoongOutDir()),
			config:       config,
			output:       config.Bp2BuildFilesMarkerFile(),
			specificArgs: []string{"--bp2build_marker", config.Bp2BuildFilesMarkerFile()},
		},
		{
			name:         bp2buildWorkspaceTag,
			description:  "Creating Bazel symlink forest",
			config:       config,
			output:       config.Bp2BuildWorkspaceMarkerFile(),
			specificArgs: []string{"--symlink_forest_marker", config.Bp2BuildWorkspaceMarkerFile()},
		},
		{
			name:        jsonModuleGraphTag,
			description: fmt.Sprintf("generating the Soong module graph at %s", config.ModuleGraphFile()),
			config:      config,
			output:      config.ModuleGraphFile(),
			specificArgs: []string{
				"--module_graph_file", config.ModuleGraphFile(),
				"--module_actions_file", config.ModuleActionsFile(),
			},
		},
		{
			name:         queryviewTag,
			description:  fmt.Sprintf("generating the Soong module graph as a Bazel workspace at %s", queryviewDir),
			config:       config,
			output:       config.QueryviewMarkerFile(),
			specificArgs: []string{"--bazel_queryview_dir", queryviewDir},
		},
		{
			name:         apiBp2buildTag,
			description:  fmt.Sprintf("generating BUILD files for API contributions at %s", apiBp2buildDir),
			config:       config,
			output:       config.ApiBp2buildMarkerFile(),
			specificArgs: []string{"--bazel_api_bp2build_dir", apiBp2buildDir},
		},
		{
			name:         soongDocsTag,
			description:  fmt.Sprintf("generating Soong docs at %s", config.SoongDocsHtml()),
			config:       config,
			output:       config.SoongDocsHtml(),
			specificArgs: []string{"--soong_docs", config.SoongDocsHtml()},
		},
	}

	// Figure out which invocations will be run under the debugger:
	//   * SOONG_DELVE if set specifies listening port
	//   * SOONG_DELVE_STEPS if set specifies specific invocations to be debugged, otherwise all are
	debuggedInvocations := make(map[string]bool)
	delvePort := os.Getenv("SOONG_DELVE")
	if delvePort != "" {
		if steps := os.Getenv("SOONG_DELVE_STEPS"); steps != "" {
			var validSteps []string
			for _, pbf := range pbfs {
				debuggedInvocations[pbf.name] = false
				validSteps = append(validSteps, pbf.name)

			}
			for _, step := range strings.Split(steps, ",") {
				if _, ok := debuggedInvocations[step]; ok {
					debuggedInvocations[step] = true
				} else {
					ctx.Fatalf("SOONG_DELVE_STEPS contains unknown soong_build step %s\n"+
						"Valid steps are %v", step, validSteps)
				}
			}
		} else {
			//  SOONG_DELVE_STEPS is not set, run all steps in the debugger
			for _, pbf := range pbfs {
				debuggedInvocations[pbf.name] = true
			}
		}
	}

	var invocations []bootstrap.PrimaryBuilderInvocation
	for _, pbf := range pbfs {
		if debuggedInvocations[pbf.name] {
			pbf.debugPort = delvePort
		}
		pbi := pbf.primaryBuilderInvocation()
		// Some invocations require adjustment:
		switch pbf.name {
		case soongBuildTag:
			if config.BazelBuildEnabled() {
				// Mixed builds call Bazel from soong_build and they therefore need the
				// Bazel workspace to be available. Make that so by adding a dependency on
				// the bp2build marker file to the action that invokes soong_build .
				pbi.OrderOnlyInputs = append(pbi.OrderOnlyInputs, config.Bp2BuildWorkspaceMarkerFile())
			}
		case bp2buildWorkspaceTag:
			pbi.Inputs = append(pbi.Inputs,
				config.Bp2BuildFilesMarkerFile(),
				filepath.Join(config.FileListDir(), "bazel.list"))
		}
		invocations = append(invocations, pbi)
	}

	// The glob .ninja files are subninja'd. However, they are generated during
	// the build itself so we write an empty file if the file does not exist yet
	// so that the subninja doesn't fail on clean builds
	for _, globFile := range bootstrapGlobFileList(config) {
		writeEmptyFile(ctx, globFile)
	}

	blueprintArgs := bootstrap.Args{
		ModuleListFile: filepath.Join(config.FileListDir(), "Android.bp.list"),
		OutFile:        shared.JoinPath(config.SoongOutDir(), "bootstrap.ninja"),
		EmptyNinjaFile: false,
	}

	blueprintCtx := blueprint.NewContext()
	blueprintCtx.AddIncludeTags(config.GetIncludeTags()...)
	blueprintCtx.SetIgnoreUnknownModuleTypes(true)
	blueprintConfig := BlueprintConfig{
		soongOutDir: config.SoongOutDir(),
		toolDir:     config.HostToolDir(),
		outDir:      config.OutDir(),
		runGoTests:  !config.skipSoongTests,
		// If we want to debug soong_build, we need to compile it for debugging
		debugCompilation:          delvePort != "",
		subninjas:                 bootstrapGlobFileList(config),
		primaryBuilderInvocations: invocations,
	}

	// since `bootstrap.ninja` is regenerated unconditionally, we ignore the deps, i.e. little
	// reason to write a `bootstrap.ninja.d` file
	_ = bootstrap.RunBlueprint(blueprintArgs, bootstrap.DoEverything, blueprintCtx, blueprintConfig)
}

func checkEnvironmentFile(currentEnv *Environment, envFile string) {
	getenv := func(k string) string {
		v, _ := currentEnv.Get(k)
		return v
	}

	if stale, _ := shared.StaleEnvFile(envFile, getenv); stale {
		os.Remove(envFile)
	}
}

func runSoong(ctx Context, config Config) {
	ctx.BeginTrace(metrics.RunSoong, "soong")
	defer ctx.EndTrace()

	// We have two environment files: .available is the one with every variable,
	// .used with the ones that were actually used. The latter is used to
	// determine whether Soong needs to be re-run since why re-run it if only
	// unused variables were changed?
	envFile := filepath.Join(config.SoongOutDir(), availableEnvFile)

	// This is done unconditionally, but does not take a measurable amount of time
	bootstrapBlueprint(ctx, config)

	soongBuildEnv := config.Environment().Copy()
	soongBuildEnv.Set("TOP", os.Getenv("TOP"))
	// For Bazel mixed builds.
	soongBuildEnv.Set("BAZEL_PATH", "./build/bazel/bin/bazel")
	// Bazel's HOME var is set to an output subdirectory which doesn't exist. This
	// prevents Bazel from file I/O in the actual user HOME directory.
	soongBuildEnv.Set("BAZEL_HOME", absPath(ctx, filepath.Join(config.BazelOutDir(), "bazelhome")))
	soongBuildEnv.Set("BAZEL_OUTPUT_BASE", config.bazelOutputBase())
	soongBuildEnv.Set("BAZEL_WORKSPACE", absPath(ctx, "."))
	soongBuildEnv.Set("BAZEL_METRICS_DIR", config.BazelMetricsDir())
	soongBuildEnv.Set("LOG_DIR", config.LogsDir())
	soongBuildEnv.Set("BAZEL_DEPS_FILE", absPath(ctx, filepath.Join(config.BazelOutDir(), "bazel.list")))

	// For Soong bootstrapping tests
	if os.Getenv("ALLOW_MISSING_DEPENDENCIES") == "true" {
		soongBuildEnv.Set("ALLOW_MISSING_DEPENDENCIES", "true")
	}

	err := writeEnvironmentFile(ctx, envFile, soongBuildEnv.AsMap())
	if err != nil {
		ctx.Fatalf("failed to write environment file %s: %s", envFile, err)
	}

	func() {
		ctx.BeginTrace(metrics.RunSoong, "environment check")
		defer ctx.EndTrace()

		checkEnvironmentFile(soongBuildEnv, config.UsedEnvFile(soongBuildTag))

		if config.BazelBuildEnabled() || config.Bp2Build() {
			checkEnvironmentFile(soongBuildEnv, config.UsedEnvFile(bp2buildFilesTag))
		}

		if config.JsonModuleGraph() {
			checkEnvironmentFile(soongBuildEnv, config.UsedEnvFile(jsonModuleGraphTag))
		}

		if config.Queryview() {
			checkEnvironmentFile(soongBuildEnv, config.UsedEnvFile(queryviewTag))
		}

		if config.ApiBp2build() {
			checkEnvironmentFile(soongBuildEnv, config.UsedEnvFile(apiBp2buildTag))
		}

		if config.SoongDocs() {
			checkEnvironmentFile(soongBuildEnv, config.UsedEnvFile(soongDocsTag))
		}
	}()

	runMicrofactory(ctx, config, "bpglob", "github.com/google/blueprint/bootstrap/bpglob",
		map[string]string{"github.com/google/blueprint": "build/blueprint"})

	ninja := func(name, ninjaFile string, targets ...string) {
		ctx.BeginTrace(metrics.RunSoong, name)
		defer ctx.EndTrace()

		if config.IsPersistentBazelEnabled() {
			bazelProxy := bazel.NewProxyServer(ctx.Logger, config.OutDir(), filepath.Join(config.SoongOutDir(), "workspace"))
			bazelProxy.Start()
			defer bazelProxy.Close()
		}

		fifo := filepath.Join(config.OutDir(), ".ninja_fifo")
		nr := status.NewNinjaReader(ctx, ctx.Status.StartTool(), fifo)
		defer nr.Close()

		ninjaArgs := []string{
			"-d", "keepdepfile",
			"-d", "stats",
			"-o", "usesphonyoutputs=yes",
			"-o", "preremoveoutputs=yes",
			"-w", "dupbuild=err",
			"-w", "outputdir=err",
			"-w", "missingoutfile=err",
			"-j", strconv.Itoa(config.Parallel()),
			"--frontend_file", fifo,
			"-f", filepath.Join(config.SoongOutDir(), ninjaFile),
		}

		if extra, ok := config.Environment().Get("SOONG_UI_NINJA_ARGS"); ok {
			ctx.Printf(`CAUTION: arguments in $SOONG_UI_NINJA_ARGS=%q, e.g. "-n", can make soong_build FAIL or INCORRECT`, extra)
			ninjaArgs = append(ninjaArgs, strings.Fields(extra)...)
		}

		ninjaArgs = append(ninjaArgs, targets...)
		cmd := Command(ctx, config, "soong "+name,
			config.PrebuiltBuildTool("ninja"), ninjaArgs...)

		var ninjaEnv Environment

		// This is currently how the command line to invoke soong_build finds the
		// root of the source tree and the output root
		ninjaEnv.Set("TOP", os.Getenv("TOP"))

		cmd.Environment = &ninjaEnv
		cmd.Sandbox = soongSandbox
		cmd.RunAndStreamOrFatal()
	}

	targets := make([]string, 0, 0)

	if config.JsonModuleGraph() {
		targets = append(targets, config.ModuleGraphFile())
	}

	if config.Bp2Build() {
		targets = append(targets, config.Bp2BuildWorkspaceMarkerFile())
	}

	if config.Queryview() {
		targets = append(targets, config.QueryviewMarkerFile())
	}

	if config.ApiBp2build() {
		targets = append(targets, config.ApiBp2buildMarkerFile())
	}

	if config.SoongDocs() {
		targets = append(targets, config.SoongDocsHtml())
	}

	if config.SoongBuildInvocationNeeded() {
		// This build generates <builddir>/build.ninja, which is used later by build/soong/ui/build/build.go#Build().
		targets = append(targets, config.SoongNinjaFile())
	}

	ninja("bootstrap", "bootstrap.ninja", targets...)

	distGzipFile(ctx, config, config.SoongNinjaFile(), "soong")
	distFile(ctx, config, config.SoongVarsFile(), "soong")

	if !config.SkipKati() {
		distGzipFile(ctx, config, config.SoongAndroidMk(), "soong")
		distGzipFile(ctx, config, config.SoongMakeVarsMk(), "soong")
	}

	if config.JsonModuleGraph() {
		distGzipFile(ctx, config, config.ModuleGraphFile(), "soong")
	}
}

func runMicrofactory(ctx Context, config Config, name string, pkg string, mapping map[string]string) {
	ctx.BeginTrace(metrics.RunSoong, name)
	defer ctx.EndTrace()
	cfg := microfactory.Config{TrimPath: absPath(ctx, ".")}
	for pkgPrefix, pathPrefix := range mapping {
		cfg.Map(pkgPrefix, pathPrefix)
	}

	exePath := filepath.Join(config.SoongOutDir(), name)
	dir := filepath.Dir(exePath)
	if err := os.MkdirAll(dir, 0777); err != nil {
		ctx.Fatalf("cannot create %s: %s", dir, err)
	}
	if _, err := microfactory.Build(&cfg, exePath, pkg); err != nil {
		ctx.Fatalf("failed to build %s: %s", name, err)
	}
}
