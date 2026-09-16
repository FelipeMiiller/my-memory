package autowire

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
)

// InjectMCPServer injeta ou atualiza a definição de um servidor MCP em um arquivo de configuração de forma não-destrutiva
func InjectMCPServer(target ClientTarget, serverName string, serverCfg ServerConfig, dryRun bool, force bool) (*InstallReport, error) {
	report := &InstallReport{
		Target: target,
	}

	cfgMap := make(map[string]interface{})
	var originalData []byte

	// 1. Carregar arquivo existente se houver
	if stat, err := os.Stat(target.ConfigPath); err == nil && !stat.IsDir() {
		data, readErr := os.ReadFile(target.ConfigPath)
		if readErr != nil {
			report.Status = StatusError
			report.Message = fmt.Sprintf("falha ao ler arquivo existente: %v", readErr)
			return report, readErr
		}
		originalData = data

		if len(data) > 0 {
			if err := json.Unmarshal(data, &cfgMap); err != nil {
				report.Status = StatusError
				report.Message = fmt.Sprintf("arquivo de configuração contém JSON inválido: %v", err)
				return report, fmt.Errorf("JSON inválido em '%s': %w", target.ConfigPath, err)
			}
		}
	}

	// 2. Localizar ou inicializar mcpServers
	var mcpServers map[string]interface{}
	if rawServers, ok := cfgMap["mcpServers"]; ok {
		if casted, isMap := rawServers.(map[string]interface{}); isMap {
			mcpServers = casted
		} else {
			mcpServers = make(map[string]interface{})
		}
	} else {
		mcpServers = make(map[string]interface{})
	}

	// 3. Preparar o payload do servidor
	newServerPayload := map[string]interface{}{
		"command": serverCfg.Command,
		"args":    serverCfg.Args,
	}
	if len(serverCfg.Env) > 0 {
		newServerPayload["env"] = serverCfg.Env
	}

	// 4. Verificar se já existe e se é idêntico
	isNew := true
	if existing, ok := mcpServers[serverName]; ok {
		isNew = false
		if !force && isPayloadEqual(existing, newServerPayload) {
			report.Status = StatusAlreadyUpToDate
			report.Message = "servidor MCP já configurado com parâmetros idênticos"
			return report, nil
		}
	}

	mcpServers[serverName] = newServerPayload
	cfgMap["mcpServers"] = mcpServers

	// 5. Serializar JSON formatado
	updatedJSON, err := json.MarshalIndent(cfgMap, "", "  ")
	if err != nil {
		report.Status = StatusError
		report.Message = fmt.Sprintf("erro ao serializar JSON: %v", err)
		return report, err
	}
	updatedJSON = append(updatedJSON, '\n')

	// 6. Se for simulação (--dry-run), reportar sem alterar o disco
	if dryRun {
		report.Status = StatusDryRun
		if isNew {
			report.Message = fmt.Sprintf("[dry-run] criaria/injetaria servidor '%s'", serverName)
		} else {
			report.Message = fmt.Sprintf("[dry-run] atualizaria servidor '%s'", serverName)
		}
		report.Diff = string(updatedJSON)
		return report, nil
	}

	// 7. Criar diretórios pais se necessário
	parentDir := filepath.Dir(target.ConfigPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		report.Status = StatusError
		report.Message = fmt.Sprintf("falha ao criar diretório '%s': %v", parentDir, err)
		return report, err
	}

	// 8. Se o arquivo já existia fisicamente, criar backup (.bak)
	if len(originalData) > 0 {
		backupFile := target.ConfigPath + ".bak"
		if err := os.WriteFile(backupFile, originalData, 0644); err == nil {
			report.BackupPath = backupFile
		}
	}

	// 9. Gravar arquivo atualizado
	if err := os.WriteFile(target.ConfigPath, updatedJSON, 0644); err != nil {
		report.Status = StatusError
		report.Message = fmt.Sprintf("falha ao gravar arquivo: %v", err)
		return report, err
	}

	if isNew {
		report.Status = StatusInstalled
		report.Message = fmt.Sprintf("servidor '%s' configurado com sucesso", serverName)
	} else {
		report.Status = StatusUpdated
		report.Message = fmt.Sprintf("servidor '%s' atualizado com sucesso", serverName)
	}

	return report, nil
}

func isPayloadEqual(existing interface{}, next map[string]interface{}) bool {
	existingMap, ok := existing.(map[string]interface{})
	if !ok {
		return false
	}

	// Comparar command
	if existingMap["command"] != next["command"] {
		return false
	}

	// Comparar args
	existingArgs, hasArgs := existingMap["args"].([]interface{})
	nextArgs, hasNextArgs := next["args"].([]string)

	if hasArgs && hasNextArgs {
		if len(existingArgs) != len(nextArgs) {
			return false
		}
		for i, a := range existingArgs {
			if fmt.Sprintf("%v", a) != nextArgs[i] {
				return false
			}
		}
	} else if hasArgs != hasNextArgs {
		return false
	}

	// Comparar env se existir
	if nextEnv, ok := next["env"].(map[string]string); ok {
		existingEnv, ok := existingMap["env"].(map[string]interface{})
		if !ok {
			return false
		}
		if len(nextEnv) != len(existingEnv) {
			return false
		}
		for k, v := range nextEnv {
			if fmt.Sprintf("%v", existingEnv[k]) != v {
				return false
			}
		}
	} else if _, ok := existingMap["env"]; ok {
		return false
	}

	return reflect.DeepEqual(existingMap["command"], next["command"])
}
