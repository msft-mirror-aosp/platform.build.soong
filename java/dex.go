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

package java

import (
	"strconv"
	"strings"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"

	"android/soong/android"
	"android/soong/remoteexec"
)

type DexProperties struct {
	// If set to true, compile dex regardless of installable.  Defaults to false.
	Compile_dex *bool

	// list of module-specific flags that will be used for dex compiles
	Dxflags []string `android:"arch_variant"`

	// A list of files containing rules that specify the classes to keep in the main dex file.
	Main_dex_rules []string `android:"path"`

	Optimize struct {
		// If false, disable all optimization.  Defaults to true for android_app and
		// android_test_helper_app modules, false for android_test, java_library, and java_test modules.
		Enabled *bool
		// True if the module containing this has it set by default.
		EnabledByDefault bool `blueprint:"mutated"`

		// Whether to continue building even if warnings are emitted.  Defaults to true.
		Ignore_warnings *bool

		// If true, runs R8 in Proguard compatibility mode, otherwise runs R8 in full mode.
		// Defaults to false for apps, true for libraries and tests.
		Proguard_compatibility *bool

		// If true, optimize for size by removing unused code.  Defaults to true for apps,
		// false for libraries and tests.
		Shrink *bool

		// If true, optimize bytecode.  Defaults to false.
		Optimize *bool

		// If true, obfuscate bytecode.  Defaults to false.
		Obfuscate *bool

		// If true, do not use the flag files generated by aapt that automatically keep
		// classes referenced by the app manifest.  Defaults to false.
		No_aapt_flags *bool

		// If true, optimize for size by removing unused resources. Defaults to false.
		Shrink_resources *bool

		// If true, use optimized resource shrinking in R8, overriding the
		// Shrink_resources setting. Defaults to false.
		// Optimized shrinking means that R8 will trace and treeshake resources together with code
		// and apply additional optimizations. This implies non final fields in the R classes.
		Optimized_shrink_resources *bool

		// Flags to pass to proguard.
		Proguard_flags []string

		// Specifies the locations of files containing proguard flags.
		Proguard_flags_files []string `android:"path"`

		// If true, transitive reverse dependencies of this module will have this
		// module's proguard spec appended to their optimization action
		Export_proguard_flags_files *bool
	}

	// Keep the data uncompressed. We always need uncompressed dex for execution,
	// so this might actually save space by avoiding storing the same data twice.
	// This defaults to reasonable value based on module and should not be set.
	// It exists only to support ART tests.
	Uncompress_dex *bool

	// Exclude kotlinc generate files: *.kotlin_module, *.kotlin_builtins. Defaults to false.
	Exclude_kotlinc_generated_files *bool

	// Disable dex container (also known as "multi-dex").
	// This may be necessary as a temporary workaround to mask toolchain bugs (see b/341652226).
	No_dex_container *bool
}

type dexer struct {
	dexProperties DexProperties

	// list of extra proguard flag files
	extraProguardFlagsFiles android.Paths
	proguardDictionary      android.OptionalPath
	proguardConfiguration   android.OptionalPath
	proguardUsageZip        android.OptionalPath
	resourcesInput          android.OptionalPath
	resourcesOutput         android.OptionalPath

	providesTransitiveHeaderJarsForR8
}

func (d *dexer) effectiveOptimizeEnabled() bool {
	return BoolDefault(d.dexProperties.Optimize.Enabled, d.dexProperties.Optimize.EnabledByDefault)
}

func (d *DexProperties) resourceShrinkingEnabled(ctx android.ModuleContext) bool {
	return !ctx.Config().Eng() && BoolDefault(d.Optimize.Optimized_shrink_resources, Bool(d.Optimize.Shrink_resources))
}

func (d *DexProperties) optimizedResourceShrinkingEnabled(ctx android.ModuleContext) bool {
	return d.resourceShrinkingEnabled(ctx) && BoolDefault(d.Optimize.Optimized_shrink_resources, ctx.Config().UseOptimizedResourceShrinkingByDefault())
}

func (d *dexer) optimizeOrObfuscateEnabled() bool {
	return d.effectiveOptimizeEnabled() && (proptools.Bool(d.dexProperties.Optimize.Optimize) || proptools.Bool(d.dexProperties.Optimize.Obfuscate))
}

