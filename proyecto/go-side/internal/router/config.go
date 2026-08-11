package router

import (
	"encoding/json"
	"net"
	"os"
	"strconv"
)

// Neighbor es un vecino directo configurado para este nodo.
type Neighbor struct {
	IP   string `json:"ip"`
	Port int    `json:"port"`
	Cost int    `json:"cost"`
}

// ID identifica al vecino como ip:puerto (ver NodeID).
func (n Neighbor) ID() string { return NodeID(n.IP, n.Port) }

// AttachedHost es el cliente o servidor no-router que usa a este nodo como gateway.
type AttachedHost struct {
	Role string `json:"role"`
	IP   string `json:"ip"`
	Port int    `json:"port"`
	Cost int    `json:"cost"`
}

// ID identifica al host adjunto como ip:puerto.
func (h AttachedHost) ID() string { return NodeID(h.IP, h.Port) }

// NodeConfig es la configuracion completa de un nodo, leida de config.json.
type NodeConfig struct {
	Name         string        `json:"name"`
	IP           string        `json:"ip"`
	Port         int           `json:"port"`
	Neighbors    []Neighbor    `json:"neighbors"`
	AttachedHost *AttachedHost `json:"attached_host"`
}

// ID identifica a este nodo como ip:puerto. La IP sola no alcanza para
// pruebas locales (127.0.0.1 se comparte entre todos los nodos).
func (c NodeConfig) ID() string { return NodeID(c.IP, c.Port) }

// NodeID arma el identificador canonico ip:puerto de cualquier nodo.
func NodeID(ip string, port int) string { return net.JoinHostPort(ip, strconv.Itoa(port)) }

// LoadConfig lee y valida el config.json de un nodo.
func LoadConfig(path string) (NodeConfig, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return NodeConfig{}, err
	}
	var config NodeConfig
	if err := json.Unmarshal(raw, &config); err != nil {
		return NodeConfig{}, err
	}
	for i := range config.Neighbors {
		if config.Neighbors[i].Cost == 0 {
			config.Neighbors[i].Cost = 1
		}
	}
	if config.AttachedHost != nil && config.AttachedHost.Cost == 0 {
		config.AttachedHost.Cost = 1
	}
	return config, nil
}
