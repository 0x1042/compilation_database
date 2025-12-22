package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	needReplace = map[string]bool{
		"-isystem": true,
		"-MF":      true,
		"-iquote":  true,
	}

	prefixs = map[string]bool{
		"-I":             true,
		"-frandom-seed=": true,
	}
)

func convert(dir, arg string) string {
	for pre := range prefixs {
		if strings.HasPrefix(arg, pre) {
			val := arg[len(pre):]

			if len(val) == 0 {
				continue
			}

			if val == "." {
				return arg
			}
			return pre + filepath.Join(dir, val)
		}
	}
	return arg
}

func translate(dir string, aquery AqueryOutput) (Targets, error) {
	ts := make(Targets, 0, len(aquery.Actions))
	for _, action := range aquery.Actions {

		var pre string
		var t Target

		realArgs := make([]string, 0, len(action.Arguments))
		dedup := make(map[string]struct{}, len(action.Arguments)+2)

		for _, args := range action.Arguments {
			if strings.HasPrefix(args, "-fdebug-prefix-map") {
				continue
			}

			ph := args

			if strings.HasSuffix(args, cpp) {
				if cxx := os.Getenv("CXX"); len(cxx) > 0 {
					ph = cxx
				}
			}

			if pre == "-c" {
				t.File = filepath.Join(dir, args)
			}

			if pre == "-o" {
				t.Output = filepath.Join(dir, args)
			}

			if _, ok := needReplace[pre]; ok {
				if args != "." {
					ph = filepath.Join(dir, args)
				}
			} else {
				ph = convert(dir, ph)
			}

			if newVer, has := pinversion[ph]; has {
				ph = newVer
			}

			pre = args

			if strings.HasPrefix(ph, "-W") || strings.HasPrefix(ph, "-std=") {
				if _, has := dedup[ph]; has {
					continue
				}
				dedup[ph] = struct{}{}
				if len(ph) > 0 {
					realArgs = append(realArgs, ph)
				}
			} else {
				if len(ph) > 0 {
					realArgs = append(realArgs, ph)
				}
			}
		}

		t.Command = strings.Join(realArgs, " ")
		t.Directory = dir

		ts = append(ts, t)
	}

	return ts, nil
}

func runv2(args []string) error {
	if err := SwitchCWDToWorkspaceRoot(); err != nil {
		return err
	}
	if err := SymlinkExternalToWorkspaceRoot(); err != nil {
		return err
	}

	dir, _ := os.LookupEnv("BUILD_WORKSPACE_DIRECTORY")

	fmt.Printf("BUILD_WORKSPACE_DIRECTORY %s", dir)

	output, err := GetCommands(args...)
	if err != nil {
		return err
	}

	tmp, err := translate(dir, output)
	if err != nil {
		return err
	}

	return Sink(tmp)
}

func main() {
	if err := runv2(os.Args[1:]); err != nil {
		fmt.Printf("unable to generate compilation database %+v", err)
		os.Exit(1)
	}
}
