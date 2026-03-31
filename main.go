package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dreamph/profy/internal/appconfig"
	"github.com/dreamph/profy/internal/envloader"
	"github.com/dreamph/profy/internal/processx"
	"github.com/dreamph/profy/internal/projectref"
)

type cliOptions struct {
	Override      bool
	Verbose       bool
	PrintEnv      bool
	WatchEnv      bool
	WatchInterval time.Duration
	ConfigHome    string

	ProjectFile string
}

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	opts, profile, command, err := parseArgs(args)
	if err != nil {
		printErr(err)
		return 2
	}
	projectID, err := projectref.ReadProjectID(opts.ProjectFile)
	if err != nil {
		printErr(err)
		return 2
	}
	projectCfg, err := appconfig.LoadProjectConfig(projectID, opts.ConfigHome)
	if err != nil {
		printErr(err)
		return 2
	}
	profileCfg, err := projectCfg.ResolveProfile(profile)
	if err != nil {
		printErr(err)
		return 2
	}
	mergedEnv, err := envloader.BuildMergedEnv(projectCfg.ProjectDir, profileCfg.Files, opts.Override)
	if err != nil {
		printErr(err)
		return 1
	}
	if err := envloader.ValidateRequiredKeys(mergedEnv, profileCfg.RequiredKeys); err != nil {
		printErr(err)
		return 1
	}

	if opts.Verbose {
		fmt.Fprintf(os.Stderr, "[profy] project: %s\n", projectID)
		fmt.Fprintf(os.Stderr, "[profy] profile: %s\n", profile)
		fmt.Fprintf(os.Stderr, "[profy] command: %s\n", strings.Join(command, " "))
	}
	if opts.PrintEnv {
		envloader.PrintEnv(os.Stdout, mergedEnv)
		return 0
	}

	buildEnv := func() ([]string, error) {
		env, err := envloader.BuildMergedEnv(projectCfg.ProjectDir, profileCfg.Files, opts.Override)
		if err != nil {
			return nil, err
		}
		if err := envloader.ValidateRequiredKeys(env, profileCfg.RequiredKeys); err != nil {
			return nil, err
		}
		return env, nil
	}

	if opts.WatchEnv {
		watchFiles := buildWatchFiles(opts.ProjectFile, projectCfg.ProjectDir, profileCfg.Files)
		reload := make(chan struct{}, 1)
		stopWatch := watchForFileChanges(watchFiles, opts.WatchInterval, reload)
		defer stopWatch()
		exitCode, err := processx.RunWithReload(command, buildEnv, reload, opts.Verbose)
		if err != nil {
			printErr(err)
			if exitCode != 0 {
				return exitCode
			}
			return 1
		}
		return exitCode
	}

	exitCode, err := processx.Run(command, mergedEnv, opts.Verbose)
	if err != nil {
		printErr(err)
		if exitCode != 0 {
			return exitCode
		}
		return 1
	}
	return exitCode
}

func parseArgs(args []string) (cliOptions, string, []string, error) {
	var opts cliOptions
	fs := flag.NewFlagSet("profy", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.BoolVar(&opts.Override, "override", false, "override existing OS environment variables with env file values")
	fs.BoolVar(&opts.Verbose, "v", false, "verbose output")
	fs.BoolVar(&opts.Verbose, "verbose", false, "verbose output")
	fs.BoolVar(&opts.PrintEnv, "print-env", false, "print resolved environment and exit")
	fs.BoolVar(&opts.WatchEnv, "watch-env", false, "restart command when env/config files change")
	fs.DurationVar(&opts.WatchInterval, "watch-interval", time.Second, "file watch interval (e.g. 500ms, 2s)")
	fs.StringVar(&opts.ConfigHome, "config-home", projectref.DefaultConfigHome(), "external profy config home")
	fs.StringVar(&opts.ProjectFile, "project-file", ".profy.yml", "project config file path")
	if err := fs.Parse(args); err != nil {
		return cliOptions{}, "", nil, usageError()
	}

	rest := fs.Args()
	if len(rest) == 0 {
		return cliOptions{}, "", nil, usageError()
	}
	profile, command := rest[0], rest[1:]
	if !opts.PrintEnv && len(command) == 0 {
		return cliOptions{}, "", nil, usageError()
	}
	if opts.WatchInterval <= 0 {
		return cliOptions{}, "", nil, errors.New("watch-interval must be > 0")
	}
	return opts, profile, command, nil
}

func usageError() error {
	return errors.New(`usage:
  profy [--override] [--verbose] [--print-env] [--watch-env] [--watch-interval DURATION] [--config-home PATH] [--project-file FILE] <profile> <command> [args...]

examples:
  profy dev air -c .air.toml
  profy sit go run ./examples/app
  profy prod ./tmp/main
  profy --print-env dev`)
}

func printErr(err error) {
	fmt.Fprintln(os.Stderr, "profy:", err)
}

type fileState struct {
	Exists bool
	Size   int64
	ModNS  int64
}

func buildWatchFiles(projectFile, projectDir string, envFiles []string) []string {
	files := make([]string, 0, len(envFiles)+2)
	files = append(files, projectFile)
	files = append(files, filepath.Join(projectDir, "profy.json"))
	for _, rel := range envFiles {
		files = append(files, filepath.Join(projectDir, rel))
	}
	return files
}

func watchForFileChanges(files []string, interval time.Duration, out chan<- struct{}) func() {
	stop := make(chan struct{})
	current := snapshotFiles(files)
	ticker := time.NewTicker(interval)

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				next := snapshotFiles(files)
				if !fileStateEqual(current, next) {
					current = next
					select {
					case out <- struct{}{}:
					default:
					}
				}
			case <-stop:
				return
			}
		}
	}()
	return func() { close(stop) }
}

func snapshotFiles(files []string) map[string]fileState {
	out := make(map[string]fileState, len(files))
	for _, p := range files {
		fi, err := os.Stat(p)
		if err != nil {
			out[p] = fileState{}
			continue
		}
		out[p] = fileState{
			Exists: true,
			Size:   fi.Size(),
			ModNS:  fi.ModTime().UnixNano(),
		}
	}
	return out
}

func fileStateEqual(a, b map[string]fileState) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}
