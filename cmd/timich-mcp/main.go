package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/rsahara/timich-mcp/internal/app"
	"github.com/rsahara/timich-mcp/internal/mcpserver"
	"github.com/rsahara/timich-mcp/internal/state"
)

var (
	version = "dev"
	commit  = "unknown"
	builtAt = "unknown"
)

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) error {
	if len(args) == 0 {
		printUsage(stdout)
		return nil
	}
	switch args[0] {
	case "pair":
		return runPair(ctx, args[1:], stdout)
	case "status":
		return runStatus(ctx, args[1:], stdout)
	case "logout":
		return runLogout(args[1:], stdout)
	case "serve":
		return runServe(ctx, args[1:])
	case "version", "--version", "-v":
		fmt.Fprintf(stdout, "timich-mcp %s commit=%s builtAt=%s\n", version, commit, builtAt)
		return nil
	case "help", "--help", "-h":
		printUsage(stdout)
		return nil
	default:
		printUsage(stderr)
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func runPair(ctx context.Context, args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("pair", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	stateDir := flags.String("state-dir", "", "state directory")
	agentURL := flags.String("agent-url", "", "Timich Agent media API URL, for example http://10.0.1.4:8082")
	pairingCode := flags.String("pairing-code", "", "pairing code created in Timich Agent Admin UI")
	deviceName := flags.String("device-name", "", "paired device name")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("pair does not accept positional arguments")
	}
	timich := newApp(*stateDir)
	result, err := timich.Pair(ctx, *agentURL, *pairingCode, *deviceName)
	if err != nil {
		return err
	}
	if result.Paired {
		fmt.Fprintln(stdout, "paired: yes")
	}
	if result.Status != nil {
		printStatus(stdout, *result.Status)
	}
	if result.Warning != "" {
		fmt.Fprintf(stdout, "warning: paired, but status check failed: %s\n", result.Warning)
	}
	return nil
}

func runStatus(ctx context.Context, args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("status", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	stateDir := flags.String("state-dir", "", "state directory")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("status does not accept positional arguments")
	}
	timich := newApp(*stateDir)
	status, err := timich.Status(ctx)
	if err != nil {
		return err
	}
	printStatus(stdout, status)
	return nil
}

func runLogout(args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("logout", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	stateDir := flags.String("state-dir", "", "state directory")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("logout does not accept positional arguments")
	}
	timich := newApp(*stateDir)
	if err := timich.Logout(); err != nil {
		return err
	}
	fmt.Fprintln(stdout, "logged out")
	return nil
}

func runServe(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("serve", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	stateDir := flags.String("state-dir", "", "state directory")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("serve does not accept positional arguments")
	}
	timich := newApp(*stateDir)
	_ = timich.CleanupOldPreviews()
	return mcpserver.Run(ctx, timich, version)
}

func newApp(stateDir string) *app.App {
	return app.New(state.NewStore(stateDir), nil, "")
}

func printStatus(stdout io.Writer, status app.StatusResult) {
	fmt.Fprintf(stdout, "paired: %s\n", yesNo(status.Paired))
	if status.AgentBaseURL != "" {
		fmt.Fprintf(stdout, "agent: %s\n", status.AgentBaseURL)
	}
	if status.AgentName != "" {
		fmt.Fprintf(stdout, "agentName: %s\n", status.AgentName)
	}
	if status.DeviceName != "" {
		fmt.Fprintf(stdout, "deviceName: %s\n", status.DeviceName)
	}
	if status.AccessMode != "" {
		fmt.Fprintf(stdout, "accessMode: %s\n", status.AccessMode)
	}
	if !status.AccessTokenExpiresAt.IsZero() {
		fmt.Fprintf(stdout, "accessTokenExpiresAt: %s\n", formatTime(status.AccessTokenExpiresAt))
	}
	if !status.RefreshTokenExpiresAt.IsZero() {
		fmt.Fprintf(stdout, "refreshTokenExpiresAt: %s\n", formatTime(status.RefreshTokenExpiresAt))
	}
	fmt.Fprintf(stdout, "capabilities: %s\n", yesNo(status.CapabilitiesOK))
	fmt.Fprintf(stdout, "timelineSearch: %s", yesNo(status.SearchOK))
	if status.SearchOK {
		fmt.Fprintf(stdout, " total=%d totalAccuracy=%s itemCount=%d", status.SearchTotal, status.SearchTotalAccuracy, status.SearchItemCount)
	}
	fmt.Fprintln(stdout)
}

func printUsage(out io.Writer) {
	fmt.Fprintln(out, strings.TrimSpace(`
timich-mcp

Usage:
  timich-mcp pair --agent-url http://HOST:8082 --pairing-code CODE [--device-name NAME]
  timich-mcp status
  timich-mcp serve
  timich-mcp logout
  timich-mcp version

Options:
  --state-dir DIR  Override the local state directory.
`))
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339)
}
