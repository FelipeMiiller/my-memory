package db

// O vetor zero é o defeito da ISSUE-040: ele PREENCHE a coluna, então
// "existe" não denuncia nada. Estas duas funções comparam o vetor gravado
// com o vetor zero do schema — uma no caminho cru (float32) e outra no
// caminho comprimido (TurboQuant 4-bit).
//
// As duas moram aqui porque as duas servem ao mesmo par de consumidores: o
// doctor (que precisa reportar o chunk sem vetor real) e a busca vetorial
// (que precisa devolver só o chunk com similaridade de verdade).

// vetorCruzoEhZero diz se o BLOB serializado do vetor cru é o vetor zero.
//
// `serializeFloat32` grava little-endian float32: o zero é 0x00000000, ou
// seja, um BLOB de bytes zero. Um vetor real tem pelo menos um byte
// diferente — o float32 mais positivo possível (0x7F7FFFFF) tem byte alto
// 0x7F, e o mais negativo (0xFF7FFFFF) tem byte alto 0xFF.
func vetorCruzoEhZero(blob []byte) bool {
	if len(blob) == 0 {
		return false
	}
	for _, b := range blob {
		if b != 0 {
			return false
		}
	}
	return true
}

// turboQuantEhZero diz se o vetor comprimido é a assinatura do vetor zero.
//
// O `Quantize` normaliza cada componente pelo maior valor absoluto e grava
// o valor quantizado como nibble deslocado em 7: o zero de um componente
// vira o nibble 7. Então o vetor comprimido do vetor zero é um blob em que
// TODO nibble vale 7 — ou seja, todo byte vale 0x77.
//
// A assinatura é exata: um vetor real tem um componente de valor absoluto
// máximo, que normaliza para ±7 e vira o nibble 0 ou 14 — nunca 7. Então
// "todos os nibbles são 7" só acontece para o vetor zero.
//
// O último byte tem o nibble alto válido só quando a dimensão é ímpar: o
// `Quantize` deixa o nibble alto em zero quando não há segundo componente.
func turboQuantEhZero(data []byte, dim int) bool {
	numBytes := (dim + 1) / 2
	if len(data) != numBytes {
		return false
	}
	for i, b := range data {
		if b&0x0F != 7 {
			return false
		}
		if i == numBytes-1 && dim%2 == 1 {
			// o nibble alto deste byte não pertence a nenhum componente
			continue
		}
		if (b>>4)&0x0F != 7 {
			return false
		}
	}
	return true
}
