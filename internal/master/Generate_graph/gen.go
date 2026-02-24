package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

var nodeTypes = []string{"Router", "Database", "Client", "Server"}
var statuses = []string{"active", "inactive", "maintenance"}
var relationTypes = []string{"CONNECTED_TO", "DEPENDS_ON"}
var suffixes = []string{"_A", "_B", "_C", "_D", "_E", "_F"}

func main() {
	rand.Seed(time.Now().UnixNano())

	const nodeCount = 50
	const edgeCount = 100

	var nodes []string
	for i := 1; i <= nodeCount; i++ {
		name := fmt.Sprintf("Node_%d%s", i, suffixes[rand.Intn(len(suffixes))])
		nodeType := nodeTypes[rand.Intn(len(nodeTypes))]
		status := statuses[rand.Intn(len(statuses))]
		priority := rand.Intn(10) + 1

		node := fmt.Sprintf("(n%d:Node {id: %d, name: '%s', type: '%s', status: '%s', priority: %d})",
			i, i, name, nodeType, status, priority)
		nodes = append(nodes, node)
	}

	fmt.Println("CREATE")
	fmt.Println(strings.Join(nodes, ",\n") + ";")
	fmt.Println()

	edges := make(map[string]bool)
	printed := 0

	for i := 1; i < nodeCount; i++ {
		rel := relationTypes[rand.Intn(len(relationTypes))]
		fmt.Printf("MATCH (a:Node {id: \"%d\"}), (b:Node {id: \"%d\"}) CREATE (a)-[:%s]->(b);\n", i, i+1, rel)
		edges[fmt.Sprintf("%d-%d", i, i+1)] = true
		printed++
	}

	for printed < edgeCount {
		a := rand.Intn(nodeCount) + 1
		b := rand.Intn(nodeCount) + 1
		if a == b {
			continue
		}
		key := fmt.Sprintf("%d-%d", a, b)
		if edges[key] || edges[fmt.Sprintf("%d-%d", b, a)] {
			continue
		}
		rel := relationTypes[rand.Intn(len(relationTypes))]
		fmt.Printf("MATCH (a:Node {id: \"%d\"}), (b:Node {id: \"%d\"}) CREATE (a)-[:%s]->(b);\n", a, b, rel)
		edges[key] = true
		printed++
	}
}
