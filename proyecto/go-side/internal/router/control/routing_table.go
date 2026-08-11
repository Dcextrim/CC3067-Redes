package control

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
)

// Route es una fila de <nodo>_tabla_enrutamiento.csv.
type Route struct {
	Destination string
	NextHopIP   string
	NextHopPort int
	Cost        int
}

var routingTableHeader = []string{"destination", "next_hop_ip", "next_hop_port", "cost"}

// WriteRoutingTable escribe el CSV producido por el plano de control. La tabla
// se construye en un archivo temporal y se reemplaza de forma atomica para que
// el plano de datos nunca observe un CSV parcialmente escrito.
func WriteRoutingTable(path string, routes map[string]Route) error {
	directory := filepath.Dir(path)
	file, err := os.CreateTemp(directory, ".routing-table-*.tmp")
	if err != nil {
		return err
	}
	temporaryPath := file.Name()
	keepTemporary := true
	defer func() {
		_ = file.Close()
		if keepTemporary {
			_ = os.Remove(temporaryPath)
		}
	}()

	writer := csv.NewWriter(file)
	if err := writer.Write(routingTableHeader); err != nil {
		return err
	}
	destinations := make([]string, 0, len(routes))
	for destination := range routes {
		destinations = append(destinations, destination)
	}
	sort.Strings(destinations)
	for _, destination := range destinations {
		route := routes[destination]
		row := []string{
			route.Destination,
			route.NextHopIP,
			strconv.Itoa(route.NextHopPort),
			strconv.Itoa(route.Cost),
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return err
	}
	keepTemporary = false
	return nil
}

// ReadRoutingTable lee un CSV previamente escrito por WriteRoutingTable.
func ReadRoutingTable(path string) (map[string]Route, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	header, err := reader.Read()
	if err != nil {
		return nil, err
	}
	if len(header) != len(routingTableHeader) {
		return nil, fmt.Errorf("encabezado de tabla de ruteo invalido")
	}
	for index := range routingTableHeader {
		if header[index] != routingTableHeader[index] {
			return nil, fmt.Errorf("encabezado de tabla de ruteo invalido")
		}
	}

	routes := map[string]Route{}
	for line := 2; ; line++ {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("fila %d: %w", line, err)
		}
		if len(row) != len(routingTableHeader) {
			return nil, fmt.Errorf("fila %d: se esperaban 4 columnas", line)
		}
		port, err := strconv.Atoi(row[2])
		if err != nil {
			return nil, fmt.Errorf("fila %d: puerto invalido: %w", line, err)
		}
		cost, err := strconv.Atoi(row[3])
		if err != nil {
			return nil, fmt.Errorf("fila %d: costo invalido: %w", line, err)
		}
		routes[row[0]] = Route{Destination: row[0], NextHopIP: row[1], NextHopPort: port, Cost: cost}
	}
	return routes, nil
}
