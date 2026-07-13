package lints

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
)

type driverDeps struct {
	lookPath   func(string) (string, error)
	getenv     func(string) string
	ensureShim func(gnoplsPath string) (string, error)
}

func defaultDriverDeps() driverDeps {
	return driverDeps{
		lookPath: exec.LookPath,
		getenv:   os.Getenv,
		ensureShim: func(gnoplsPath string) (string, error) {
			return cachedGnoplsShim(os.TempDir(), runtime.GOOS, os.Getpid(), gnoplsPath)
		},
	}
}

// gnoDriverEnv routes go/packages.Load (used internally by
// golangci-lint) through gnopls so gno.land/... import paths
// resolve. Returns nil when gnopls or GNOROOT is missing — callers
// fall back to plain golangci-lint, which produces empty stdout on
// pure-gno packages and is already handled as "no issues".
func gnoDriverEnv(d driverDeps) []string {
	gnoplsPath, err := d.lookPath("gnopls")
	if err != nil {
		return nil
	}
	gnoroot := d.getenv("GNOROOT")
	if gnoroot == "" {
		return nil
	}
	shim, err := d.ensureShim(gnoplsPath)
	if err != nil {
		return nil
	}
	env := []string{
		"GOPACKAGESDRIVER=" + shim,
		"GNOROOT=" + gnoroot,
	}
	if v := d.getenv("GNOBUILTIN"); v != "" {
		env = append(env, "GNOBUILTIN="+v)
	}
	return env
}

var (
	shimOnce sync.Once
	shimPath string
	shimErr  error

	envOnce  sync.Once
	envCache []string
)

// cachedDriverEnv returns the full env to hand to the golangci-lint
// subprocess (process env + driver overrides), or nil when gnopls
// is unavailable and no override is needed. Detection involves
// PATH traversal, shim creation, and an os.Environ() snapshot —
// all of which would otherwise repeat per CheckPackage call.
func cachedDriverEnv() []string {
	envOnce.Do(func() {
		extra := gnoDriverEnv(defaultDriverDeps())
		if extra == nil {
			return
		}
		envCache = append(os.Environ(), extra...)
	})
	return envCache
}

// cachedGnoplsShim writes the GOPACKAGESDRIVER shim once per
// process. GOPACKAGESDRIVER is consumed as a single executable
// path, so wrapping `gnopls resolve` requires a script.
func cachedGnoplsShim(tempDir, goos string, pid int, gnoplsPath string) (string, error) {
	shimOnce.Do(func() {
		shimPath, shimErr = writeGnoplsShim(tempDir, goos, pid, gnoplsPath)
	})
	return shimPath, shimErr
}

func writeGnoplsShim(tempDir, goos string, pid int, gnoplsPath string) (string, error) {
	if goos == "windows" {
		path := filepath.Join(tempDir, fmt.Sprintf("tlin-gnopls-driver-%d.cmd", pid))
		body := fmt.Sprintf("@echo off\r\n\"%s\" resolve %%*\r\n", gnoplsPath)
		if err := os.WriteFile(path, []byte(body), 0o700); err != nil {
			return "", err
		}
		return path, nil
	}
	path := filepath.Join(tempDir, fmt.Sprintf("tlin-gnopls-driver-%d", pid))
	body := fmt.Sprintf("#!/bin/sh\nexec %q resolve \"$@\"\n", gnoplsPath)
	if err := os.WriteFile(path, []byte(body), 0o700); err != nil {
		return "", err
	}
	return path, nil
}
