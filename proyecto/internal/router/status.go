package router

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"cc3067/lab3/internal/router/control"
	"cc3067/lab3/internal/router/forwarding"
)

// Endpoint HTTP de monitoreo, fuera del protocolo exigido por el laboratorio
// (los otros equipos lo usan desde un dashboard externo para inspeccionar
// nuestros nodos). No lleva autenticacion: cualquiera que alcance el puerto
// en la tailnet puede leer /status o disparar /send.

type statusNeighbor struct {
	ID           string  `json:"id"`
	IP           string  `json:"ip"`
	Port         int     `json:"port"`
	Cost         int     `json:"cost"`
	Alive        bool    `json:"alive"`
	LastHelloAge float64 `json:"last_hello_age"`
}

type statusAttachedHost struct {
	ID   string `json:"id"`
	IP   string `json:"ip"`
	Port int    `json:"port"`
	Cost int    `json:"cost"`
}

type statusRoute struct {
	Destination string `json:"destination"`
	NextHopIP   string `json:"next_hop_ip"`
	NextHopPort int    `json:"next_hop_port"`
	Cost        int    `json:"cost"`
}

type statusResponse struct {
	Kind         string                    `json:"kind"`
	Name         string                    `json:"name"`
	SelfID       string                    `json:"self_id"`
	IP           string                    `json:"ip"`
	Port         int                       `json:"port"`
	Converged    bool                      `json:"converged"`
	Seq          int                       `json:"seq"`
	Neighbors    []statusNeighbor          `json:"neighbors"`
	AttachedHost *statusAttachedHost       `json:"attached_host"`
	Graph        map[string]map[string]int `json:"graph"`
	RoutingTable []statusRoute             `json:"routing_table"`
}

func (n *Node) buildStatus() statusResponse {
	now := time.Now()

	n.mu.RLock()
	converged := n.converged
	neighbors := make([]statusNeighbor, 0, len(n.Config.Neighbors))
	for _, neighbor := range n.Config.Neighbors {
		id := neighbor.ID()
		age := -1.0
		if at, ok := n.lastHelloAt[id]; ok {
			age = now.Sub(at).Seconds()
		}
		neighbors = append(neighbors, statusNeighbor{
			ID: id, IP: neighbor.IP, Port: neighbor.Port, Cost: neighbor.Cost,
			Alive: n.activeNeighbors[id], LastHelloAge: age,
		})
	}
	routes := make([]statusRoute, 0, len(n.routes))
	for destination, route := range n.routes {
		routes = append(routes, statusRoute{
			Destination: destination, NextHopIP: route.NextHopIP,
			NextHopPort: route.NextHopPort, Cost: route.Cost,
		})
	}
	n.mu.RUnlock()

	sort.Slice(neighbors, func(i, j int) bool { return neighbors[i].ID < neighbors[j].ID })
	sort.Slice(routes, func(i, j int) bool { return routes[i].Destination < routes[j].Destination })

	var attachedHost *statusAttachedHost
	if n.Config.AttachedHost != nil {
		attachedHost = &statusAttachedHost{
			ID: n.Config.AttachedHost.ID(), IP: n.Config.AttachedHost.IP,
			Port: n.Config.AttachedHost.Port, Cost: n.Config.AttachedHost.Cost,
		}
	}

	return statusResponse{
		Kind: "router", Name: n.Config.Name, SelfID: n.Config.ID(),
		IP: n.Config.IP, Port: n.Config.Port,
		Converged: converged, Seq: n.store.CurrentSeq(),
		Neighbors: neighbors, AttachedHost: attachedHost,
		Graph: n.store.Snapshot(), RoutingTable: routes,
	}
}

type sendRequest struct {
	Target string `json:"target"`
	Text   string `json:"text"`
}

type sendResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

func (n *Node) handleSend(req sendRequest) sendResponse {
	if req.Target == "" || req.Text == "" {
		return sendResponse{OK: false, Error: "target y text son obligatorios"}
	}
	n.mu.RLock()
	route, ok := n.routes[req.Target]
	n.mu.RUnlock()
	if !ok {
		return sendResponse{OK: false, Error: fmt.Sprintf("sin ruta hacia %s", req.Target)}
	}
	envelope, err := forwarding.EncodeEnvelope(control.Payload{From: n.Config.ID(), To: req.Target, Msg: req.Text})
	if err != nil {
		return sendResponse{OK: false, Error: err.Error()}
	}
	n.SendMessage(route.NextHopIP, route.NextHopPort, envelope)
	return sendResponse{OK: true}
}

func withCORS(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		handler(w, r)
	}
}

// StartStatusServer levanta el endpoint de monitoreo /status y /send en
// segundo plano. No es parte del protocolo del laboratorio; port=0 lo
// desactiva.
func (n *Node) StartStatusServer(port int) {
	if port == 0 {
		return
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/status", withCORS(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(n.buildStatus())
	}))
	mux.HandleFunc("/send", withCORS(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req sendRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(sendResponse{OK: false, Error: "JSON invalido"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(n.handleSend(req))
	}))
	address := fmt.Sprintf(":%d", port)
	go func() {
		_ = http.ListenAndServe(address, mux)
	}()
}
