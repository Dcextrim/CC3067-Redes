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

func TestHammingCorrectsEverySingleBit(t *testing.T) {
	message := "10110010"
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

func TestHammingGenericLength(t *testing.T) {
	message := strings.Repeat("101001", 167)[:1000]
	encoded, err := HammingEncode(message)
	if err != nil {
		t.Fatal(err)
	}
	r, _ := RequiredParityBits(len(message))
	if len(encoded) != len(message)+r {
		t.Fatalf("longitud = %d", len(encoded))
	}
	result := HammingDecode(encoded, len(message))
	if !result.Valid || result.DataBits != message {
		t.Fatalf("round trip invalido: %+v", result)
	}
}
