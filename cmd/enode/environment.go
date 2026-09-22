package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/taeels/enode/internal/enode"
	execenv "github.com/taeels/enode/internal/environment"
)

func runEnvironmentCmd(args []string) int {
	return runEnvironmentCmdWith(args, nil, enode.ExecutionRuntimeVerifier{})
}

func runEnvironmentCmdWith(args []string, inspector execenv.Inspector, verifier execenv.RuntimeVerifier) int {
	if len(args) == 0 || (args[0] != "check" && args[0] != "apply") {
		fmt.Fprintln(os.Stderr, "usage: enode env check|apply --config PATH [--json]")
		return 2
	}
	action := args[0]
	fs := flag.NewFlagSet("env "+action, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := fs.String("config", "", "node config path")
	jsonOutput := fs.Bool("json", false, "print structured JSON")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	resolved, tried := enode.ResolveConfig(*configPath)
	if resolved == "" {
		fmt.Fprintf(os.Stderr, "error: no config file found (tried %v)\n", tried)
		return 1
	}
	local, err := enode.LoadLocal(resolved)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	doc, binding, err := enode.LoadExecutionEnvironment(resolved, local)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	if action == "check" {
		report := execenv.CheckWithRuntime(context.Background(), doc, binding, inspector, verifier)
		printEnvironmentValue(report, *jsonOutput)
		if report.State != execenv.StateReady {
			return 2
		}
		return 0
	}
	manifest, err := (execenv.Preparer{Inspector: inspector, Verifier: verifier}).Apply(context.Background(), doc, binding)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	printEnvironmentValue(manifest, *jsonOutput)
	return 0
}

func printEnvironmentValue(value any, asJSON bool) {
	if asJSON {
		_ = json.NewEncoder(os.Stdout).Encode(value)
		return
	}
	switch v := value.(type) {
	case execenv.Report:
		fmt.Printf("state: %s\nprofile_sha256: %s\n", v.State, v.ProfileSHA256)
		for _, fact := range v.Facts {
			fmt.Printf("  %-34s %-18s %s\n", fact.Name, fact.State, fact.Observed)
			if fact.Remediation != "" {
				fmt.Printf("    remediation: %s\n", fact.Remediation)
			}
		}
		for _, op := range v.Operations {
			fmt.Printf("  plan: %-30s source=%s\n", op.Kind, op.Source)
		}
	case execenv.Manifest:
		fmt.Printf("prepared_environment_id: %s\nprofile_sha256: %s\n",
			v.PreparedEnvironmentID, v.Profile.SHA256)
	}
}
