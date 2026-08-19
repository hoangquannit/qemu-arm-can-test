package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"example.com/qemu-arm-can-test/internal/evidence"
	"example.com/qemu-arm-can-test/internal/harness"
	"example.com/qemu-arm-can-test/internal/manifest"
	"example.com/qemu-arm-can-test/internal/socketcan"
)

const (
	exitPassed       = 0
	exitTestFailed   = 1
	exitHarnessError = 2
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(exitHarnessError)
	}

	switch os.Args[1] {
	case "run":
		os.Exit(runCommand(os.Args[2:]))
	case "doctor":
		os.Exit(doctorCommand())
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "testctl: unknown command %q\n\n", os.Args[1])
		usage()
		os.Exit(exitHarnessError)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `Usage:
  testctl doctor
  testctl run [--output runs] <manifest.yaml>

Exit codes:
  0  all tests passed
  1  ECU behavior test failed
  2  manifest or harness error`)
}

func runCommand(arguments []string) int {
	flags := flag.NewFlagSet("run", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	outputRoot := flags.String("output", "runs", "evidence output directory")
	interfaceOverride := flags.String("interface", "", "override the manifest SocketCAN interface")
	if err := flags.Parse(arguments); err != nil {
		return exitHarnessError
	}
	if flags.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "testctl run: exactly one manifest path is required")
		return exitHarnessError
	}

	run, rawManifest, err := manifest.Load(flags.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "testctl: %v\n", err)
		return exitHarnessError
	}
	if *interfaceOverride != "" {
		run.Interface = *interfaceOverride
	}
	cases, err := run.Validate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "testctl: invalid manifest: %v\n", err)
		return exitHarnessError
	}

	now := time.Now()
	runID, evidenceDir, err := evidence.CreateRunDirectory(*outputRoot, run.Name, now)
	if err != nil {
		fmt.Fprintf(os.Stderr, "testctl: %v\n", err)
		return exitHarnessError
	}
	if err := evidence.WriteManifest(evidenceDir, rawManifest); err != nil {
		fmt.Fprintf(os.Stderr, "testctl: %v\n", err)
		return exitHarnessError
	}

	fmt.Println("ARM Virtual ECU Test Harness")
	fmt.Printf("Run       %s\n", runID)
	fmt.Printf("Test      %s\n", run.Name)
	fmt.Printf("CAN       %s\n", run.Interface)
	fmt.Printf("Evidence  %s\n\n", evidenceDir)

	bus, err := socketcan.Open(run.Interface)
	if err != nil {
		fmt.Fprintf(os.Stderr, "testctl: open CAN interface: %v\n", err)
		return exitHarnessError
	}
	defer bus.Close()

	baseContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(baseContext, run.Timeout.Duration)
	defer cancel()

	result, err := (harness.Runner{Output: os.Stdout}).Run(ctx, run, cases, bus, runID, evidenceDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "testctl: run harness: %v\n", err)
		return exitHarnessError
	}
	if err := evidence.WriteResult(result); err != nil {
		fmt.Fprintf(os.Stderr, "testctl: %v\n", err)
		return exitHarnessError
	}

	fmt.Printf("\nRESULT    %s\n", result.Status)
	fmt.Printf("DURATION  %s\n", result.Duration.Round(time.Millisecond))
	fmt.Printf("EVIDENCE  %s\n", result.EvidenceDir)
	if result.Status == "passed" {
		return exitPassed
	}
	return exitTestFailed
}

func doctorCommand() int {
	fmt.Printf("%-14s %s\n", "Host OS", runtime.GOOS)
	if runtime.GOOS != "linux" {
		fmt.Println("SocketCAN      unavailable (Linux host required)")
		return exitHarnessError
	}

	failed := false
	for _, command := range []string{"ip", "docker", "qemu-system-arm"} {
		path, err := exec.LookPath(command)
		if err != nil {
			fmt.Printf("%-14s missing\n", command)
			failed = true
			continue
		}
		fmt.Printf("%-14s %s\n", command, path)
	}
	if failed {
		return exitHarnessError
	}
	return exitPassed
}