var d8, d8RE = pctx.MultiCommandRemoteStaticRules("d8",
	blueprint.RuleParams{
		Command: `rm -rf "$outDir" && mkdir -p "$outDir" && ` +
			`$d8Template${config.D8Cmd} ${config.D8Flags} $d8Flags --output $outDir --no-dex-input-jar $in && ` +
			`$zipTemplate${config.SoongZipCmd} $zipFlags -o $outDir/classes.dex.jar -C $outDir -f "$outDir/classes*.dex" && ` +
			`${config.MergeZipsCmd} -D -stripFile "**/*.class" $mergeZipsFlags $out $outDir/classes.dex.jar $in && ` +
			`rm -f "$outDir"/classes*.dex "$outDir/classes.dex.jar"`,
		CommandDeps: []string{
			"${config.D8Cmd}",
			"${config.D8Jar}",
			"${config.SoongZipCmd}",
			"${config.MergeZipsCmd}",
		},
	}, map[string]*remoteexec.REParams{
		"$d8Template": &remoteexec.REParams{
			Labels:          map[string]string{"type": "compile", "compiler": "d8"},
			Inputs:          []string{"${config.D8Jar}"},
			ExecStrategy:    "${config.RED8ExecStrategy}",
			ToolchainInputs: []string{"${config.JavaCmd}"},
			Platform:        map[string]string{remoteexec.PoolKey: "${config.REJavaPool}"},
		},
		"$zipTemplate": &remoteexec.REParams{
			Labels:       map[string]string{"type": "tool", "name": "soong_zip"},
			Inputs:       []string{"${config.SoongZipCmd}", "$outDir"},
			OutputFiles:  []string{"$outDir/classes.dex.jar"},
			ExecStrategy: "${config.RED8ExecStrategy}",
			Platform:     map[string]string{remoteexec.PoolKey: "${config.REJavaPool}"},
		},
	}, []string{"outDir", "d8Flags", "zipFlags", "mergeZipsFlags"}, nil)

var r8, r8RE = pctx.MultiCommandRemoteStaticRules("r8",
	blueprint.RuleParams{
		Command: `rm -rf "$outDir" && mkdir -p "$outDir" && ` +
			`rm -f "$outDict" && rm -f "$outConfig" && rm -rf "${outUsageDir}" && ` +
			`mkdir -p $$(dirname ${outUsage}) && ` +
			`$r8Template${config.R8Cmd} ${config.R8Flags} $r8Flags -injars $in --output $outDir ` +
			`--no-data-resources ` +
			`-printmapping ${outDict} ` +
			`-printconfiguration ${outConfig} ` +
			`-printusage ${outUsage} ` +
			`--deps-file ${out}.d && ` +
			`touch "${outDict}" "${outConfig}" "${outUsage}" && ` +
			`${config.SoongZipCmd} -o ${outUsageZip} -C ${outUsageDir} -f ${outUsage} && ` +
			`rm -rf ${outUsageDir} && ` +
			`$zipTemplate${config.SoongZipCmd} $zipFlags -o $outDir/classes.dex.jar -C $outDir -f "$outDir/classes*.dex" && ` +
			`${config.MergeZipsCmd} -D -stripFile "**/*.class" $mergeZipsFlags $out $outDir/classes.dex.jar $in && ` +
			`rm -f "$outDir"/classes*.dex "$outDir/classes.dex.jar"`,
		Depfile: "${out}.d",
		Deps:    blueprint.DepsGCC,
		CommandDeps: []string{
			"${config.R8Cmd}",
			"${config.SoongZipCmd}",
			"${config.MergeZipsCmd}",
		},
	}, map[string]*remoteexec.REParams{
		"$r8Template": &remoteexec.REParams{
			Labels:          map[string]string{"type": "compile", "compiler": "r8"},
			Inputs:          []string{"$implicits", "${config.R8Jar}"},
			OutputFiles:     []string{"${outUsage}", "${outConfig}", "${outDict}", "${resourcesOutput}", "${outR8ArtProfile}"},
			ExecStrategy:    "${config.RER8ExecStrategy}",
			ToolchainInputs: []string{"${config.JavaCmd}"},
			Platform:        map[string]string{remoteexec.PoolKey: "${config.REJavaPool}"},
		},
		"$zipTemplate": &remoteexec.REParams{
			Labels:       map[string]string{"type": "tool", "name": "soong_zip"},
			Inputs:       []string{"${config.SoongZipCmd}", "$outDir"},
			OutputFiles:  []string{"$outDir/classes.dex.jar"},
			ExecStrategy: "${config.RER8ExecStrategy}",
			Platform:     map[string]string{remoteexec.PoolKey: "${config.REJavaPool}"},
		},
		"$zipUsageTemplate": &remoteexec.REParams{
			Labels:       map[string]string{"type": "tool", "name": "soong_zip"},
			Inputs:       []string{"${config.SoongZipCmd}", "${outUsage}"},
			OutputFiles:  []string{"${outUsageZip}"},
			ExecStrategy: "${config.RER8ExecStrategy}",
			Platform:     map[string]string{remoteexec.PoolKey: "${config.REJavaPool}"},
		},
	}, []string{"outDir", "outDict", "outConfig", "outUsage", "outUsageZip", "outUsageDir",
		"r8Flags", "zipFlags", "mergeZipsFlags", "resourcesOutput", "outR8ArtProfile"}, []string{"implicits"})

