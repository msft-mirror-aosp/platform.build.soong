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

package bazel

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/google/blueprint/proptools"
)

type artifactId int
type depsetId int
type pathFragmentId int

// artifact contains relevant portions of Bazel's aquery proto, Artifact.
// Represents a single artifact, whether it's a source file or a derived output file.
type artifact struct {
	Id             artifactId
	PathFragmentId pathFragmentId
}

type pathFragment struct {
	Id       pathFragmentId
	Label    string
	ParentId pathFragmentId
}

// KeyValuePair represents Bazel's aquery proto, KeyValuePair.
type KeyValuePair struct {
	Key   string
	Value string
}

// AqueryDepset is a depset definition from Bazel's aquery response. This is
// akin to the `depSetOfFiles` in the response proto, except:
//   - direct artifacts are enumerated by full path instead of by ID
//   - it has a hash of the depset contents, instead of an int ID (for determinism)
//
// A depset is a data structure for efficient transitive handling of artifact
// paths. A single depset consists of one or more artifact paths and one or
// more "child" depsets.
type AqueryDepset struct {
	ContentHash            string
	DirectArtifacts        []string
	TransitiveDepSetHashes []string
}

// depSetOfFiles contains relevant portions of Bazel's aquery proto, DepSetOfFiles.
// Represents a data structure containing one or more files. Depsets in Bazel are an efficient
// data structure for storing large numbers of file paths.
type depSetOfFiles struct {
	Id                  depsetId
	DirectArtifactIds   []artifactId
	TransitiveDepSetIds []depsetId
}

// action contains relevant portions of Bazel's aquery proto, Action.
// Represents a single command line invocation in the Bazel build graph.
type action struct {
	Arguments            []string
	EnvironmentVariables []KeyValuePair
	InputDepSetIds       []depsetId
	Mnemonic             string
	OutputIds            []artifactId
	TemplateContent      string
	Substitutions        []KeyValuePair
	FileContents         string
}

// actionGraphContainer contains relevant portions of Bazel's aquery proto, ActionGraphContainer.
// An aquery response from Bazel contains a single ActionGraphContainer proto.
type actionGraphContainer struct {
	Artifacts     []artifact
	Actions       []action
	DepSetOfFiles []depSetOfFiles
	PathFragments []pathFragment
}

// BuildStatement contains information to register a build statement corresponding (one to one)
// with a Bazel action from Bazel's action graph.
type BuildStatement struct {
	Command      string
	Depfile      *string
	OutputPaths  []string
	SymlinkPaths []string
	Env          []KeyValuePair
	Mnemonic     string

	// Inputs of this build statement, either as unexpanded depsets or expanded
	// input paths. There should be no overlap between these fields; an input
	// path should either be included as part of an unexpanded depset or a raw
	// input path string, but not both.
	InputDepsetHashes []string
	InputPaths        []string
	FileContents      string
}

// A helper type for aquery processing which facilitates retrieval of path IDs from their
// less readable Bazel structures (depset and path fragment).
type aqueryArtifactHandler struct {
	// Switches to true if any depset contains only `bazelToolsDependencySentinel`
	bazelToolsDependencySentinelNeeded bool
	// Maps depset id to AqueryDepset, a representation of depset which is
	// post-processed for middleman artifact handling, unhandled artifact
	// dropping, content hashing, etc.
	depsetIdToAqueryDepset map[depsetId]AqueryDepset
	// Maps content hash to AqueryDepset.
	depsetHashToAqueryDepset map[string]AqueryDepset

	// depsetIdToArtifactIdsCache is a memoization of depset flattening, because flattening
	// may be an expensive operation.
	depsetHashToArtifactPathsCache map[string][]string
	// Maps artifact ids to fully expanded paths.
	artifactIdToPath map[artifactId]string
}

// The tokens should be substituted with the value specified here, instead of the
// one returned in 'substitutions' of TemplateExpand action.
var templateActionOverriddenTokens = map[string]string{
	// Uses "python3" for %python_binary% instead of the value returned by aquery
	// which is "py3wrapper.sh". See removePy3wrapperScript.
	"%python_binary%": "python3",
}

// The file name of py3wrapper.sh, which is used by py_binary targets.
const py3wrapperFileName = "/py3wrapper.sh"

// A file to be put into depsets that are otherwise empty
const bazelToolsDependencySentinel = "BAZEL_TOOLS_DEPENDENCY_SENTINEL"

