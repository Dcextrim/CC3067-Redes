// Package algorithms contiene los esquemas de integridad de la capa de Enlace.
package algorithms

import (
	"fmt"
	"strings"
)

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

// RequiredParityBits retorna el menor r tal que m+r+1 <= 2^r.
func RequiredParityBits(messageBits int) (int, error) {
	if messageBits < 0 {
		return 0, fmt.Errorf("messageBits no puede ser negativo")
	}
	r := 0
	for messageBits+r+1 > (1 << r) {
		r++
	}
	return r, nil
}

// HammingEncode inserta paridad par en las posiciones potencia de dos.
func HammingEncode(dataBits string) (string, error) {
	if err := validateBits(dataBits); err != nil {
		return "", err
	}
	if dataBits == "" {
		return "", nil
	}
	r, _ := RequiredParityBits(len(dataBits))
	codeword := make([]byte, len(dataBits)+r+1)
	dataIndex := 0
	// Las potencias de dos se reservan para paridad; el resto recibe los datos.
	for position := 1; position < len(codeword); position++ {
		if position&(position-1) != 0 {
			codeword[position] = dataBits[dataIndex] - '0'
			dataIndex++
		}
	}
	// Una paridad cubre los indices que tienen activo su bit correspondiente.
	for parityPosition := 1; parityPosition < (1 << r); parityPosition <<= 1 {
		parity := byte(0)
		for position := 1; position < len(codeword); position++ {
			if position&parityPosition != 0 {
				parity ^= codeword[position]
			}
		}
		codeword[parityPosition] = parity
	}
	var builder strings.Builder
	builder.Grow(len(codeword) - 1)
	for _, bit := range codeword[1:] {
		builder.WriteByte(bit + '0')
	}
	return builder.String(), nil
}

func hammingSyndrome(codeword []byte, parityCount int) int {
	// La suma de paridades fallidas forma la posicion binaria del error.
	syndrome := 0
	for parityPosition := 1; parityPosition < (1 << parityCount); parityPosition <<= 1 {
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

// HammingDecode verifica, corrige un error y extrae messageLength bits.
func HammingDecode(encodedBits string, messageLength int) HammingResult {
	if err := validateBits(encodedBits); err != nil {
		return HammingResult{Valid: false, Error: err.Error()}
	}
	r, err := RequiredParityBits(messageLength)
	if err != nil {
		return HammingResult{Valid: false, Error: err.Error()}
	}
	expectedLength := messageLength + r
	if len(encodedBits) != expectedLength {
		return HammingResult{Valid: false, Error: fmt.Sprintf("longitud Hamming invalida: se esperaban %d bits", expectedLength)}
	}
	if encodedBits == "" {
		return HammingResult{Valid: true}
	}
	codeword := make([]byte, len(encodedBits)+1)
	for index := range encodedBits {
		codeword[index+1] = encodedBits[index] - '0'
	}
	syndrome := hammingSyndrome(codeword, r)
	corrected := false
	if syndrome != 0 {
		// Hamming SEC corrige un bit volteando la posicion indicada por el sindrome.
		if syndrome >= len(codeword) {
			return HammingResult{Valid: false, Syndrome: syndrome, Error: "el sindrome apunta fuera de la trama; error no corregible"}
		}
		codeword[syndrome] ^= 1
		corrected = true
		if hammingSyndrome(codeword, r) != 0 {
			return HammingResult{Valid: false, Syndrome: syndrome, Error: "la trama conserva paridad invalida despues de corregir"}
		}
	}
	var data strings.Builder
	data.Grow(messageLength)
	for position := 1; position < len(codeword); position++ {
		if position&(position-1) != 0 {
			data.WriteByte(codeword[position] + '0')
		}
	}
	value := data.String()
	return HammingResult{DataBits: value[:messageLength], Corrected: corrected, Syndrome: syndrome, Valid: true}
}