func (d *dexer) dexCommonFlags(ctx android.ModuleContext,
	dexParams *compileDexParams) (flags []string, deps android.Paths) {

	flags = d.dexProperties.Dxflags
	// Translate all the DX flags to D8 ones until all the build files have been migrated
	// to D8 flags. See: b/69377755
	flags = android.RemoveListFromList(flags,
		[]string{"--core-library", "--dex", "--multi-dex"})

	for _, f := range android.PathsForModuleSrc(ctx, d.dexProperties.Main_dex_rules) {
		flags = append(flags, "--main-dex-rules", f.String())
		deps = append(deps, f)
	}

	var requestReleaseMode bool
	requestReleaseMode, flags = android.RemoveFromList("--release", flags)

	if ctx.Config().Getenv("NO_OPTIMIZE_DX") != "" || ctx.Config().Getenv("GENERATE_DEX_DEBUG") != "" {
		flags = append(flags, "--debug")
		requestReleaseMode = false
	}

	// Don't strip out debug information for eng builds, unless the target
	// explicitly provided the `--release` build flag. This allows certain
	// test targets to remain optimized as part of eng test_suites builds.
	if requestReleaseMode {
		flags = append(flags, "--release")
	} else if ctx.Config().Eng() {
		flags = append(flags, "--debug")
	}

	// Supplying the platform build flag disables various features like API modeling and desugaring.
	// For targets with a stable min SDK version (i.e., when the min SDK is both explicitly specified
	// and managed+versioned), we suppress this flag to ensure portability.
	// Note: Targets with a min SDK kind of core_platform (e.g., framework.jar) or unspecified (e.g.,
	// services.jar), are not classified as stable, which is WAI.
	// TODO(b/232073181): Expand to additional min SDK cases after validation.
	var addAndroidPlatformBuildFlag = false
	if !dexParams.sdkVersion.Stable() {
		addAndroidPlatformBuildFlag = true
	}

	effectiveVersion, err := dexParams.minSdkVersion.EffectiveVersion(ctx)
	if err != nil {
		ctx.PropertyErrorf("min_sdk_version", "%s", err)
	}
	if !Bool(d.dexProperties.No_dex_container) && effectiveVersion.FinalOrFutureInt() >= 36 && ctx.Config().UseDexV41() {
		// W is 36, but we have not bumped the SDK version yet, so check for both.
		if ctx.Config().PlatformSdkVersion().FinalInt() >= 36 ||
			ctx.Config().PlatformSdkCodename() == "Baklava" {
			flags = append([]string{"-JDcom.android.tools.r8.dexContainerExperiment"}, flags...)
		}
	}

	// If the specified SDK level is 10000, then configure the compiler to use the
	// current platform SDK level and to compile the build as a platform build.
	var minApiFlagValue = effectiveVersion.FinalOrFutureInt()
	if minApiFlagValue == 10000 {
		minApiFlagValue = ctx.Config().PlatformSdkVersion().FinalInt()
		addAndroidPlatformBuildFlag = true
	}
	flags = append(flags, "--min-api "+strconv.Itoa(minApiFlagValue))

	if addAndroidPlatformBuildFlag {
		flags = append(flags, "--android-platform-build")
	}
	return flags, deps
}

func (d *dexer) d8Flags(ctx android.ModuleContext, dexParams *compileDexParams) (d8Flags []string, d8Deps android.Paths, artProfileOutput *android.OutputPath) {
	flags := dexParams.flags
	d8Flags = append(d8Flags, flags.bootClasspath.FormRepeatedClassPath("--lib ")...)
	d8Flags = append(d8Flags, flags.dexClasspath.FormRepeatedClassPath("--lib ")...)

	d8Deps = append(d8Deps, flags.bootClasspath...)
	d8Deps = append(d8Deps, flags.dexClasspath...)

	if flags, deps, profileOutput := d.addArtProfile(ctx, dexParams); profileOutput != nil {
		d8Flags = append(d8Flags, flags...)
		d8Deps = append(d8Deps, deps...)
		artProfileOutput = profileOutput
	}

	return d8Flags, d8Deps, artProfileOutput
}

