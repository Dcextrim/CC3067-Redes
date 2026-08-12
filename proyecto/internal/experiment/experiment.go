// Package experiment ejecuta simulaciones reproducibles de Hamming(7,4)
// sometido a un canal con flips independientes por bit.
package experiment

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"cc3067/lab3/internal/router/algorithms"
	"cc3067/lab3/internal/router/codec"
)

// Config define una corrida completa y reproducible.
type Config struct {
	Seed       int64
	Trials     int
	FrameSizes []int
	ErrorRates []float64
}

// DefaultConfig replica las 25 condiciones usadas para el reporte.
func DefaultConfig() Config {
	return Config{
		Seed:       3067,
		Trials:     300,
		FrameSizes: []int{64, 128, 256, 512, 1024},
		ErrorRates: []float64{0, 0.001, 0.005, 0.01, 0.02},
	}
}

// Result resume una combinacion de tamano de frame y tasa de error.
type Result struct {
	FrameBytes           int
	MessageBits          int
	ErrorRate            float64
	Trials               int
	EncodedBits          int
	RedundancyBits       int
	OverheadPct          float64
	MeanFlips            float64
	ExactRecoveryRate    float64
	RejectedRate         float64
	SilentCorruptionRate float64
	CorrectedRate        float64
}

type summary struct {
	Seed                 int64     `json:"seed"`
	TrialsPerCombination int       `json:"trials_per_combination"`
	Combinations         int       `json:"combinations"`
	TotalTransmissions   int       `json:"total_transmissions"`
	FrameSizesBytes      []int     `json:"frame_sizes_bytes"`
	ErrorRates           []float64 `json:"error_rates"`
	Algorithm            string    `json:"algorithm"`
	NoiseModel           string    `json:"noise_model"`
	Generator            string    `json:"generator"`
}

func frameForSize(size int) (string, error) {
	type payload struct {
		From string `json:"from"`
		To   string `json:"to"`
		Msg  string `json:"msg"`
	}
	base := payload{From: "100.0.0.1:5000", To: "100.0.0.2:5000"}
	empty, err := json.Marshal(base)
	if err != nil {
		return "", err
	}
	paddingSize := size - len(empty)
	if paddingSize < 0 {
		return "", fmt.Errorf("el frame solicitado (%d bytes) es menor que el JSON minimo (%d bytes)", size, len(empty))
	}
	var padding strings.Builder
	padding.Grow(paddingSize)
	for index := 0; index < paddingSize; index++ {
		padding.WriteByte(byte('A' + index%26))
	}
	base.Msg = padding.String()
	raw, err := json.Marshal(base)
	if err != nil {
		return "", err
	}
	if len(raw) != size {
		return "", fmt.Errorf("el frame generado mide %d bytes, se esperaban %d", len(raw), size)
	}
	return string(raw), nil
}

func validateConfig(config Config) error {
	if config.Trials <= 0 {
		return fmt.Errorf("trials debe ser positivo")
	}
	if len(config.FrameSizes) == 0 || len(config.ErrorRates) == 0 {
		return fmt.Errorf("se requiere al menos un tamano y una tasa de error")
	}
	for _, size := range config.FrameSizes {
		if size <= 0 {
			return fmt.Errorf("los tamanos de frame deben ser positivos")
		}
	}
	for _, rate := range config.ErrorRates {
		if rate < 0 || rate > 1 {
			return fmt.Errorf("las tasas de error deben estar entre 0 y 1")
		}
	}
	return nil
}

