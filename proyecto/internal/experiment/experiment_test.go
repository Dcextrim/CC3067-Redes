package experiment

import (
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFrameForSizeHasExactByteLength(t *testing.T) {
	for _, size := range []int{64, 128, 256, 512, 1024} {
		frame, err := frameForSize(size)
		if err != nil {
			t.Fatal(err)
		}
		if len([]byte(frame)) != size {
			t.Fatalf("frame de %d bytes genero %d bytes", size, len([]byte(frame)))
		}
	}
}

func TestRunIsDeterministic(t *testing.T) {
	config := Config{Seed: 7, Trials: 10, FrameSizes: []int{64}, ErrorRates: []float64{0.01}}
	first, err := Run(config)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Run(config)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 1 || len(second) != 1 || first[0] != second[0] {
		t.Fatalf("resultados no deterministas: %+v / %+v", first, second)
	}
}

func TestGenerateWritesDataAndSVGFigures(t *testing.T) {
	root := t.TempDir()
	config := Config{Seed: 1, Trials: 2, FrameSizes: []int{64}, ErrorRates: []float64{0}}
	if _, err := Generate(root, config); err != nil {
		t.Fatal(err)
	}
	paths := []string{
		filepath.Join(root, "data", "experiment_results.csv"),
		filepath.Join(root, "data", "experiment_summary.json"),
		filepath.Join(root, "figures", "overhead.svg"),
		filepath.Join(root, "figures", "recovery_rate.svg"),
		filepath.Join(root, "figures", "outcomes.svg"),
	}
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasSuffix(path, ".svg") {
			if !strings.Contains(string(raw), "<svg") {
				t.Fatalf("%s no contiene una grafica SVG", path)
			}
			decoder := xml.NewDecoder(strings.NewReader(string(raw)))
			for {
				if _, err := decoder.Token(); err != nil {
					if err == io.EOF {
						break
					}
					t.Fatalf("%s no es XML valido: %v", path, err)
				}
			}
		}
	}
}
