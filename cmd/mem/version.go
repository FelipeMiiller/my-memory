package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"runtime"
)

var (
	// Version é injetado via -ldflags "-X main.Version=..."
	Version = "dev"
	// GitCommit é o hash SHA do commit injetado via -ldflags "-X main.GitCommit=..."
	GitCommit = "none"
	// BuildDate é a data ISO 8601 da compilação injetada via -ldflags "-X main.BuildDate=..."
	BuildDate = "unknown"
)

// VersionInfo armazena os metadados de versão do executável.
type VersionInfo struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

// GetVersionInfo retorna a estrutura preenchida com metadados do binário e runtime.
func GetVersionInfo() VersionInfo {
	return VersionInfo{
		Version:   Version,
		GitCommit: GitCommit,
		BuildDate: BuildDate,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}
}

// formatVersion formata a versão para exibição amigável em linha de comando.
func formatVersion() string {
	info := GetVersionInfo()
	return fmt.Sprintf("my-memory %s (commit: %s, built: %s, go: %s, %s/%s)",
		info.Version,
		info.GitCommit,
		info.BuildDate,
		info.GoVersion,
		info.OS,
		info.Arch,
	)
}

// runVersionCLI executa o comando de exibição de versão, suportando modo texto e JSON.
func runVersionCLI(args []string) error {
	versionCmd := flag.NewFlagSet("version", flag.ExitOnError)
	jsonOutput := versionCmd.Bool("json", false, "Exibe metadados de versão em formato JSON")
	if err := versionCmd.Parse(args); err != nil {
		return err
	}

	if *jsonOutput {
		info := GetVersionInfo()
		data, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			return fmt.Errorf("falha ao serializar versão em JSON: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	fmt.Println(formatVersion())
	return nil
}
