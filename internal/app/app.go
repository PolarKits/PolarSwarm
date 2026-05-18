package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/PolarKits/PolarSwarm/internal/agent"
	"github.com/PolarKits/PolarSwarm/internal/config"
	gh "github.com/PolarKits/PolarSwarm/internal/github"
	"github.com/PolarKits/PolarSwarm/internal/workflow"
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
	case "issue":
		return c.runIssue(args[1:])
	case "writeback":
		return c.runWriteback(args[1:])
	case "version", "-v", "--version":
		fmt.Fprintf(c.Stdout, "polarswarm %s\n", Version)
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func (c CLI) runWriteback(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing writeback command")
	}
	if args[0] != "dry-run" {
		return fmt.Errorf("unknown writeback command %q", args[0])
	}

	var repo gh.Repository
	var fixturePath string
	number := 0
	role := "developer"
	branch := ""
	status := agent.StatusCompleted
	target := workflow.StateReview

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--repo":
			i++
			if i >= len(args) {
				return fmt.Errorf("--repo requires owner/name")
			}
			parsed, err := parseRepository(args[i])
			if err != nil {
				return err
			}
			repo = parsed
		case "--number":
			i++
			if i >= len(args) {
				return fmt.Errorf("--number requires an issue number")
			}
			parsed, err := strconv.Atoi(args[i])
			if err != nil {
				return fmt.Errorf("--number requires an integer: %w", err)
			}
			number = parsed
		case "--fixture":
			i++
			if i >= len(args) {
				return fmt.Errorf("--fixture requires a path")
			}
			fixturePath = args[i]
		case "--role":
			i++
			if i >= len(args) {
				return fmt.Errorf("--role requires a role")
			}
			role = args[i]
		case "--branch":
			i++
			if i >= len(args) {
				return fmt.Errorf("--branch requires a branch")
			}
			branch = args[i]
		case "--status":
			i++
			if i >= len(args) {
				return fmt.Errorf("--status requires completed or failed")
			}
			status = agent.Status(args[i])
			if !status.IsValid() {
				return fmt.Errorf("--status must be completed or failed")
			}
		case "--target-state":
			i++
			if i >= len(args) {
				return fmt.Errorf("--target-state requires a workflow state")
			}
			target = workflow.State(args[i])
			if !target.IsValid() {
				return fmt.Errorf("--target-state is invalid: %s", target)
			}
		default:
			return fmt.Errorf("unknown writeback dry-run argument %q", args[i])
		}
	}

	if fixturePath == "" {
		return fmt.Errorf("writeback dry-run requires --fixture")
	}
	if branch == "" {
		branch = fmt.Sprintf("task/%d", number)
	}

	client, err := gh.LoadFakeClient(fixturePath)
	if err != nil {
		return err
	}
	issue, err := (gh.IssueReader{Client: client}).ReadIssue(context.Background(), repo, number)
	if err != nil {
		return err
	}

	result, err := (agent.MockRunner{Status: status}).Run(context.Background(), agent.Request{
		Role: role,
		Issue: agent.IssueRef{
			Repository: repo.String(),
			Number:     issue.Number,
			Title:      issue.Title,
			URL:        issue.HTMLURL,
		},
		Branch: branch,
	})
	if err != nil {
		return err
	}

	plan, err := gh.PlanAgentResultWrite(result, labelNames(issue.Labels), target, gh.WriteOptions{DryRun: true})
	if err != nil {
		return err
	}
	fmt.Fprint(c.Stdout, plan.DryRunText())
	return nil
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

func (c CLI) runIssue(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing issue command")
	}
	if args[0] != "read" {
		return fmt.Errorf("unknown issue command %q", args[0])
	}

	var repo gh.Repository
	var fixturePath string
	number := 0
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--repo":
			i++
			if i >= len(args) {
				return fmt.Errorf("--repo requires owner/name")
			}
			parsed, err := parseRepository(args[i])
			if err != nil {
				return err
			}
			repo = parsed
		case "--number":
			i++
			if i >= len(args) {
				return fmt.Errorf("--number requires an issue number")
			}
			parsed, err := strconv.Atoi(args[i])
			if err != nil {
				return fmt.Errorf("--number requires an integer: %w", err)
			}
			number = parsed
		case "--fixture":
			i++
			if i >= len(args) {
				return fmt.Errorf("--fixture requires a path")
			}
			fixturePath = args[i]
		default:
			return fmt.Errorf("unknown issue read argument %q", args[i])
		}
	}

	if fixturePath == "" {
		return fmt.Errorf("issue read currently requires --fixture")
	}

	client, err := gh.LoadFakeClient(fixturePath)
	if err != nil {
		return err
	}
	issue, err := (gh.IssueReader{Client: client}).ReadIssue(context.Background(), repo, number)
	if err != nil {
		return err
	}

	fmt.Fprintf(c.Stdout, "issue %s#%d %s [%s]\n", issue.Repository, issue.Number, issue.Title, issue.State)
	fmt.Fprintf(c.Stdout, "labels: %s\n", formatLabels(issue.Labels))
	fmt.Fprintf(c.Stdout, "comments: %d\n", issue.Comments.Count)
	if issue.Comments.Latest != nil {
		fmt.Fprintf(c.Stdout, "latest_comment: %s\n", issue.Comments.Latest.Author)
	}
	return nil
}

func formatLabels(labels []gh.Label) string {
	if len(labels) == 0 {
		return "(none)"
	}
	names := make([]string, 0, len(labels))
	for _, label := range labels {
		names = append(names, label.Name)
	}
	return strings.Join(names, ", ")
}

func labelNames(labels []gh.Label) []string {
	names := make([]string, 0, len(labels))
	for _, label := range labels {
		names = append(names, label.Name)
	}
	return names
}

func parseRepository(value string) (gh.Repository, error) {
	owner, name, ok := strings.Cut(value, "/")
	if !ok || owner == "" || name == "" {
		return gh.Repository{}, fmt.Errorf("--repo must be owner/name")
	}
	return gh.Repository{Owner: owner, Name: name}, nil
}

func printHelp(w io.Writer) {
	fmt.Fprintln(w, "PolarSwarm - local IssueOps orchestrator")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  polarswarm help")
	fmt.Fprintln(w, "  polarswarm config check [--config path]")
	fmt.Fprintln(w, "  polarswarm issue read --repo owner/name --number n --fixture path")
	fmt.Fprintln(w, "  polarswarm writeback dry-run --repo owner/name --number n --fixture path [--target-state state]")
	fmt.Fprintln(w, "  polarswarm version")
}
