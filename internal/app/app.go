package app

import (
	"fmt"
	"io"
	"os"

	"github.com/PolarKits/PolarSwarm/internal/config"
)

const Version = "0.0.0-dev"

type CLI struct {
	Stdout io.Writer
	Stderr io.Writer
}

func Run(args []string) error {
	cli := CLI{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	return cli.Run(args)
}

func (c CLI) Run(args []string) error {
	if c.Stdout == nil {
		c.Stdout = io.Discard
	}

	if len(args) == 0 {
		printHelp(c.Stdout)
		return nil
	}

	switch args[0] {
	case "help", "-h", "--help":
		printHelp(c.Stdout)
		return nil
	case "config":
		return c.runConfig(args[1:])
	case "version", "-v", "--version":
		fmt.Fprintf(c.Stdout, "polarswarm %s\n", Version)
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func (c CLI) runConfig(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing config command")
	}
	if args[0] != "check" {
		return fmt.Errorf("unknown config command %q", args[0])
	}

	path := config.DefaultPath
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--config":
			i++
			if i >= len(args) {
				return fmt.Errorf("--config requires a path")
			}
			path = args[i]
		default:
			return fmt.Errorf("unknown config check argument %q", args[i])
		}
	}

	cfg, err := config.Load(path)
	if err != nil {
		return err
	}

	fmt.Fprintf(c.Stdout, "config ok: %s\n", cfg.Summary())
	return nil
}

func printHelp(w io.Writer) {
	fmt.Fprintln(w, "PolarSwarm - local IssueOps orchestrator")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  polarswarm help")
	fmt.Fprintln(w, "  polarswarm config check [--config path]")
	fmt.Fprintln(w, "  polarswarm version")
}
