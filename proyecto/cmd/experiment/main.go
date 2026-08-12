// Command experiment regenera datos y graficas de ruido/Hamming usando Go.
package main

import (
	"flag"
	"fmt"
	"os"

	"cc3067/lab3/internal/experiment"
)

func main() {
	output := flag.String("output", "experiments", "directorio para data/ y figures/")
	flag.Parse()

	config := experiment.DefaultConfig()
	results, err := experiment.Generate(*output, config)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("Generadas %d condiciones y %d transmisiones con Go.\n", len(results), len(results)*config.Trials)
	fmt.Printf("Resultados: %s\n", *output)
}
