package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/KofTwentyTwo/limen/internal/config"
	limenexec "github.com/KofTwentyTwo/limen/internal/exec"
	"github.com/KofTwentyTwo/limen/internal/state"
	"github.com/KofTwentyTwo/limen/internal/ui"
)

var version = "dev"

type appDeps struct {
	runUI   func(ui.App) (ui.Result, error)
	handoff func(limenexec.Plan, limenexec.Options) int
}

func main() {
	os.Exit(runWithDeps(os.Args[1:], os.Stdout, os.Stderr, appDeps{
		runUI:   ui.Run,
		handoff: limenexec.Handoff,
	}))
}

func run(args []string, stdout io.Writer, stderr io.Writer) int {
	return runWithDeps(args, stdout, stderr, appDeps{
		runUI:   ui.Run,
		handoff: limenexec.Handoff,
	})
}

func runWithDeps(args []string, stdout io.Writer, stderr io.Writer, deps appDeps) int {
	flags := flag.NewFlagSet("limen", flag.ContinueOnError)
	flags.SetOutput(stderr)

	configPath := flags.String("config", config.DefaultPath(), "path to hosts.json")
	statePath := flags.String("state", state.DefaultPath(), "path to state.json")
	showVersion := flags.Bool("version", false, "print version")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if *showVersion {
		fmt.Fprintln(stdout, version)
		return 0
	}
	if flags.NArg() != 0 {
		fmt.Fprintf(stderr, "limen: unexpected argument %q\n", flags.Arg(0))
		return 2
	}

	cfg, err := config.Load(*configPath, stderr)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	st := state.Load(*statePath)

	result, err := deps.runUI(ui.App{Config: cfg, State: st})
	if err != nil {
		fmt.Fprintf(stderr, "limen: ui failed: %v\n", err)
		return 1
	}
	if !result.HasPlan {
		return result.ExitCode
	}
	if result.Message != "" {
		fmt.Fprintln(stderr, result.Message)
	}

	return deps.handoff(result.Plan, limenexec.Options{
		StatePath: *statePath,
		Stderr:    stderr,
	})
}
