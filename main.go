package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/KofTwentyTwo/limen/internal/config"
	"github.com/KofTwentyTwo/limen/internal/state"
)

type output struct {
	Config config.Config `json:"config"`
	State  state.State   `json:"state"`
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("limen", flag.ContinueOnError)
	flags.SetOutput(stderr)

	configPath := flags.String("config", config.DefaultPath(), "path to hosts.json")
	statePath := flags.String("state", state.DefaultPath(), "path to state.json")
	if err := flags.Parse(args); err != nil {
		return 2
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
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output{Config: cfg, State: st}); err != nil {
		fmt.Fprintf(stderr, "limen: cannot write output: %v\n", err)
		return 1
	}

	return 0
}
