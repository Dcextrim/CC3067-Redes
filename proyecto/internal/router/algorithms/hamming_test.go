package algorithms

import (
	"strings"
	"testing"
)

func TestHammingManualVector(t *testing.T) {
	encoded, err := HammingEncode("1011")
	if err != nil {
		t.Fatal(err)
	}
	if encoded != "0110011" {
		t.Fatalf("Hamming(7,4) = %s, se esperaba 0110011", encoded)
	}
}

func TestHammingCorrectsEverySingleBitWithinOneBlock(t *testing.T) {
	message := "1011" // un solo bloque de 4 bits
	encoded, _ := HammingEncode(message)
	for index := range encoded {
		corrupted := []byte(encoded)
		if corrupted[index] == '0' {
			corrupted[index] = '1'
		} else {
			corrupted[index] = '0'
		}
		result := HammingDecode(string(corrupted), len(message))
		if !result.Valid || !result.Corrected || result.DataBits != message || result.Syndrome != index+1 {
			t.Fatalf("fallo al corregir la posicion %d: %+v", index+1, result)
		}
	}
}

func TestHammingOneErrorPerBlockIsIndependentlyCorrected(t *testing.T) {
	// Un SEC generico sobre todo el frame solo tolera UN bit volteado en
	// todo el mensaje; por bloques, cada bloque de 7 tolera el suyo.
	message := "10110010" // dos bloques de 4 bits
	encoded, _ := HammingEncode(message)
	corrupted := []byte(encoded)
	corrupted[2] = flip(corrupted[2]) // bloque 0
	corrupted[9] = flip(corrupted[9]) // bloque 1
	result := HammingDecode(string(corrupted), len(message))
	if !result.Valid || !result.Corrected || result.DataBits != message {
		t.Fatalf("fallo al corregir errores independientes por bloque: %+v", result)
	}
}

func flip(bit byte) byte {
	if bit == '0' {
		return '1'
	}
	return '0'
}

func TestHammingGenericLength(t *testing.T) {
	message := strings.Repeat("101001", 167)[:1000]
	encoded, err := HammingEncode(message)
	if err != nil {
		t.Fatal(err)
	}
	blocks := (len(message) + blockDataBits - 1) / blockDataBits
	if len(encoded) != blocks*blockCodeBits {
		t.Fatalf("longitud = %d", len(encoded))
	}
	result := HammingDecode(encoded, len(message))
	if !result.Valid || result.DataBits != message {
		t.Fatalf("round trip invalido: %+v", result)
	}
}
