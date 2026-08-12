package experiment

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"
)

const (
	svgWidth   = 1200.0
	svgHeight  = 680.0
	plotLeft   = 105.0
	plotTop    = 90.0
	plotRight  = 1140.0
	plotBottom = 570.0
)

var palette = []string{"#440154", "#3b528b", "#21918c", "#5ec962", "#fde725"}

func svgStart(title string) *strings.Builder {
	builder := &strings.Builder{}
	fmt.Fprintf(builder, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %.0f %.0f" role="img" aria-label="%s">`, svgWidth, svgHeight, html.EscapeString(title))
	builder.WriteString(`<rect width="100%" height="100%" fill="white"/>`)
	builder.WriteString(`<style>text{font-family:"Segoe UI",Arial,sans-serif;fill:#202124}.title{font-size:28px;font-weight:600}.label{font-size:17px}.tick{font-size:14px}.legend{font-size:15px}.grid{stroke:#dfe3e6;stroke-width:1}.axis{stroke:#202124;stroke-width:2}</style>`)
	fmt.Fprintf(builder, `<text class="title" x="%.0f" y="45" text-anchor="middle">%s</text>`, svgWidth/2, html.EscapeString(title))
	return builder
}

func axes(builder *strings.Builder, yTicks int, yLabel func(int) string, xLabels []string, xTitle, yTitle string) []float64 {
	width := plotRight - plotLeft
	height := plotBottom - plotTop
	for index := 0; index <= yTicks; index++ {
		y := plotBottom - height*float64(index)/float64(yTicks)
		fmt.Fprintf(builder, `<line class="grid" x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f"/>`, plotLeft, y, plotRight, y)
		fmt.Fprintf(builder, `<text class="tick" x="%.1f" y="%.1f" text-anchor="end">%s</text>`, plotLeft-12, y+5, html.EscapeString(yLabel(index)))
	}
	fmt.Fprintf(builder, `<line class="axis" x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f"/><line class="axis" x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f"/>`, plotLeft, plotBottom, plotRight, plotBottom, plotLeft, plotTop, plotLeft, plotBottom)
	xValues := make([]float64, len(xLabels))
	for index, label := range xLabels {
		x := plotLeft + width*float64(index)/float64(max(len(xLabels)-1, 1))
		xValues[index] = x
		fmt.Fprintf(builder, `<text class="tick" x="%.1f" y="%.1f" text-anchor="middle">%s</text>`, x, plotBottom+28, html.EscapeString(label))
	}
	fmt.Fprintf(builder, `<text class="label" x="%.1f" y="650" text-anchor="middle">%s</text>`, (plotLeft+plotRight)/2, html.EscapeString(xTitle))
	fmt.Fprintf(builder, `<text class="label" transform="translate(28 %.1f) rotate(-90)" text-anchor="middle">%s</text>`, (plotTop+plotBottom)/2, html.EscapeString(yTitle))
	return xValues
}

func saveSVG(path string, builder *strings.Builder) error {
	builder.WriteString(`</svg>`)
	return os.WriteFile(path, []byte(builder.String()), 0o644)
}

func writeOverhead(path string, config Config, results []Result) error {
	title := "Overhead de Hamming(7,4) segun el tamano del frame"
	builder := svgStart(title)
	labels := make([]string, len(config.FrameSizes))
	values := make([]float64, 0, len(config.FrameSizes))
	for index, size := range config.FrameSizes {
		labels[index] = fmt.Sprintf("%d B", size)
		for _, result := range results {
			if result.FrameBytes == size && result.ErrorRate == config.ErrorRates[0] {
				values = append(values, result.OverheadPct)
				break
			}
		}
	}
	xValues := axes(builder, 4, func(index int) string { return fmt.Sprintf("%d%%", index*20) }, labels, "Tamano nominal del frame DATA", "Redundancia / datos originales")
	var points strings.Builder
	for index, value := range values {
		y := plotBottom - (plotBottom-plotTop)*value/80
		fmt.Fprintf(&points, "%.1f,%.1f ", xValues[index], y)
	}
	fmt.Fprintf(builder, `<polyline points="%s" fill="none" stroke="#007c83" stroke-width="4"/>`, points.String())
	for index, value := range values {
		y := plotBottom - (plotBottom-plotTop)*value/80
		fmt.Fprintf(builder, `<circle cx="%.1f" cy="%.1f" r="7" fill="#007c83"/><text class="tick" x="%.1f" y="%.1f" text-anchor="middle">%.1f%%</text>`, xValues[index], y, xValues[index], y-14, value)
	}
	return saveSVG(path, builder)
}

func writeRecovery(path string, config Config, results []Result) error {
	title := "Recuperacion exacta de Hamming(7,4) por frame y ruido"
	builder := svgStart(title)
	labels := make([]string, len(config.ErrorRates))
	for index, rate := range config.ErrorRates {
		labels[index] = fmt.Sprintf("%g%%", rate*100)
	}
	xValues := axes(builder, 5, func(index int) string { return fmt.Sprintf("%d%%", index*20) }, labels, "Probabilidad de flip por bit", "Frames recuperados exactamente")
	for sizeIndex, size := range config.FrameSizes {
		var points strings.Builder
		for rateIndex, rate := range config.ErrorRates {
			value := 0.0
			for _, result := range results {
				if result.FrameBytes == size && result.ErrorRate == rate {
					value = result.ExactRecoveryRate
					break
				}
			}
			y := plotBottom - (plotBottom-plotTop)*value
			fmt.Fprintf(&points, "%.1f,%.1f ", xValues[rateIndex], y)
		}
		color := palette[sizeIndex%len(palette)]
		fmt.Fprintf(builder, `<polyline points="%s" fill="none" stroke="%s" stroke-width="4"/>`, points.String(), color)
		legendX := 930.0
		legendY := 112.0 + float64(sizeIndex)*25
		fmt.Fprintf(builder, `<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="4"/><text class="legend" x="%.1f" y="%.1f">%d B</text>`, legendX, legendY, legendX+28, legendY, color, legendX+36, legendY+5, size)
	}
	return saveSVG(path, builder)
}

func writeOutcomes(path string, config Config, results []Result) error {
	targetSize := 128
	if len(config.FrameSizes) > 0 {
		targetSize = config.FrameSizes[len(config.FrameSizes)/2]
	}
	title := fmt.Sprintf("Resultados por condicion para frames de %d B", targetSize)
	builder := svgStart(title)
	labels := make([]string, len(config.ErrorRates))
	for index, rate := range config.ErrorRates {
		labels[index] = fmt.Sprintf("%g%%", rate*100)
	}
	xValues := axes(builder, 5, func(index int) string { return fmt.Sprintf("%d%%", index*20) }, labels, "Probabilidad de flip por bit", "Proporcion de transmisiones")
	barWidth := 100.0
	for rateIndex, rate := range config.ErrorRates {
		var selected Result
		for _, result := range results {
			if result.FrameBytes == targetSize && result.ErrorRate == rate {
				selected = result
				break
			}
		}
		values := []float64{selected.ExactRecoveryRate, selected.RejectedRate, selected.SilentCorruptionRate}
		colors := []string{"#2a9d8f", "#8d99ae", "#c44536"}
		bottom := plotBottom
		for index, value := range values {
			height := (plotBottom - plotTop) * value
			bottom -= height
			fmt.Fprintf(builder, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="%s"/>`, xValues[rateIndex]-barWidth/2, bottom, barWidth, height, colors[index])
		}
	}
	legendLabels := []string{"Recuperado exacto", "Rechazado", "Corrupcion silenciosa"}
	legendColors := []string{"#2a9d8f", "#8d99ae", "#c44536"}
	for index, label := range legendLabels {
		x := 300.0 + float64(index)*250
		fmt.Fprintf(builder, `<rect x="%.1f" y="610" width="22" height="16" fill="%s"/><text class="legend" x="%.1f" y="624">%s</text>`, x, legendColors[index], x+30, label)
	}
	return saveSVG(path, builder)
}

func writeFigures(root string, config Config, results []Result) error {
	writers := []struct {
		name string
		fn   func(string, Config, []Result) error
	}{
		{"overhead.svg", writeOverhead},
		{"recovery_rate.svg", writeRecovery},
		{"outcomes.svg", writeOutcomes},
	}
	for _, writer := range writers {
		if err := writer.fn(filepath.Join(root, writer.name), config, results); err != nil {
			return err
		}
	}
	return nil
}
