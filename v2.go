package main

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
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
		var t Target
		realArgs := make([]string, 0, len(action.Arguments))
		if strings.HasSuffix(action.Arguments[0], cpp) {
			if cxx := os.Getenv("CXX"); len(cxx) > 0 {
				realArgs = append(realArgs, cxx)
			} else {
				realArgs = append(realArgs, action.Arguments[0])
			}
		}

		dedup := make(map[string]struct{}, len(action.Arguments)+2)

		for i, args := range action.Arguments[1:] {
			pre := action.Arguments[i]
			if strings.HasPrefix(args, "-fdebug-prefix-map") {
				continue
			}

			ph := args

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

	log.Info().Str("BUILD_WORKSPACE_DIRECTORY", dir).Send()

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
	now := time.Now()
	defer func(start time.Time) {
		log.Info().Dur("cost", time.Since(start)).Msg("generate compile database")
	}(now)

	if err := runv2(os.Args[1:]); err != nil {
		log.Error().Err(err).Msg("unable to generate compilation database")
		os.Exit(1)
	}
}
