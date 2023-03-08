// Copyright 2020 The Android Open Source Project
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

package rust

import (
	"path/filepath"

	"android/soong/android"
	"android/soong/cc"
	"android/soong/fuzz"
	"android/soong/rust/config"
)

func init() {
	android.RegisterModuleType("rust_fuzz", RustFuzzFactory)
}

type fuzzDecorator struct {
	*binaryDecorator

	fuzzPackagedModule  fuzz.FuzzPackagedModule
	sharedLibraries     android.Paths
	installedSharedDeps []string
}

var _ compiler = (*fuzzDecorator)(nil)

// rust_binary produces a binary that is runnable on a device.
func RustFuzzFactory() android.Module {
	module, _ := NewRustFuzz(android.HostAndDeviceSupported)
	return module.Init()
}

func NewRustFuzz(hod android.HostOrDeviceSupported) (*Module, *fuzzDecorator) {
	module, binary := NewRustBinary(hod)
	fuzz := &fuzzDecorator{
		binaryDecorator: binary,
	}

	// Change the defaults for the binaryDecorator's baseCompiler
	fuzz.binaryDecorator.baseCompiler.dir = "fuzz"
	fuzz.binaryDecorator.baseCompiler.dir64 = "fuzz"
	fuzz.binaryDecorator.baseCompiler.location = InstallInData
	module.sanitize.SetSanitizer(cc.Fuzzer, true)
	module.compiler = fuzz
	return module, fuzz
}

func (fuzzer *fuzzDecorator) compilerFlags(ctx ModuleContext, flags Flags) Flags {
	flags = fuzzer.binaryDecorator.compilerFlags(ctx, flags)

	// `../lib` for installed fuzz targets (both host and device), and `./lib` for fuzz target packages.
	flags.LinkFlags = append(flags.LinkFlags, `-Wl,-rpath,\$$ORIGIN/lib`)

	if ctx.InstallInVendor() {
		flags.LinkFlags = append(flags.LinkFlags, `-Wl,-rpath,\$$ORIGIN/../../lib`)
	} else {
		flags.LinkFlags = append(flags.LinkFlags, `-Wl,-rpath,\$$ORIGIN/../lib`)

	}
	return flags
}

func (fuzzer *fuzzDecorator) compilerDeps(ctx DepsContext, deps Deps) Deps {
	if libFuzzerRuntimeLibrary := config.LibFuzzerRuntimeLibrary(ctx.toolchain()); libFuzzerRuntimeLibrary != "" {
		deps.StaticLibs = append(deps.StaticLibs, libFuzzerRuntimeLibrary)
	}
	deps.SharedLibs = append(deps.SharedLibs, "libc++")
	deps.Rlibs = append(deps.Rlibs, "liblibfuzzer_sys")

	deps = fuzzer.binaryDecorator.compilerDeps(ctx, deps)

	return deps
}

func (fuzzer *fuzzDecorator) compilerProps() []interface{} {
	return append(fuzzer.binaryDecorator.compilerProps(),
		&fuzzer.fuzzPackagedModule.FuzzProperties)
}

func (fuzzer *fuzzDecorator) compile(ctx ModuleContext, flags Flags, deps PathDeps) buildOutput {

	out := fuzzer.binaryDecorator.compile(ctx, flags, deps)

	return out
}

func (fuzzer *fuzzDecorator) stdLinkage(ctx *depsContext) RustLinkage {
	return RlibLinkage
}

func (fuzzer *fuzzDecorator) autoDep(ctx android.BottomUpMutatorContext) autoDep {
	return rlibAutoDep
}

func (fuzz *fuzzDecorator) install(ctx ModuleContext) {
	fuzz.binaryDecorator.baseCompiler.dir = filepath.Join(
		"fuzz", ctx.Target().Arch.ArchType.String(), ctx.ModuleName())
	fuzz.binaryDecorator.baseCompiler.dir64 = filepath.Join(
		"fuzz", ctx.Target().Arch.ArchType.String(), ctx.ModuleName())
	fuzz.binaryDecorator.baseCompiler.install(ctx)

	fuzz.fuzzPackagedModule = cc.PackageFuzzModule(ctx, fuzz.fuzzPackagedModule, pctx)

	installBase := "fuzz"

	// Grab the list of required shared libraries.
	fuzz.sharedLibraries, _ = cc.CollectAllSharedDependencies(ctx)

	for _, lib := range fuzz.sharedLibraries {
		fuzz.installedSharedDeps = append(fuzz.installedSharedDeps,
			cc.SharedLibraryInstallLocation(
				lib, ctx.Host(), installBase, ctx.Arch().ArchType.String()))

		// Also add the dependency on the shared library symbols dir.
		if !ctx.Host() {
			fuzz.installedSharedDeps = append(fuzz.installedSharedDeps,
				cc.SharedLibrarySymbolsInstallLocation(lib, installBase, ctx.Arch().ArchType.String()))
		}
	}
}
