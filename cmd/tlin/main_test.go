package main

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestMainFunction_Example demonstrates how to perform integration testing
// of the main function. It uses a parent-child process approach to test
// the actual main function execution.
func TestMainFunction_Example(t *testing.T) {
	// Test runner is the parent process
	// Fork into child process (= actual main call)
	if os.Getenv("TEST_MAIN_EXAMPLE") != "1" {
		// 1) Child process invocation section
		// Create temporary directory for testing
		tempDir, err := os.MkdirTemp("", "main-test-example")
		assert.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Example of executing "tlin init" subcommand
		cmd := exec.Command(os.Args[0], "-test.run=TestMainFunction_Example")
		cmd.Env = append(os.Environ(), "TEST_MAIN_EXAMPLE=1")
		cmd.Env = append(cmd.Env, fmt.Sprintf("TEST_TMP_DIR=%s", tempDir))

		// Actual subcommand arguments
		// Following `os.Args[0]`, we could add "init" etc.
		// However, here we'll use a trick to make the child process
		// use mainTestArgs, so we can pass it via environment
		// variables instead of command line arguments.
		output, err := cmd.CombinedOutput()

		// Error occurs here if child process calls `os.Exit(1)`
		exitError, _ := err.(*exec.ExitError)
		if exitError != nil && exitError.ExitCode() != 0 {
			t.Fatalf("process failed with exit code %d: %s",
				exitError.ExitCode(), string(output))
		}

		// Verification logic for output (= stdout+stderr)
		t.Logf("child process output: %s", output)

		// Example: verify if config file was created properly
		configPath := filepath.Join(tempDir, ".tlin.yaml")
		_, statErr := os.Stat(configPath)
		assert.NoError(t, statErr, "config file must exist after init command")
		return
	}

	// 2) Child process section
	//    Direct call to `main()` happens here
	//    Assuming execution of "tlin init" command
	//    Get temporary directory path from `TEST_TMP_DIR`
	tempDir := os.Getenv("TEST_TMP_DIR")
	if tempDir == "" {
		fmt.Println("TEST_TMP_DIR not set")
		os.Exit(1)
	}

	// Can simulate actual CLI arguments
	// Example: tlin init --config=<path>
	os.Args = []string{
		"tlin", // fake argv[0]
		"init", // actual subcommand
		"--config", filepath.Join(tempDir, ".tlin.yaml"),
	}

	main()
	// After execution, `os.Exit(0)` returns to parent process
	os.Exit(0)
}