func indexBy[K comparable, V any](values []V, keyFn func(v V) K) map[K]V {
	m := map[K]V{}
	for _, v := range values {
		m[keyFn(v)] = v
	}
	return m
}

func newAqueryHandler(aqueryResult actionGraphContainer) (*aqueryArtifactHandler, error) {
	pathFragments := indexBy(aqueryResult.PathFragments, func(pf pathFragment) pathFragmentId {
		return pf.Id
	})

	artifactIdToPath := map[artifactId]string{}
	for _, artifact := range aqueryResult.Artifacts {
		artifactPath, err := expandPathFragment(artifact.PathFragmentId, pathFragments)
		if err != nil {
			return nil, err
		}
		artifactIdToPath[artifact.Id] = artifactPath
	}

	// Map middleman artifact ContentHash to input artifact depset ID.
	// Middleman artifacts are treated as "substitute" artifacts for mixed builds. For example,
	// if we find a middleman action which has inputs [foo, bar], and output [baz_middleman], then,
	// for each other action which has input [baz_middleman], we add [foo, bar] to the inputs for
	// that action instead.
	middlemanIdToDepsetIds := map[artifactId][]depsetId{}
	for _, actionEntry := range aqueryResult.Actions {
		if actionEntry.Mnemonic == "Middleman" {
			for _, outputId := range actionEntry.OutputIds {
				middlemanIdToDepsetIds[outputId] = actionEntry.InputDepSetIds
			}
		}
	}

	depsetIdToDepset := indexBy(aqueryResult.DepSetOfFiles, func(d depSetOfFiles) depsetId {
		return d.Id
	})

	aqueryHandler := aqueryArtifactHandler{
		depsetIdToAqueryDepset:         map[depsetId]AqueryDepset{},
		depsetHashToAqueryDepset:       map[string]AqueryDepset{},
		depsetHashToArtifactPathsCache: map[string][]string{},
		artifactIdToPath:               artifactIdToPath,
	}

	// Validate and adjust aqueryResult.DepSetOfFiles values.
	for _, depset := range aqueryResult.DepSetOfFiles {
		_, err := aqueryHandler.populateDepsetMaps(depset, middlemanIdToDepsetIds, depsetIdToDepset)
		if err != nil {
			return nil, err
		}
	}

	return &aqueryHandler, nil
}

// Ensures that the handler's depsetIdToAqueryDepset map contains an entry for the given
// depset.
func (a *aqueryArtifactHandler) populateDepsetMaps(depset depSetOfFiles, middlemanIdToDepsetIds map[artifactId][]depsetId, depsetIdToDepset map[depsetId]depSetOfFiles) (AqueryDepset, error) {
	if aqueryDepset, containsDepset := a.depsetIdToAqueryDepset[depset.Id]; containsDepset {
		return aqueryDepset, nil
	}
	transitiveDepsetIds := depset.TransitiveDepSetIds
	var directArtifactPaths []string
	for _, artifactId := range depset.DirectArtifactIds {
		path, pathExists := a.artifactIdToPath[artifactId]
		if !pathExists {
			return AqueryDepset{}, fmt.Errorf("undefined input artifactId %d", artifactId)
		}
		// Filter out any inputs which are universally dropped, and swap middleman
		// artifacts with their corresponding depsets.
		if depsetsToUse, isMiddleman := middlemanIdToDepsetIds[artifactId]; isMiddleman {
			// Swap middleman artifacts with their corresponding depsets and drop the middleman artifacts.
			transitiveDepsetIds = append(transitiveDepsetIds, depsetsToUse...)
		} else if strings.HasSuffix(path, py3wrapperFileName) ||
			strings.HasPrefix(path, "../bazel_tools") {
			// Drop these artifacts.
			// See go/python-binary-host-mixed-build for more details.
			// 1) Drop py3wrapper.sh, just use python binary, the launcher script generated by the
			// TemplateExpandAction handles everything necessary to launch a Pythin application.
			// 2) ../bazel_tools: they have MODIFY timestamp 10years in the future and would cause the
			// containing depset to always be considered newer than their outputs.
		} else {
			directArtifactPaths = append(directArtifactPaths, path)
		}
	}

	var childDepsetHashes []string
	for _, childDepsetId := range transitiveDepsetIds {
		childDepset, exists := depsetIdToDepset[childDepsetId]
		if !exists {
			return AqueryDepset{}, fmt.Errorf("undefined input depsetId %d (referenced by depsetId %d)", childDepsetId, depset.Id)
		}
		childAqueryDepset, err := a.populateDepsetMaps(childDepset, middlemanIdToDepsetIds, depsetIdToDepset)
		if err != nil {
			return AqueryDepset{}, err
		}
		childDepsetHashes = append(childDepsetHashes, childAqueryDepset.ContentHash)
	}
	if len(directArtifactPaths) == 0 && len(childDepsetHashes) == 0 {
		// We could omit this depset altogether but that requires cleanup on
		// transitive dependents.
		// As a simpler alternative, we use this sentinel file as a dependency.
		directArtifactPaths = append(directArtifactPaths, bazelToolsDependencySentinel)
		a.bazelToolsDependencySentinelNeeded = true
	}
	aqueryDepset := AqueryDepset{
		ContentHash:            depsetContentHash(directArtifactPaths, childDepsetHashes),
		DirectArtifacts:        directArtifactPaths,
		TransitiveDepSetHashes: childDepsetHashes,
	}
	a.depsetIdToAqueryDepset[depset.Id] = aqueryDepset
	a.depsetHashToAqueryDepset[aqueryDepset.ContentHash] = aqueryDepset
	return aqueryDepset, nil
}

