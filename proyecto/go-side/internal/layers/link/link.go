// Package link selecciona, calcula y verifica la redundancia de cada trama.
package link

import (
	"fmt"
	"strings"

	"cc3067/lab2/go-side/internal/algorithms"
)

const (
	// Hamming y CRC32 son los identificadores canonicos compartidos con Python.
	Hamming = "hamming"
	CRC32   = "crc32"
)

// IntegrityResult unifica la salida de algoritmos detectores y correctores.
type IntegrityResult struct {
	OK          bool
	MessageBits string
	Corrected   bool
	Error       string
	Syndrome    int
}

// NormalizeAlgorithm convierte opciones de consola al identificador del protocolo.
func NormalizeAlgorithm(value string) (string, error) {
	normalized := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(value)), "-", "")
	switch normalized {
	case "1", "hamming":
		return Hamming, nil
	case "2", "crc", "crc32":
		return CRC32, nil
	default:
		return "", fmt.Errorf("algoritmo invalido; use hamming o crc32")
	}
}

// CalcularIntegridad agrega la redundancia definida por el algoritmo elegido.
func CalcularIntegridad(messageBits, algorithm string) (string, error) {
	selected, err := NormalizeAlgorithm(algorithm)
	if err != nil {
		return "", err
	}
	if selected == Hamming {
		return algorithms.HammingEncode(messageBits)
	}
	return algorithms.CRC32Encode(messageBits)
}

// VerificarIntegridad acepta, rechaza o corrige una trama recibida.
func VerificarIntegridad(frameBits, algorithm string, messageLength int) IntegrityResult {
	selected, err := NormalizeAlgorithm(algorithm)
	if err != nil {
		return IntegrityResult{Error: err.Error()}
	}
	if selected == Hamming {
		return CorregirMensaje(frameBits, messageLength)
	}
	ok, data, message := algorithms.CRC32Verify(frameBits, messageLength)
	return IntegrityResult{OK: ok, MessageBits: data, Error: message}
}

// CorregirMensaje expone explicitamente el servicio corrector de Hamming.
func CorregirMensaje(frameBits string, messageLength int) IntegrityResult {
	result := algorithms.HammingDecode(frameBits, messageLength)
	return IntegrityResult{result.Valid, result.DataBits, result.Corrected, result.Error, result.Syndrome}
}

// RedundancyBits calcula el overhead teorico sin construir la trama.
func RedundancyBits(messageLength int, algorithm string) (int, error) {
	selected, err := NormalizeAlgorithm(algorithm)
	if err != nil {
		return 0, err
	}
	if selected == Hamming {
		return algorithms.RequiredParityBits(messageLength)
	}
	return 32, nil
}
