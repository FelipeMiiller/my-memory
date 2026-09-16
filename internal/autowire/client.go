package autowire

// ClientType representa o tipo do cliente de IA suportado
type ClientType string

const (
	ClientClaudeDesktop ClientType = "claude"
	ClientCursor        ClientType = "cursor"
	ClientVSCode        ClientType = "vscode"
	ClientWindsurf      ClientType = "windsurf"
)

// ConfigScope define se a configuração é de nível de máquina (global) ou de repositório (workspace)
type ConfigScope string

const (
	ScopeGlobal    ConfigScope = "global"
	ScopeWorkspace ConfigScope = "workspace"
	ScopeAll       ConfigScope = "all"
)

// ServerConfig representa o bloco de inicialização do servidor MCP serializado no JSON
type ServerConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env,omitempty"`
}

// ClientTarget descreve um alvo de configuração identificado no sistema operacional ou workspace
type ClientTarget struct {
	Type        ClientType  `json:"type"`
	Name        string      `json:"name"`
	Scope       ConfigScope `json:"scope"`
	ConfigPath  string      `json:"config_path"`
	Exists      bool        `json:"exists"`        // true se o arquivo de configuração já existe fisicamente
	ParentDirOK bool        `json:"parent_dir_ok"` // true se o diretório pai da ferramenta existe
}

// InstallStatus descreve o resultado da injeção
type InstallStatus string

const (
	StatusInstalled       InstallStatus = "installed"
	StatusUpdated         InstallStatus = "updated"
	StatusAlreadyUpToDate InstallStatus = "up_to_date"
	StatusDryRun          InstallStatus = "dry_run"
	StatusSkipped         InstallStatus = "skipped"
	StatusError           InstallStatus = "error"
)

// InstallReport resume a operação realizada em cada ferramenta
type InstallReport struct {
	Target     ClientTarget  `json:"target"`
	Status     InstallStatus `json:"status"`
	BackupPath string        `json:"backup_path,omitempty"`
	Message    string        `json:"message"`
	Diff       string        `json:"diff,omitempty"`
}