// getInputPaths flattens the depsets of the given IDs and returns all transitive
// input paths contained in these depsets.
// This is a potentially expensive operation, and should not be invoked except
// for actions which need specialized input handling.
func (a *aqueryArtifactHandler) getInputPaths(depsetIds []depsetId) ([]string, error) {
	var inputPaths []string

	for _, inputDepSetId := range depsetIds {
		depset := a.depsetIdToAqueryDepset[inputDepSetId]
		inputArtifacts, err := a.artifactPathsFromDepsetHash(depset.ContentHash)
		if err != nil {
			return nil, err
		}
		for _, inputPath := range inputArtifacts {
			inputPaths = append(inputPaths, inputPath)
		}
	}

	return inputPaths, nil
}

func (a *aqueryArtifactHandler) artifactPathsFromDepsetHash(depsetHash string) ([]string, error) {
	if result, exists := a.depsetHashToArtifactPathsCache[depsetHash]; exists {
		return result, nil
	}
	if depset, exists := a.depsetHashToAqueryDepset[depsetHash]; exists {
		result := depset.DirectArtifacts
		for _, childHash := range depset.TransitiveDepSetHashes {
			childArtifactIds, err := a.artifactPathsFromDepsetHash(childHash)
			if err != nil {
				return nil, err
			}
			result = append(result, childArtifactIds...)
		}
		a.depsetHashToArtifactPathsCache[depsetHash] = result
		return result, nil
	} else {
		return nil, fmt.Errorf("undefined input depset hash %s", depsetHash)
	}
}

