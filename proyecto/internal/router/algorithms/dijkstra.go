// Package algorithms: rutas mas cortas sobre el grafo construido con los LSA.
package algorithms

import "container/heap"

// PathResult es el costo total y el primer salto desde el origen a un destino.
type PathResult struct {
	Cost    int
	NextHop string
}

type queueItem struct {
	cost int
	node string
}

type priorityQueue []queueItem

func (pq priorityQueue) Len() int            { return len(pq) }
func (pq priorityQueue) Less(i, j int) bool  { return pq[i].cost < pq[j].cost }
func (pq priorityQueue) Swap(i, j int)       { pq[i], pq[j] = pq[j], pq[i] }
func (pq *priorityQueue) Push(x interface{}) { *pq = append(*pq, x.(queueItem)) }
func (pq *priorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	*pq = old[:n-1]
	return item
}

// ShortestPaths calcula costo y primer salto de source hacia cada nodo alcanzable.
func ShortestPaths(graph map[string]map[string]int, source string) map[string]PathResult {
	costs := map[string]int{source: 0}
	firstHop := map[string]string{}
	visited := map[string]bool{}

	pq := &priorityQueue{{0, source}}
	heap.Init(pq)

	for pq.Len() > 0 {
		item := heap.Pop(pq).(queueItem)
		if visited[item.node] {
			continue
		}
		visited[item.node] = true
		for neighbor, weight := range graph[item.node] {
			newCost := item.cost + weight
			currentCost, known := costs[neighbor]
			if !known || newCost < currentCost {
				costs[neighbor] = newCost
				// El primer salto se hereda del nodo previo, salvo el caso base.
				if item.node == source {
					firstHop[neighbor] = neighbor
				} else {
					firstHop[neighbor] = firstHop[item.node]
				}
				heap.Push(pq, queueItem{newCost, neighbor})
			}
		}
	}

	results := make(map[string]PathResult, len(costs))
	for node, cost := range costs {
		if node == source {
			continue
		}
		results[node] = PathResult{Cost: cost, NextHop: firstHop[node]}
	}
	return results
}
