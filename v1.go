package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type CompileCommandGenerator struct {
	compilerPathCache map[string]string
}

func (g *CompileCommandGenerator) RemapCompilerPath(path string) (mapped string, err error) {
	if mapped, ok := g.compilerPathCache[path]; ok {
		return mapped, nil
	}
	// Cache a successful result
	defer func() {
		if err == nil {
			g.compilerPathCache[path] = mapped
		}
	}()
	// dir, file := filepath.Split(path)
	_, file := filepath.Split(path)
	if file != "cc_wrapper.sh" {
		mapped = path
		return mapped, err
	}
	if cxx := os.Getenv("CXX"); len(cxx) > 0 {
		mapped = cxx
	} else if cxx := os.Getenv("cxx"); len(cxx) > 0 {
		mapped = cxx
	} else {
		mapped = path
	}
	return mapped, err
}

func (g *CompileCommandGenerator) GetCppCommandForFiles(action AqueryAction) (src string, args []string, err error) {
	if len(action.Arguments) == 0 {
		err = errors.New("empty arguments for compiler action")
		return src, args, err
	}
	args = make([]string, 0, len(action.Arguments)+2)
	// TODO(bazel): We might need to preprocess the compiler location if it's a llvm_toolchain one
	// because in the current form, the compiler headers are in the wrong place.
	compilerPath, err := g.RemapCompilerPath(action.Arguments[0])
	if err != nil {
		return src, args, err
	}

	dedup := make(map[string]struct{}, len(action.Arguments)+2)

	args = append(args, compilerPath)
	for i, curr := range action.Arguments[1:] {
		prev := action.Arguments[i]
		if strings.HasPrefix(curr, "-fdebug-prefix-map") {
			continue
		}
		if prev == "-c" {
			src = curr
		}

		if newVer, has := pinversion[curr]; has {
			curr = newVer
		}

		if strings.HasPrefix(curr, "-W") || strings.HasPrefix(curr, "-std=") {
			if _, has := dedup[curr]; has {
				continue
			}
		}

		args = append(args, curr)
		dedup[curr] = struct{}{}
	}
	if src == "" {
		err = fmt.Errorf("unable to find source .cc file for targetId %d", action.TargetID)
		return src, args, err
	}
	return src, args, err
}

// ConvertCompileCommands converts from Bazel's aquery format to de-Bazeled compile_commands.json entries.
func (g *CompileCommandGenerator) ConvertCompileCommands(output AqueryOutput) ([]CompileCommand, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("unable to get cwd: %w", err)
	}
	// Ignore tools (which lead to duplicates) - we only need to
	// generate entries for code that we build as part of Redpanda or
	// it's tests.
	isToolConfig := make(map[int]bool, len(output.Configuration))
	for _, config := range output.Configuration {
		isToolConfig[config.ID] = config.IsTool
	}
	cmds := make([]CompileCommand, 0, len(output.Actions))
	for _, action := range output.Actions {
		if isToolConfig[action.ConfigurationID] {
			continue
		}
		src, args, err := g.GetCppCommandForFiles(action)
		if err != nil {
			return nil, fmt.Errorf("unable to get cpp command: %w", err)
		}
		// Skip Bazel internal files
		if strings.HasPrefix(src, "external/bazel_tools/") {
			continue
		}
		cmds = append(cmds, CompileCommand{
			File:      src,
			Arguments: args,
			Directory: cwd,
		})
	}
	return cmds, nil
}

func run(args []string) error {
	if err := SwitchCWDToWorkspaceRoot(); err != nil {
		return err
	}
	if err := SymlinkExternalToWorkspaceRoot(); err != nil {
		return err
	}

	output, err := GetCommands(args...)
	if err != nil {
		return err
	}

	generator := CompileCommandGenerator{
		compilerPathCache: map[string]string{},
	}

	cmds, err := generator.ConvertCompileCommands(output)
	if err != nil {
		return err
	}
	return Sink(cmds)
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Printf("unable to generate compilation database %+v", err)
		os.Exit(1)
	}
}