// AqueryBuildStatements returns a slice of BuildStatements and a slice of AqueryDepset
// which should be registered (and output to a ninja file) to correspond with Bazel's
// action graph, as described by the given action graph json proto.
// BuildStatements are one-to-one with actions in the given action graph, and AqueryDepsets
// are one-to-one with Bazel's depSetOfFiles objects.
func AqueryBuildStatements(aqueryJsonProto []byte) ([]BuildStatement, []AqueryDepset, error) {
	var aqueryResult actionGraphContainer
	err := json.Unmarshal(aqueryJsonProto, &aqueryResult)
	if err != nil {
		return nil, nil, err
	}
	aqueryHandler, err := newAqueryHandler(aqueryResult)
	if err != nil {
		return nil, nil, err
	}

	var buildStatements []BuildStatement
	if aqueryHandler.bazelToolsDependencySentinelNeeded {
		buildStatements = append(buildStatements, BuildStatement{
			Command:     fmt.Sprintf("touch '%s'", bazelToolsDependencySentinel),
			OutputPaths: []string{bazelToolsDependencySentinel},
			Mnemonic:    bazelToolsDependencySentinel,
		})
	}

	for _, actionEntry := range aqueryResult.Actions {
		if shouldSkipAction(actionEntry) {
			continue
		}

		var buildStatement BuildStatement
		if actionEntry.isSymlinkAction() {
			buildStatement, err = aqueryHandler.symlinkActionBuildStatement(actionEntry)
		} else if actionEntry.isTemplateExpandAction() && len(actionEntry.Arguments) < 1 {
			buildStatement, err = aqueryHandler.templateExpandActionBuildStatement(actionEntry)
		} else if actionEntry.isFileWriteAction() {
			buildStatement, err = aqueryHandler.fileWriteActionBuildStatement(actionEntry)
		} else if actionEntry.isSymlinkTreeAction() {
			buildStatement, err = aqueryHandler.symlinkTreeActionBuildStatement(actionEntry)
		} else if len(actionEntry.Arguments) < 1 {
			return nil, nil, fmt.Errorf("received action with no command: [%s]", actionEntry.Mnemonic)
		} else {
			buildStatement, err = aqueryHandler.normalActionBuildStatement(actionEntry)
		}

		if err != nil {
			return nil, nil, err
		}
		buildStatements = append(buildStatements, buildStatement)
	}

	depsetsByHash := map[string]AqueryDepset{}
	var depsets []AqueryDepset
	for _, aqueryDepset := range aqueryHandler.depsetIdToAqueryDepset {
		if prevEntry, hasKey := depsetsByHash[aqueryDepset.ContentHash]; hasKey {
			// Two depsets collide on hash. Ensure that their contents are identical.
			if !reflect.DeepEqual(aqueryDepset, prevEntry) {
				return nil, nil, fmt.Errorf("two different depsets have the same hash: %v, %v", prevEntry, aqueryDepset)
			}
		} else {
			depsetsByHash[aqueryDepset.ContentHash] = aqueryDepset
			depsets = append(depsets, aqueryDepset)
		}
	}

	// Build Statements and depsets must be sorted by their content hash to
	// preserve determinism between builds (this will result in consistent ninja file
	// output). Note they are not sorted by their original IDs nor their Bazel ordering,
	// as Bazel gives nondeterministic ordering / identifiers in aquery responses.
	sort.Slice(buildStatements, func(i, j int) bool {
		// For build statements, compare output lists. In Bazel, each output file
		// may only have one action which generates it, so this will provide
		// a deterministic ordering.
		outputs_i := buildStatements[i].OutputPaths
		outputs_j := buildStatements[j].OutputPaths
		if len(outputs_i) != len(outputs_j) {
			return len(outputs_i) < len(outputs_j)
		}
		if len(outputs_i) == 0 {
			// No outputs for these actions, so compare commands.
			return buildStatements[i].Command < buildStatements[j].Command
		}
		// There may be multiple outputs, but the output ordering is deterministic.
		return outputs_i[0] < outputs_j[0]
	})
	sort.Slice(depsets, func(i, j int) bool {
		return depsets[i].ContentHash < depsets[j].ContentHash
	})
	return buildStatements, depsets, nil
}

// depsetContentHash computes and returns a SHA256 checksum of the contents of
// the given depset. This content hash may serve as the depset's identifier.
// Using a content hash for an identifier is superior for determinism. (For example,
// using an integer identifier which depends on the order in which the depsets are
// created would result in nondeterministic depset IDs.)
func depsetContentHash(directPaths []string, transitiveDepsetHashes []string) string {
	h := sha256.New()
	// Use newline as delimiter, as paths cannot contain newline.
	h.Write([]byte(strings.Join(directPaths, "\n")))
	h.Write([]byte(strings.Join(transitiveDepsetHashes, "")))
	fullHash := base64.RawURLEncoding.EncodeToString(h.Sum(nil))
	return fullHash
}

func (a *aqueryArtifactHandler) depsetContentHashes(inputDepsetIds []depsetId) ([]string, error) {
	var hashes []string
	for _, depsetId := range inputDepsetIds {
		if aqueryDepset, exists := a.depsetIdToAqueryDepset[depsetId]; !exists {
			return nil, fmt.Errorf("undefined input depsetId %d", depsetId)
		} else {
			hashes = append(hashes, aqueryDepset.ContentHash)
		}
	}
	return hashes, nil
}

func (a *aqueryArtifactHandler) normalActionBuildStatement(actionEntry action) (BuildStatement, error) {
	command := strings.Join(proptools.ShellEscapeListIncludingSpaces(actionEntry.Arguments), " ")
	inputDepsetHashes, err := a.depsetContentHashes(actionEntry.InputDepSetIds)
	if err != nil {
		return BuildStatement{}, err
	}
	outputPaths, depfile, err := a.getOutputPaths(actionEntry)
	if err != nil {
		return BuildStatement{}, err
	}

	buildStatement := BuildStatement{
		Command:           command,
		Depfile:           depfile,
		OutputPaths:       outputPaths,
		InputDepsetHashes: inputDepsetHashes,
		Env:               actionEntry.EnvironmentVariables,
		Mnemonic:          actionEntry.Mnemonic,
	}
	return buildStatement, nil
}

