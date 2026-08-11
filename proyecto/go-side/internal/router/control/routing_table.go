package control

import (
	"encoding/csv"
	"os"
	"strconv"
)

// Route es una fila de <nodo>_tabla_enrutamiento.csv.
type Route struct {
	Destination  string
	NextHopIP    string
	NextHopPort  int
	Cost         int
}

// WriteRoutingTable escribe el CSV producido por el plano de control.
func WriteRoutingTable(path string, routes map[string]Route) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if err := writer.Write([]string{"destination", "next_hop_ip", "next_hop_port", "cost"}); err != nil {
		return err
	}
	for _, route := range routes {
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
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	routes := make(map[string]Route, len(rows))
	for _, row := range rows[1:] { // saltar encabezado
		port, _ := strconv.Atoi(row[2])
		cost, _ := strconv.Atoi(row[3])
		routes[row[0]] = Route{Destination: row[0], NextHopIP: row[1], NextHopPort: port, Cost: cost}
	}
	return routes, nil
}