// Run ejecuta todas las transmisiones con una unica fuente pseudoaleatoria.
func Run(config Config) ([]Result, error) {
	if err := validateConfig(config); err != nil {
		return nil, err
	}
	source := rand.New(rand.NewSource(config.Seed))
	results := make([]Result, 0, len(config.FrameSizes)*len(config.ErrorRates))
	for _, size := range config.FrameSizes {
		frame, err := frameForSize(size)
		if err != nil {
			return nil, err
		}
		messageBits, err := codec.CodificarMensaje(frame)
		if err != nil {
			return nil, err
		}
		encoded, err := algorithms.HammingEncode(messageBits)
		if err != nil {
			return nil, err
		}
		redundancy := len(encoded) - len(messageBits)
		for _, rate := range config.ErrorRates {
			exact, rejected, silent, corrected, totalFlips := 0, 0, 0, 0, 0
			for trial := 0; trial < config.Trials; trial++ {
				noisy, flips, err := algorithms.AplicarRuido(encoded, rate, source)
				if err != nil {
					return nil, err
				}
				totalFlips += flips
				decoded := algorithms.HammingDecode(noisy, len(messageBits))
				switch {
				case !decoded.Valid:
					rejected++
				case decoded.DataBits == messageBits:
					exact++
				default:
					silent++
				}
				if decoded.Corrected {
					corrected++
				}
			}
			results = append(results, Result{
				FrameBytes:           size,
				MessageBits:          len(messageBits),
				ErrorRate:            rate,
				Trials:               config.Trials,
				EncodedBits:          len(encoded),
				RedundancyBits:       redundancy,
				OverheadPct:          100 * float64(redundancy) / float64(len(messageBits)),
				MeanFlips:            float64(totalFlips) / float64(config.Trials),
				ExactRecoveryRate:    float64(exact) / float64(config.Trials),
				RejectedRate:         float64(rejected) / float64(config.Trials),
				SilentCorruptionRate: float64(silent) / float64(config.Trials),
				CorrectedRate:        float64(corrected) / float64(config.Trials),
			})
		}
	}
	return results, nil
}

var csvHeader = []string{
	"frame_bytes", "message_bits", "error_rate", "trials", "encoded_bits",
	"redundancy_bits", "overhead_pct", "mean_flips", "exact_recovery_rate",
	"rejected_rate", "silent_corruption_rate", "corrected_rate",
}

func number(value float64) string {
	return strconv.FormatFloat(value, 'g', -1, 64)
}

func writeCSV(path string, results []Result) error {
	stream, err := os.Create(path)
	if err != nil {
		return err
	}
	defer stream.Close()
	writer := csv.NewWriter(stream)
	if err := writer.Write(csvHeader); err != nil {
		return err
	}
	for _, result := range results {
		record := []string{
			strconv.Itoa(result.FrameBytes), strconv.Itoa(result.MessageBits), number(result.ErrorRate),
			strconv.Itoa(result.Trials), strconv.Itoa(result.EncodedBits), strconv.Itoa(result.RedundancyBits),
			number(result.OverheadPct), number(result.MeanFlips), number(result.ExactRecoveryRate),
			number(result.RejectedRate), number(result.SilentCorruptionRate), number(result.CorrectedRate),
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}

func writeSummary(path string, config Config, results []Result) error {
	value := summary{
		Seed: config.Seed, TrialsPerCombination: config.Trials,
		Combinations: len(results), TotalTransmissions: len(results) * config.Trials,
		FrameSizesBytes: config.FrameSizes, ErrorRates: config.ErrorRates,
		Algorithm: "Hamming(7,4) por bloques", NoiseModel: "flip independiente por bit",
		Generator: "Go",
	}
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}

// Generate ejecuta la simulacion y escribe datos y graficas bajo outputRoot.
func Generate(outputRoot string, config Config) ([]Result, error) {
	results, err := Run(config)
	if err != nil {
		return nil, err
	}
	dataRoot := filepath.Join(outputRoot, "data")
	figuresRoot := filepath.Join(outputRoot, "figures")
	if err := os.MkdirAll(dataRoot, 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(figuresRoot, 0o755); err != nil {
		return nil, err
	}
	if err := writeCSV(filepath.Join(dataRoot, "experiment_results.csv"), results); err != nil {
		return nil, err
	}
	if err := writeSummary(filepath.Join(dataRoot, "experiment_summary.json"), config, results); err != nil {
		return nil, err
	}
	if err := writeFigures(figuresRoot, config, results); err != nil {
		return nil, err
	}
	return results, nil
}