func (a *aqueryArtifactHandler) templateExpandActionBuildStatement(actionEntry action) (BuildStatement, error) {
	outputPaths, depfile, err := a.getOutputPaths(actionEntry)
	if err != nil {
		return BuildStatement{}, err
	}
	if len(outputPaths) != 1 {
		return BuildStatement{}, fmt.Errorf("Expect 1 output to template expand action, got: output %q", outputPaths)
	}
	expandedTemplateContent := expandTemplateContent(actionEntry)
	// The expandedTemplateContent is escaped for being used in double quotes and shell unescape,
	// and the new line characters (\n) are also changed to \\n which avoids some Ninja escape on \n, which might
	// change \n to space and mess up the format of Python programs.
	// sed is used to convert \\n back to \n before saving to output file.
	// See go/python-binary-host-mixed-build for more details.
	command := fmt.Sprintf(`/bin/bash -c 'echo "%[1]s" | sed "s/\\\\n/\\n/g" > %[2]s && chmod a+x %[2]s'`,
		escapeCommandlineArgument(expandedTemplateContent), outputPaths[0])
	inputDepsetHashes, err := a.depsetContentHashes(actionEntry.InputDepSetIds)
	if err != nil {
		return BuildStatement{}, err
	}

	buildStatement := BuildStatement{
		Command:           command,
		Depfile:           depfile,
		OutputPaths:       outputPaths,
		InputDepsetHashes: inputDepsetHashes,
		Env:               actionEntry.EnvironmentVariables,
		Mnemonic:          actionEntry.Mnemonic,
	}
	return buildStatement, nil
}

func (a *aqueryArtifactHandler) fileWriteActionBuildStatement(actionEntry action) (BuildStatement, error) {
	outputPaths, _, err := a.getOutputPaths(actionEntry)
	var depsetHashes []string
	if err == nil {
		depsetHashes, err = a.depsetContentHashes(actionEntry.InputDepSetIds)
	}
	if err != nil {
		return BuildStatement{}, err
	}
	return BuildStatement{
		Depfile:           nil,
		OutputPaths:       outputPaths,
		Env:               actionEntry.EnvironmentVariables,
		Mnemonic:          actionEntry.Mnemonic,
		InputDepsetHashes: depsetHashes,
		FileContents:      actionEntry.FileContents,
	}, nil
}

func (a *aqueryArtifactHandler) symlinkTreeActionBuildStatement(actionEntry action) (BuildStatement, error) {
	outputPaths, _, err := a.getOutputPaths(actionEntry)
	if err != nil {
		return BuildStatement{}, err
	}
	inputPaths, err := a.getInputPaths(actionEntry.InputDepSetIds)
	if err != nil {
		return BuildStatement{}, err
	}
	if len(inputPaths) != 1 || len(outputPaths) != 1 {
		return BuildStatement{}, fmt.Errorf("Expect 1 input and 1 output to symlink action, got: input %q, output %q", inputPaths, outputPaths)
	}
	// The actual command is generated in bazelSingleton.GenerateBuildActions
	return BuildStatement{
		Depfile:     nil,
		OutputPaths: outputPaths,
		Env:         actionEntry.EnvironmentVariables,
		Mnemonic:    actionEntry.Mnemonic,
		InputPaths:  inputPaths,
	}, nil
}

