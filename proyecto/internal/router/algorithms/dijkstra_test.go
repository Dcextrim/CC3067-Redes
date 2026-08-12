package algorithms

import "testing"

func TestShortestPathsFromU(t *testing.T) {
	graph := map[string]map[string]int{
		"U": {"X": 1, "Y": 1, "C": 1},
		"X": {"U": 1, "Y": 1},
		"Y": {"U": 1, "X": 1, "Z": 1},
		"Z": {"Y": 1, "S": 1},
		"C": {"U": 1},
		"S": {"Z": 1},
	}
	result := ShortestPaths(graph, "U")

	cases := map[string]PathResult{
		"C": {Cost: 1, NextHop: "C"},
		"X": {Cost: 1, NextHop: "X"},
		"Y": {Cost: 1, NextHop: "Y"},
		"Z": {Cost: 2, NextHop: "Y"},
		"S": {Cost: 3, NextHop: "Y"},
	}
	for node, want := range cases {
		got, ok := result[node]
		if !ok || got != want {
			t.Fatalf("%s: got %+v want %+v (ok=%v)", node, got, want, ok)
		}
	}
}

func TestUnreachableNodeExcluded(t *testing.T) {
	graph := map[string]map[string]int{
		"A": {"B": 1},
		"B": {"A": 1},
		"C": {},
	}
	result := ShortestPaths(graph, "A")
	if _, ok := result["C"]; ok {
		t.Fatalf("C no deberia ser alcanzable")
	}
}
