package store

import (
	"crypto/sha256"
	"encoding/hex"
)

// CalculateContentHash calcula o hash SHA-256 em formato hexadecimal de 64 caracteres
// para qualquer fatia de bytes representando o conteúdo de um documento.
func CalculateContentHash(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}