func (a *aqueryArtifactHandler) symlinkActionBuildStatement(actionEntry action) (BuildStatement, error) {
	outputPaths, depfile, err := a.getOutputPaths(actionEntry)
	if err != nil {
		return BuildStatement{}, err
	}

	inputPaths, err := a.getInputPaths(actionEntry.InputDepSetIds)
	if err != nil {
		return BuildStatement{}, err
	}
	if len(inputPaths) != 1 || len(outputPaths) != 1 {
		return BuildStatement{}, fmt.Errorf("Expect 1 input and 1 output to symlink action, got: input %q, output %q", inputPaths, outputPaths)
	}
	out := outputPaths[0]
	outDir := proptools.ShellEscapeIncludingSpaces(filepath.Dir(out))
	out = proptools.ShellEscapeIncludingSpaces(out)
	in := filepath.Join("$PWD", proptools.ShellEscapeIncludingSpaces(inputPaths[0]))
	// Use absolute paths, because some soong actions don't play well with relative paths (for example, `cp -d`).
	command := fmt.Sprintf("mkdir -p %[1]s && rm -f %[2]s && ln -sf %[3]s %[2]s", outDir, out, in)
	symlinkPaths := outputPaths[:]

	buildStatement := BuildStatement{
		Command:      command,
		Depfile:      depfile,
		OutputPaths:  outputPaths,
		InputPaths:   inputPaths,
		Env:          actionEntry.EnvironmentVariables,
		Mnemonic:     actionEntry.Mnemonic,
		SymlinkPaths: symlinkPaths,
	}
	return buildStatement, nil
}

func (a *aqueryArtifactHandler) getOutputPaths(actionEntry action) (outputPaths []string, depfile *string, err error) {
	for _, outputId := range actionEntry.OutputIds {
		outputPath, exists := a.artifactIdToPath[outputId]
		if !exists {
			err = fmt.Errorf("undefined outputId %d", outputId)
			return
		}
		ext := filepath.Ext(outputPath)
		if ext == ".d" {
			if depfile != nil {
				err = fmt.Errorf("found multiple potential depfiles %q, %q", *depfile, outputPath)
				return
			} else {
				depfile = &outputPath
			}
		} else {
			outputPaths = append(outputPaths, outputPath)
		}
	}
	return
}

// expandTemplateContent substitutes the tokens in a template.
func expandTemplateContent(actionEntry action) string {
	var replacerString []string
	for _, pair := range actionEntry.Substitutions {
		value := pair.Value
		if val, ok := templateActionOverriddenTokens[pair.Key]; ok {
			value = val
		}
		replacerString = append(replacerString, pair.Key, value)
	}
	replacer := strings.NewReplacer(replacerString...)
	return replacer.Replace(actionEntry.TemplateContent)
}

func escapeCommandlineArgument(str string) string {
	// \->\\, $->\$, `->\`, "->\", \n->\\n, '->'"'"'
	replacer := strings.NewReplacer(
		`\`, `\\`,
		`$`, `\$`,
		"`", "\\`",
		`"`, `\"`,
		"\n", "\\n",
		`'`, `'"'"'`,
	)
	return replacer.Replace(str)
}

func (a action) isSymlinkAction() bool {
	return a.Mnemonic == "Symlink" || a.Mnemonic == "SolibSymlink" || a.Mnemonic == "ExecutableSymlink"
}

func (a action) isTemplateExpandAction() bool {
	return a.Mnemonic == "TemplateExpand"
}

func (a action) isFileWriteAction() bool {
	return a.Mnemonic == "FileWrite" || a.Mnemonic == "SourceSymlinkManifest"
}

func (a action) isSymlinkTreeAction() bool {
	return a.Mnemonic == "SymlinkTree"
}

func shouldSkipAction(a action) bool {
	// Middleman actions are not handled like other actions; they are handled separately as a
	// preparatory step so that their inputs may be relayed to actions depending on middleman
	// artifacts.
	if a.Mnemonic == "Middleman" {
		return true
	}
	// PythonZipper is bogus action returned by aquery, ignore it (b/236198693)
	if a.Mnemonic == "PythonZipper" {
		return true
	}
	// Skip "Fail" actions, which are placeholder actions designed to always fail.
	if a.Mnemonic == "Fail" {
		return true
	}
	if a.Mnemonic == "BaselineCoverage" {
		return true
	}
	return false
}

func expandPathFragment(id pathFragmentId, pathFragmentsMap map[pathFragmentId]pathFragment) (string, error) {
	var labels []string
	currId := id
	// Only positive IDs are valid for path fragments. An ID of zero indicates a terminal node.
	for currId > 0 {
		currFragment, ok := pathFragmentsMap[currId]
		if !ok {
			return "", fmt.Errorf("undefined path fragment id %d", currId)
		}
		labels = append([]string{currFragment.Label}, labels...)
		if currId == currFragment.ParentId {
			return "", fmt.Errorf("fragment cannot refer to itself as parent %#v", currFragment)
		}
		currId = currFragment.ParentId
	}
	return filepath.Join(labels...), nil
}