func (d *dexer) r8Flags(ctx android.ModuleContext, dexParams *compileDexParams, debugMode bool) (r8Flags []string, r8Deps android.Paths, artProfileOutput *android.OutputPath) {
	flags := dexParams.flags
	opt := d.dexProperties.Optimize

	// When an app contains references to APIs that are not in the SDK specified by
	// its LOCAL_SDK_VERSION for example added by support library or by runtime
	// classes added by desugaring, we artifically raise the "SDK version" "linked" by
	// ProGuard, to
	// - suppress ProGuard warnings of referencing symbols unknown to the lower SDK version.
	// - prevent ProGuard stripping subclass in the support library that extends class added in the higher SDK version.
	// See b/20667396
	// TODO(b/360905238): Remove SdkSystemServer exception after resolving missing class references.
	if !dexParams.sdkVersion.Stable() || dexParams.sdkVersion.Kind == android.SdkSystemServer {
		var proguardRaiseDeps classpath
		ctx.VisitDirectDepsWithTag(proguardRaiseTag, func(m android.Module) {
			if dep, ok := android.OtherModuleProvider(ctx, m, JavaInfoProvider); ok {
				proguardRaiseDeps = append(proguardRaiseDeps, dep.RepackagedHeaderJars...)
			}
		})
		r8Flags = append(r8Flags, proguardRaiseDeps.FormJavaClassPath("-libraryjars"))
		r8Deps = append(r8Deps, proguardRaiseDeps...)
	}

	r8Flags = append(r8Flags, flags.bootClasspath.FormJavaClassPath("-libraryjars"))
	r8Deps = append(r8Deps, flags.bootClasspath...)
	r8Flags = append(r8Flags, flags.dexClasspath.FormJavaClassPath("-libraryjars"))
	r8Deps = append(r8Deps, flags.dexClasspath...)

	transitiveStaticLibsLookupMap := map[android.Path]bool{}
	for _, jar := range d.transitiveStaticLibsHeaderJarsForR8.ToList() {
		transitiveStaticLibsLookupMap[jar] = true
	}
	transitiveHeaderJars := android.Paths{}
	for _, jar := range d.transitiveLibsHeaderJarsForR8.ToList() {
		if _, ok := transitiveStaticLibsLookupMap[jar]; ok {
			// don't include a lib if it is already packaged in the current JAR as a static lib
			continue
		}
		transitiveHeaderJars = append(transitiveHeaderJars, jar)
	}
	transitiveClasspath := classpath(transitiveHeaderJars)
	r8Flags = append(r8Flags, transitiveClasspath.FormJavaClassPath("-libraryjars"))
	r8Deps = append(r8Deps, transitiveClasspath...)

	flagFiles := android.Paths{
		android.PathForSource(ctx, "build/make/core/proguard.flags"),
	}

	flagFiles = append(flagFiles, d.extraProguardFlagsFiles...)
	// TODO(ccross): static android library proguard files

	flagFiles = append(flagFiles, android.PathsForModuleSrc(ctx, opt.Proguard_flags_files)...)

	flagFiles = android.FirstUniquePaths(flagFiles)

	r8Flags = append(r8Flags, android.JoinWithPrefix(flagFiles.Strings(), "-include "))
	r8Deps = append(r8Deps, flagFiles...)

	// TODO(b/70942988): This is included from build/make/core/proguard.flags
	r8Deps = append(r8Deps, android.PathForSource(ctx,
		"build/make/core/proguard_basic_keeps.flags"))

	r8Flags = append(r8Flags, opt.Proguard_flags...)

	if BoolDefault(opt.Proguard_compatibility, true) {
		r8Flags = append(r8Flags, "--force-proguard-compatibility")
	}

	// Avoid unnecessary stack frame noise by only injecting source map ids for non-debug
	// optimized or obfuscated targets.
	if (Bool(opt.Optimize) || Bool(opt.Obfuscate)) && !debugMode {
		// TODO(b/213833843): Allow configuration of the prefix via a build variable.
		var sourceFilePrefix = "go/retraceme "
		var sourceFileTemplate = "\"" + sourceFilePrefix + "%MAP_ID\""
		r8Flags = append(r8Flags, "--map-id-template", "%MAP_HASH")
		r8Flags = append(r8Flags, "--source-file-template", sourceFileTemplate)
	}

	// TODO(ccross): Don't shrink app instrumentation tests by default.
	if !Bool(opt.Shrink) {
		r8Flags = append(r8Flags, "-dontshrink")
	}

	if !Bool(opt.Optimize) {
		r8Flags = append(r8Flags, "-dontoptimize")
	}

	// TODO(ccross): error if obufscation + app instrumentation test.
	if !Bool(opt.Obfuscate) {
		r8Flags = append(r8Flags, "-dontobfuscate")
	}
	// TODO(ccross): if this is an instrumentation test of an obfuscated app, use the
	// dictionary of the app and move the app from libraryjars to injars.

	// TODO(b/180878971): missing classes should be added to the relevant builds.
	// TODO(b/229727645): do not use true as default for Android platform builds.
	if proptools.BoolDefault(opt.Ignore_warnings, true) {
		r8Flags = append(r8Flags, "-ignorewarnings")
	}

	// resourcesInput is empty when we don't use resource shrinking, if on, pass these to R8
	if d.resourcesInput.Valid() {
		r8Flags = append(r8Flags, "--resource-input", d.resourcesInput.Path().String())
		r8Deps = append(r8Deps, d.resourcesInput.Path())
		r8Flags = append(r8Flags, "--resource-output", d.resourcesOutput.Path().String())
		if d.dexProperties.optimizedResourceShrinkingEnabled(ctx) {
			r8Flags = append(r8Flags, "--optimized-resource-shrinking")
			if Bool(d.dexProperties.Optimize.Optimized_shrink_resources) {
				// Explicitly opted into optimized shrinking, no need for keeping R$id entries
				r8Flags = append(r8Flags, "--force-optimized-resource-shrinking")
			}
		}
	}

	if flags, deps, profileOutput := d.addArtProfile(ctx, dexParams); profileOutput != nil {
		r8Flags = append(r8Flags, flags...)
		r8Deps = append(r8Deps, deps...)
		artProfileOutput = profileOutput
	}

	return r8Flags, r8Deps, artProfileOutput
}

