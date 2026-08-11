// Package algorithms contiene los esquemas de integridad de la capa de Enlace.
package algorithms

import (
	"fmt"
	"strings"
)

// Hamming(7,4) por bloques: paridad par, correccion de un bit por bloque de
// 7 (4 bits de datos + 3 de paridad). El ultimo bloque se rellena con ceros
// hasta completar 4 bits; el llamador recuerda la longitud original (campo
// "len" del envoltorio DATA) para descartar el relleno al decodificar.
const (
	blockDataBits   = 4
	blockParityBits = 3
	blockCodeBits   = blockDataBits + blockParityBits
)

var parityPositions = [...]int{1, 2, 4}

// HammingResult describe la verificacion y posible correccion de una trama.
type HammingResult struct {
	DataBits  string
	Corrected bool
	Syndrome  int
	Valid     bool
	Error     string
}

func validateBits(bits string) error {
	for _, bit := range bits {
		if bit != '0' && bit != '1' {
			return fmt.Errorf("la cadena solo puede contener bits 0 y 1")
		}
	}
	return nil
}

func blocksNeeded(messageBits int) int {
	return (messageBits + blockDataBits - 1) / blockDataBits
}

// RequiredParityBits retorna los bits de paridad totales para proteger
// messageBits en bloques de 4.
func RequiredParityBits(messageBits int) (int, error) {
	if messageBits < 0 {
		return 0, fmt.Errorf("messageBits no puede ser negativo")
	}
	return blocksNeeded(messageBits) * blockParityBits, nil
}

// encodeBlock inserta paridad par en las posiciones 1, 2, 4 de un bloque de 4 bits.
func encodeBlock(dataBits string) string {
	codeword := make([]byte, blockCodeBits+1)
	dataIndex := 0
	for position := 1; position < len(codeword); position++ {
		if position&(position-1) != 0 {
			codeword[position] = dataBits[dataIndex] - '0'
			dataIndex++
		}
	}
	for _, parityPosition := range parityPositions {
		parity := byte(0)
		for position := 1; position < len(codeword); position++ {
			if position&parityPosition != 0 {
				parity ^= codeword[position]
			}
		}
		codeword[parityPosition] = parity
	}
	var builder strings.Builder
	builder.Grow(blockCodeBits)
	for _, bit := range codeword[1:] {
		builder.WriteByte(bit + '0')
	}
	return builder.String()
}

// HammingEncode aplica Hamming(7,4) a dataBits en bloques de 4, con relleno de ceros.
func HammingEncode(dataBits string) (string, error) {
	if err := validateBits(dataBits); err != nil {
		return "", err
	}
	if dataBits == "" {
		return "", nil
	}
	padded := dataBits + strings.Repeat("0", blocksNeeded(len(dataBits))*blockDataBits-len(dataBits))
	var builder strings.Builder
	builder.Grow(len(padded) / blockDataBits * blockCodeBits)
	for start := 0; start < len(padded); start += blockDataBits {
		builder.WriteString(encodeBlock(padded[start : start+blockDataBits]))
	}
	return builder.String(), nil
}

func blockSyndrome(codeword []byte) int {
	syndrome := 0
	for _, parityPosition := range parityPositions {
		parity := byte(0)
		for position := 1; position < len(codeword); position++ {
			if position&parityPosition != 0 {
				parity ^= codeword[position]
			}
		}
		if parity != 0 {
			syndrome += parityPosition
		}
	}
	return syndrome
}

// decodeBlock verifica y corrige un unico bloque de 7 bits.
func decodeBlock(encodedBlock string) (data string, corrected bool, syndrome int, valid bool, errMsg string) {
	codeword := make([]byte, blockCodeBits+1)
	for index := 0; index < blockCodeBits; index++ {
		codeword[index+1] = encodedBlock[index] - '0'
	}
	syndrome = blockSyndrome(codeword)
	if syndrome != 0 {
		// Hamming SEC corrige un bit volteando la posicion indicada por el sindrome.
		if syndrome >= len(codeword) {
			return "", false, syndrome, false, "el sindrome apunta fuera del bloque; error no corregible"
		}
		codeword[syndrome] ^= 1
		corrected = true
		if blockSyndrome(codeword) != 0 {
			return "", false, syndrome, false, "el bloque conserva paridad invalida despues de corregir"
		}
	}
	var builder strings.Builder
	builder.Grow(blockDataBits)
	for position := 1; position < len(codeword); position++ {
		if position&(position-1) != 0 {
			builder.WriteByte(codeword[position] + '0')
		}
	}
	return builder.String(), corrected, syndrome, true, ""
}

// HammingDecode verifica/corrige cada bloque de 7 bits y extrae messageLength bits de datos.
func HammingDecode(encodedBits string, messageLength int) HammingResult {
	if err := validateBits(encodedBits); err != nil {
		return HammingResult{Valid: false, Error: err.Error()}
	}
	if messageLength < 0 {
		return HammingResult{Valid: false, Error: "messageLength no puede ser negativo"}
	}
	blocks := blocksNeeded(messageLength)
	expectedLength := blocks * blockCodeBits
	if len(encodedBits) != expectedLength {
		return HammingResult{Valid: false, Error: fmt.Sprintf("longitud Hamming invalida: se esperaban %d bits", expectedLength)}
	}
	if encodedBits == "" {
		return HammingResult{Valid: true}
	}

	var dataBuilder strings.Builder
	dataBuilder.Grow(blocks * blockDataBits)
	correctedAny := false
	lastSyndrome := 0
	for blockIndex := 0; blockIndex < blocks; blockIndex++ {
		start := blockIndex * blockCodeBits
		block := encodedBits[start : start+blockCodeBits]
		data, corrected, syndrome, valid, errMsg := decodeBlock(block)
		if !valid {
			return HammingResult{Valid: false, Syndrome: syndrome, Error: fmt.Sprintf("bloque %d: %s", blockIndex, errMsg)}
		}
		dataBuilder.WriteString(data)
		if corrected {
			correctedAny = true
			lastSyndrome = syndrome
		}
	}
	value := dataBuilder.String()
	return HammingResult{DataBits: value[:messageLength], Corrected: correctedAny, Syndrome: lastSyndrome, Valid: true}
}
