package app

import (
	"fmt"
	"io"
	"os"
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
	case "version", "-v", "--version":
		fmt.Fprintf(c.Stdout, "polarswarm %s\n", Version)
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func printHelp(w io.Writer) {
	fmt.Fprintln(w, "PolarSwarm - local IssueOps orchestrator")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  polarswarm help")
	fmt.Fprintln(w, "  polarswarm version")
}
