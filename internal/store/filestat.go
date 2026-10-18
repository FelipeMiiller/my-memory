package store

import (
	"os"
)

// FileStatValues devolve o mtime (Unix, em segundos) e o tamanho em bytes de um
// arquivo, lidos do disco no momento da indexação (ADR-057).
//
// São os metadados que descrevem o arquivo, não o indexador: mtime é o instante
// em que o autor salvou a nota, enquanto updated_at é o instante em que nós
// indexamos. Um "mem index" completo dá a todos os documentos o mesmo
// updated_at, o que tornaria o decaimento temporal um no-op.
//
// Falha de stat não interrompe a indexação: devolve (0, 0), que é exatamente o
// valor que uma linha legada tem e que os leitores já sabem tratar como
// "mtime desconhecido".
func FileStatValues(path string) (mtime int64, size int64) {
	info, err := os.Stat(path)
	if err != nil || info == nil {
		return 0, 0
	}
	return info.ModTime().Unix(), info.Size()
}