type compileDexParams struct {
	flags           javaBuilderFlags
	sdkVersion      android.SdkSpec
	minSdkVersion   android.ApiLevel
	classesJar      android.Path
	jarName         string
	artProfileInput *string
}

// Adds --art-profile to r8/d8 command.
// r8/d8 will output a generated profile file to match the optimized dex code.
func (d *dexer) addArtProfile(ctx android.ModuleContext, dexParams *compileDexParams) (flags []string, deps android.Paths, artProfileOutputPath *android.OutputPath) {
	if dexParams.artProfileInput == nil {
		return nil, nil, nil
	}
	artProfileInputPath := android.PathForModuleSrc(ctx, *dexParams.artProfileInput)
	artProfileOutputPathValue := android.PathForModuleOut(ctx, "profile.prof.txt").OutputPath
	artProfileOutputPath = &artProfileOutputPathValue
	flags = []string{
		"--art-profile",
		artProfileInputPath.String(),
		artProfileOutputPath.String(),
	}
	deps = append(deps, artProfileInputPath)
	return flags, deps, artProfileOutputPath

}

// Return the compiled dex jar and (optional) profile _after_ r8 optimization
func (d *dexer) compileDex(ctx android.ModuleContext, dexParams *compileDexParams) (android.Path, android.Path) {

	// Compile classes.jar into classes.dex and then javalib.jar
	javalibJar := android.PathForModuleOut(ctx, "dex", dexParams.jarName).OutputPath
	outDir := android.PathForModuleOut(ctx, "dex")

	zipFlags := "--ignore_missing_files"
	if proptools.Bool(d.dexProperties.Uncompress_dex) {
		zipFlags += " -L 0"
	}

	commonFlags, commonDeps := d.dexCommonFlags(ctx, dexParams)

	// Exclude kotlinc generated files when "exclude_kotlinc_generated_files" is set to true.
	mergeZipsFlags := ""
	if proptools.BoolDefault(d.dexProperties.Exclude_kotlinc_generated_files, false) {
		mergeZipsFlags = "-stripFile META-INF/*.kotlin_module -stripFile **/*.kotlin_builtins"
	}

	useR8 := d.effectiveOptimizeEnabled()
	var artProfileOutputPath *android.OutputPath
	if useR8 {
		proguardDictionary := android.PathForModuleOut(ctx, "proguard_dictionary")
		d.proguardDictionary = android.OptionalPathForPath(proguardDictionary)
		proguardConfiguration := android.PathForModuleOut(ctx, "proguard_configuration")
		d.proguardConfiguration = android.OptionalPathForPath(proguardConfiguration)
		proguardUsageDir := android.PathForModuleOut(ctx, "proguard_usage")
		proguardUsage := proguardUsageDir.Join(ctx, ctx.Namespace().Path,
			android.ModuleNameWithPossibleOverride(ctx), "unused.txt")
		proguardUsageZip := android.PathForModuleOut(ctx, "proguard_usage.zip")
		d.proguardUsageZip = android.OptionalPathForPath(proguardUsageZip)
		resourcesOutput := android.PathForModuleOut(ctx, "package-res-shrunken.apk")
		d.resourcesOutput = android.OptionalPathForPath(resourcesOutput)
		implicitOutputs := android.WritablePaths{
			proguardDictionary,
			proguardUsageZip,
			proguardConfiguration,
		}
		debugMode := android.InList("--debug", commonFlags)
		r8Flags, r8Deps, r8ArtProfileOutputPath := d.r8Flags(ctx, dexParams, debugMode)
		rule := r8
		args := map[string]string{
			"r8Flags":        strings.Join(append(commonFlags, r8Flags...), " "),
			"zipFlags":       zipFlags,
			"outDict":        proguardDictionary.String(),
			"outConfig":      proguardConfiguration.String(),
			"outUsageDir":    proguardUsageDir.String(),
			"outUsage":       proguardUsage.String(),
			"outUsageZip":    proguardUsageZip.String(),
			"outDir":         outDir.String(),
			"mergeZipsFlags": mergeZipsFlags,
		}
		if r8ArtProfileOutputPath != nil {
			artProfileOutputPath = r8ArtProfileOutputPath
			implicitOutputs = append(
				implicitOutputs,
				artProfileOutputPath,
			)
			// Add the implicit r8 Art profile output to args so that r8RE knows
			// about this implicit output
			args["outR8ArtProfile"] = artProfileOutputPath.String()
		}

		if ctx.Config().UseRBE() && ctx.Config().IsEnvTrue("RBE_R8") {
			rule = r8RE
			args["implicits"] = strings.Join(r8Deps.Strings(), ",")
		}
		if d.resourcesInput.Valid() {
			implicitOutputs = append(implicitOutputs, resourcesOutput)
			args["resourcesOutput"] = resourcesOutput.String()
		}
		ctx.Build(pctx, android.BuildParams{
			Rule:            rule,
			Description:     "r8",
			Output:          javalibJar,
			ImplicitOutputs: implicitOutputs,
			Input:           dexParams.classesJar,
			Implicits:       r8Deps,
			Args:            args,
		})
	} else {
		implicitOutputs := android.WritablePaths{}
		d8Flags, d8Deps, d8ArtProfileOutputPath := d.d8Flags(ctx, dexParams)
		if d8ArtProfileOutputPath != nil {
			artProfileOutputPath = d8ArtProfileOutputPath
			implicitOutputs = append(
				implicitOutputs,
				artProfileOutputPath,
			)
		}
		d8Deps = append(d8Deps, commonDeps...)
		rule := d8
		if ctx.Config().UseRBE() && ctx.Config().IsEnvTrue("RBE_D8") {
			rule = d8RE
		}
		ctx.Build(pctx, android.BuildParams{
			Rule:            rule,
			Description:     "d8",
			Output:          javalibJar,
			Input:           dexParams.classesJar,
			ImplicitOutputs: implicitOutputs,
			Implicits:       d8Deps,
			Args: map[string]string{
				"d8Flags":        strings.Join(append(commonFlags, d8Flags...), " "),
				"zipFlags":       zipFlags,
				"outDir":         outDir.String(),
				"mergeZipsFlags": mergeZipsFlags,
			},
		})
	}
	if proptools.Bool(d.dexProperties.Uncompress_dex) {
		alignedJavalibJar := android.PathForModuleOut(ctx, "aligned", dexParams.jarName).OutputPath
		TransformZipAlign(ctx, alignedJavalibJar, javalibJar, nil)
		javalibJar = alignedJavalibJar
	}

	return javalibJar, artProfileOutputPath
}
