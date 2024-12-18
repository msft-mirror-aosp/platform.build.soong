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

package cc

import (
	"sort"
	"strings"

	"android/soong/android"
)

func init() {
	// Use singleton type to gather all generated soong modules.
	android.RegisterParallelSingletonType("stublibraries", stubLibrariesSingleton)
}

func stubLibrariesSingleton() android.Singleton {
	return &stubLibraries{}
}

type stubLibraries struct {
	stubLibraries       []string
	vendorStubLibraries []string

	apiListCoverageXmlPaths []string
}

// Check if the module defines stub, or itself is stub
func IsStubTarget(m *Module) bool {
	return m.IsStubs() || m.HasStubsVariants()
}

// Get target file name to be installed from this module
func getInstalledFileName(ctx android.SingletonContext, m *Module) string {
	for _, ps := range android.OtherModuleProviderOrDefault(
		ctx, m.Module(), android.InstallFilesProvider).PackagingSpecs {
		if name := ps.FileName(); name != "" {
			return name
		}
	}
	return ""
}

func (s *stubLibraries) GenerateBuildActions(ctx android.SingletonContext) {
	// Visit all generated soong modules and store stub library file names.
	stubLibraryMap := make(map[string]bool)
	vendorStubLibraryMap := make(map[string]bool)
	ctx.VisitAllModules(func(module android.Module) {
		if m, ok := module.(*Module); ok {
			if IsStubTarget(m) {
				if name := getInstalledFileName(ctx, m); name != "" {
					stubLibraryMap[name] = true
					if m.InVendor() {
						vendorStubLibraryMap[name] = true
					}
				}
			}
			if m.library != nil && android.IsModulePreferred(m) {
				if p := m.library.getAPIListCoverageXMLPath().String(); p != "" {
					s.apiListCoverageXmlPaths = append(s.apiListCoverageXmlPaths, p)
				}
			}
		}
	})
	s.stubLibraries = android.SortedKeys(stubLibraryMap)
	s.vendorStubLibraries = android.SortedKeys(vendorStubLibraryMap)

	android.WriteFileRule(ctx, StubLibrariesFile(ctx), strings.Join(s.stubLibraries, " "))
}

func StubLibrariesFile(ctx android.PathContext) android.WritablePath {
	return android.PathForIntermediates(ctx, "stub_libraries.txt")
}

func (s *stubLibraries) MakeVars(ctx android.MakeVarsContext) {
	// Convert stub library file names into Makefile variable.
	ctx.Strict("STUB_LIBRARIES", strings.Join(s.stubLibraries, " "))
	ctx.Strict("SOONG_STUB_VENDOR_LIBRARIES", strings.Join(s.vendorStubLibraries, " "))

	// Export the list of API XML files to Make.
	sort.Strings(s.apiListCoverageXmlPaths)
	ctx.Strict("SOONG_CC_API_XML", strings.Join(s.apiListCoverageXmlPaths, " "))
}
