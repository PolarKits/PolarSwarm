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
	"github.com/PolarKits/PolarSwarm/internal/doctor"
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
	case "acceptance":
		return c.runAcceptance(args[1:])
	case "config":
		return c.runConfig(args[1:])
	case "issue":
		return c.runIssue(args[1:])
	case "writeback":
		return c.runWriteback(args[1:])
	case "version", "-v", "--version":
		fmt.Fprintf(c.Stdout, "polarswarm %s\n", Version)
		return nil
	case "doctor":
		return c.runDoctor(args[1:])
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func (c CLI) runAcceptance(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing acceptance command")
	}
	if args[0] != "dry-run" {
		return fmt.Errorf("unknown acceptance command %q", args[0])
	}

	var repo gh.Repository
	var fixturePath string
	number := 0
	role := "developer"
	branch := ""
	targetLabel := "status:in-progress"
	forceFailure := false

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
		case "--target-label":
			i++
			if i >= len(args) {
				return fmt.Errorf("--target-label requires a status label")
			}
			targetLabel = args[i]
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
		case "--force-failure":
			forceFailure = true
		default:
			return fmt.Errorf("unknown acceptance dry-run argument %q", args[i])
		}
	}

	if fixturePath == "" {
		return fmt.Errorf("acceptance dry-run requires --fixture")
	}

	client, err := gh.LoadFakeClient(fixturePath)
	if err != nil {
		return err
	}
	cfg := config.Config{
		GitHub: config.GitHubConfig{
			Owner: repo.Owner,
			Repo:  repo.Name,
		},
		Workflow: config.WorkflowConfig{
			TargetLabel: targetLabel,
			DryRun:      true,
		},
	}
	loop := AcceptanceLoop{
		Reader: gh.IssueReader{Client: client},
		Runner: agent.MockRunner{},
	}
	result, err := loop.Run(context.Background(), AcceptanceOptions{
		Config:       cfg,
		Repository:   repo,
		IssueNumber:  number,
		Role:         role,
		Branch:       branch,
		ForceFailure: forceFailure,
	})
	if err != nil {
		return err
	}

	fmt.Fprintf(c.Stdout, "acceptance dry-run for %s#%d\n", repo, number)
	fmt.Fprintf(c.Stdout, "operation_id: %s\n", result.OperationID)
	if result.Skipped {
		fmt.Fprintf(c.Stdout, "skipped: %s\n", result.SkipReason)
		return nil
	}
	fmt.Fprintf(c.Stdout, "dispatchable: yes\n")
	fmt.Fprintf(c.Stdout, "state: %s -> %s\n", result.Current, result.Target)
	fmt.Fprint(c.Stdout, result.WritePlan.DryRunText())
	return nil
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

func (c CLI) runDoctor(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing doctor command: config, github, labels, llm, capabilities, workflows, store, worktrees, or --list")
	}

	switch args[0] {
	case "github":
		return c.runDoctorGitHub(args[1:])
	case "config":
		return c.runDoctorConfig()
	case "labels":
		return c.runDoctorLabels()
	case "--list":
		c.listDoctorCategories()
		return nil
	default:
		return fmt.Errorf("unknown doctor command %q", args[0])
	}
}

func (c CLI) runDoctorGitHub(args []string) error {
	var owner, repo string

	for i := 0; i < len(args); i++ {
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
			owner = parsed.Owner
			repo = parsed.Name
		default:
			return fmt.Errorf("unknown doctor github argument %q", args[i])
		}
	}

	// Load config if available to get owner/repo
	cfg, err := config.Load(config.DefaultPath)
	if err == nil && owner == "" {
		owner = cfg.GitHub.Owner
		repo = cfg.GitHub.Repo
	}

	g := doctor.GitHub{
		Owner:  owner,
		Repo:   repo,
		Output: c.Stdout,
	}

	results, err := g.Run(context.Background())
	if err != nil {
		// Error already printed, just return it
		return err
	}

	// Print summary
	fmt.Fprintf(c.Stdout, "\nSummary: %s\n", doctor.FormatSummary(results))
	if doctor.HasFailures(results) {
		return fmt.Errorf("github health check failed")
	}
	return nil
}

func (c CLI) runDoctorConfig() error {
	path := config.DefaultPath
	cfg, err := config.Load(path)
	if err != nil {
		return err
	}
	fmt.Fprintf(c.Stdout, "[ config ]  ok  %s\n", cfg.Summary())
	return nil
}

func (c CLI) runDoctorLabels() error {
	// Load config to get owner/repo
	cfg, err := config.Load(config.DefaultPath)
	if err != nil {
		// Config not required for labels check - might be run before init
		fmt.Fprintf(c.Stderr, "[ labels ]  %s  Config not found, using defaults\n", "⚠")
	}

	l := doctor.Labels{
		Owner:  cfg.GitHub.Owner,
		Repo:   cfg.GitHub.Repo,
		Output: c.Stdout,
	}

	result, err := l.Run(context.Background())
	if err != nil {
		return err
	}

	// Print summary
	if len(result.Missing) > 0 || len(result.Mismatches) > 0 {
		fmt.Fprintf(c.Stdout, "\nMissing %d of %d standard labels\n", len(result.Missing), result.Total)
		return nil
	}
	fmt.Fprintf(c.Stdout, "\nAll %d standard labels present\n", result.Total)
	return nil
}

func (c CLI) listDoctorCategories() {
	fmt.Fprintln(c.Stdout, "Available doctor categories:")
	fmt.Fprintln(c.Stdout, "  config         - Check configuration files")
	fmt.Fprintln(c.Stdout, "  github         - Check GitHub token and repository access")
	fmt.Fprintln(c.Stdout, "  labels         - Check repository labels")
	fmt.Fprintln(c.Stdout, "  capabilities   - Check GitHub capabilities")
	fmt.Fprintln(c.Stdout, "  workflows      - Check GitHub Actions workflows")
	fmt.Fprintln(c.Stdout, "  store          - Check local data store")
	fmt.Fprintln(c.Stdout, "  worktrees      - Check git worktrees")
	fmt.Fprintln(c.Stdout, "  llm            - Check LLM backend connectivity (uses tokens)")
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
	fmt.Fprintln(w, "  polarswarm acceptance dry-run --repo owner/name --number n --fixture path [--target-label status:state] [--force-failure]")
	fmt.Fprintln(w, "  polarswarm config check [--config path]")
	fmt.Fprintln(w, "  polarswarm doctor github [--repo owner/name]")
	fmt.Fprintln(w, "  polarswarm doctor config")
	fmt.Fprintln(w, "  polarswarm doctor labels")
	fmt.Fprintln(w, "  polarswarm issue read --repo owner/name --number n --fixture path")
	fmt.Fprintln(w, "  polarswarm writeback dry-run --repo owner/name --number n --fixture path [--target-state state]")
	fmt.Fprintln(w, "  polarswarm version")
}
