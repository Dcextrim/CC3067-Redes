package link

import (
	"fmt"
	"strings"

	"cc3067/lab2/go-side/internal/algorithms"
)

const (
	Hamming = "hamming"
	CRC32   = "crc32"
)

type IntegrityResult struct {
	OK          bool
	MessageBits string
	Corrected   bool
	Error       string
	Syndrome    int
}

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

func VerificarIntegridad(frameBits, algorithm string, messageLength int) IntegrityResult {
	selected, err := NormalizeAlgorithm(algorithm)
	if err != nil {
		return IntegrityResult{Error: err.Error()}
	}
	if selected == Hamming {
		result := algorithms.HammingDecode(frameBits, messageLength)
		return IntegrityResult{result.Valid, result.DataBits, result.Corrected, result.Error, result.Syndrome}
	}
	ok, data, message := algorithms.CRC32Verify(frameBits, messageLength)
	return IntegrityResult{OK: ok, MessageBits: data, Error: message}
}

func CorregirMensaje(frameBits string, messageLength int) IntegrityResult {
	result := algorithms.HammingDecode(frameBits, messageLength)
	return IntegrityResult{result.Valid, result.DataBits, result.Corrected, result.Error, result.Syndrome}
}

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
