package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestGetVersionInfo(t *testing.T) {
	origVersion := Version
	origCommit := GitCommit
	origDate := BuildDate
	defer func() {
		Version = origVersion
		GitCommit = origCommit
		BuildDate = origDate
	}()

	Version = "v1.2.3"
	GitCommit = "abcdef123456"
	BuildDate = "2026-09-13T20:00:00Z"

	info := GetVersionInfo()

	if info.Version != "v1.2.3" {
		t.Errorf("esperado Version 'v1.2.3', obteve '%s'", info.Version)
	}
	if info.GitCommit != "abcdef123456" {
		t.Errorf("esperado GitCommit 'abcdef123456', obteve '%s'", info.GitCommit)
	}
	if info.BuildDate != "2026-09-13T20:00:00Z" {
		t.Errorf("esperado BuildDate '2026-09-13T20:00:00Z', obteve '%s'", info.BuildDate)
	}
	if info.OS != runtime.GOOS {
		t.Errorf("esperado OS '%s', obteve '%s'", runtime.GOOS, info.OS)
	}
	if info.Arch != runtime.GOARCH {
		t.Errorf("esperado Arch '%s', obteve '%s'", runtime.GOARCH, info.Arch)
	}
	if !strings.HasPrefix(info.GoVersion, "go") {
		t.Errorf("esperado GoVersion iniciando com 'go', obteve '%s'", info.GoVersion)
	}
}

func TestFormatVersion(t *testing.T) {
	origVersion := Version
	origCommit := GitCommit
	origDate := BuildDate
	defer func() {
		Version = origVersion
		GitCommit = origCommit
		BuildDate = origDate
	}()

	Version = "v2.0.0"
	GitCommit = "1234567"
	BuildDate = "2026-01-01T00:00:00Z"

	formatted := formatVersion()
	if !strings.Contains(formatted, "my-memory v2.0.0") {
		t.Errorf("formatVersion deve conter 'my-memory v2.0.0', obteve: %s", formatted)
	}
	if !strings.Contains(formatted, "commit: 1234567") {
		t.Errorf("formatVersion deve conter 'commit: 1234567', obteve: %s", formatted)
	}
	if !strings.Contains(formatted, "built: 2026-01-01T00:00:00Z") {
		t.Errorf("formatVersion deve conter data de compilação, obteve: %s", formatted)
	}
}

func TestRunVersionCLI_Text(t *testing.T) {
	origVersion := Version
	defer func() { Version = origVersion }()
	Version = "v1.0.0-rc1"

	// Capturar stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runVersionCLI([]string{})
	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("runVersionCLI falhou: %v", err)
	}

	var buf bytes.Buffer
	io.Copy(&buf, r)
	out := buf.String()

	if !strings.Contains(out, "my-memory v1.0.0-rc1") {
		t.Errorf("saída CLI não contém versão esperada: %s", out)
	}
}

func TestRunVersionCLI_JSON(t *testing.T) {
	origVersion := Version
	origCommit := GitCommit
	defer func() {
		Version = origVersion
		GitCommit = origCommit
	}()
	Version = "v1.0.0"
	GitCommit = "fedcba987654"

	// Capturar stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runVersionCLI([]string{"--json"})
	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("runVersionCLI --json falhou: %v", err)
	}

	var buf bytes.Buffer
	io.Copy(&buf, r)
	out := buf.String()

	var parsed VersionInfo
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("falha ao fazer unmarshal do JSON da CLI: %v (raw: %s)", err, out)
	}

	if parsed.Version != "v1.0.0" {
		t.Errorf("esperado parsed.Version 'v1.0.0', obteve '%s'", parsed.Version)
	}
	if parsed.GitCommit != "fedcba987654" {
		t.Errorf("esperado parsed.GitCommit 'fedcba987654', obteve '%s'", parsed.GitCommit)
	}
	if parsed.OS != runtime.GOOS {
		t.Errorf("esperado parsed.OS '%s', obteve '%s'", runtime.GOOS, parsed.OS)
	}
}
