package worker

import (
	"container/internal/common"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"regexp"
	"sync"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type Partition struct {
	Info *common.PartitionInfo
	mu   sync.Mutex
}

var (
	partition Partition = Partition{
		Info: &common.PartitionInfo{
			Vertices:      []int{},
			Edges:         make(map[int][]int),
			Ghosts:        make(map[int]struct{}),
			GhostVertices: []int{},
		},
	}
	neo4jDriver neo4j.DriverWithContext
)

func InitNeo4j(uri, user, pass string) (neo4j.DriverWithContext, error) {
	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(user, pass, ""), func(c *neo4j.Config) {

	})
	if err != nil {
		return nil, fmt.Errorf("neo4j driver error: %w", err)
	}

	ctx := context.Background()
	if err = driver.VerifyConnectivity(ctx); err != nil {
		return nil, fmt.Errorf("neo4j connectivity error: %w", err)
	}

	neo4jDriver = driver
	return driver, nil
}

func logPartitionState() {
	for {
		time.Sleep(5 * time.Second)
		partition.mu.Lock()
		fmt.Println("----- Partition State -----")
		fmt.Printf("Vertices: %v\n", partition.Info.Vertices)
		fmt.Println("Edges:")
		for id, edges := range partition.Info.Edges {
			fmt.Printf("   %d -> %v\n", id, edges)
		}
		fmt.Printf("Ghost vertices: %v\n", partition.Info.GhostVertices)
		fmt.Printf("Ghost map: %v\n", partition.Info.Ghosts)
		fmt.Println("----------------------------")
		partition.mu.Unlock()
	}
}

func StartCohortServer(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("start listen error: %w", err)
	}
	defer ln.Close()
	log.Printf("Cohort server started at %s", addr)
	go logPartitionState()
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Accept error: %v", err)
			continue
		}
		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {
	defer conn.Close()
	dec := json.NewDecoder(conn)
	enc := json.NewEncoder(conn)
	var msg common.Message
	if err := dec.Decode(&msg); err != nil {
		log.Printf("Decode error: %v", err)
		return
	}
	var cypherErr error
	var results []map[string]interface{}
	var partialMatches []common.LocalPartialMatch

	switch msg.Type {
	case common.GetWorkerInfo:
		partition.mu.Lock()
		infoCopy := &common.PartitionInfo{
			Vertices:      append([]int(nil), partition.Info.Vertices...),
			Edges:         make(map[int][]int),
			GhostVertices: append([]int(nil), partition.Info.GhostVertices...),
			Ghosts:        make(map[int]struct{}),
		}
		for k, v := range partition.Info.Edges {
			infoCopy.Edges[k] = append([]int(nil), v...)
		}
		for k, v := range partition.Info.Ghosts {
			infoCopy.Ghosts[k] = v
		}
		partition.mu.Unlock()

		response := common.Message{Type: common.Ack, PartitionInfo: infoCopy}
		enc.Encode(response)
		return

	case common.GetNodeDetails:
		query := fmt.Sprintf("MATCH (n) WHERE n.vertex_id = %d AND NOT n:Ghost RETURN n", msg.Vertex.ID)
		rawResults, err := runCypher(query, nil)
		if err != nil {
			cypherErr = err
		} else {
			if len(rawResults) > 0 {
				if node, ok := rawResults[0]["n"].(neo4j.Node); ok {
					results = []map[string]interface{}{{"n": convertNeo4jNodeToMap(node)}}
				}
			}
		}
	case common.GetEdgeDetails:
		query := fmt.Sprintf("MATCH ({vertex_id: %d})-[r]->({vertex_id: %d}) RETURN r", msg.Edge.V1, msg.Edge.V2)
		rawResults, err := runCypher(query, nil)
		if err != nil {
			cypherErr = err
		} else {
			if len(rawResults) > 0 {
				if rel, ok := rawResults[0]["r"].(neo4j.Relationship); ok {
					edgeDetails := map[string]interface{}{
						"type":  rel.Type,
						"props": rel.Props,
					}
					results = []map[string]interface{}{edgeDetails}
				}
			}
		}
	case common.AddVertex:
		partition.mu.Lock()
		vid := msg.Vertex.ID

		log.Printf("Worker adding new regular vertex %d.", vid)
		partition.Info.Vertices = append(partition.Info.Vertices, vid)
		if _, ok := partition.Info.Edges[vid]; !ok {
			partition.Info.Edges[vid] = msg.Vertex.Edges
		}
		if msg.Query != "" {
			partition.Info.QueryCache = append(partition.Info.QueryCache, msg.Query)
		}
		partition.mu.Unlock()

		if msg.Query != "" {
			results, cypherErr = runCypher(msg.Query, nil)
			if cypherErr != nil {
				enc.Encode(common.Message{Type: "Error", Result: cypherErr.Error()})
				return
			}
		}
	case common.AddEdge:
		partition.mu.Lock()
		v1 := msg.Edge.V1
		v2 := msg.Edge.V2

		if _, ok := partition.Info.Edges[v1]; !ok {
			partition.Info.Edges[v1] = []int{}
		}
		found := false
		for _, id := range partition.Info.Edges[v1] {
			if id == v2 {
				found = true
				break
			}
		}
		if !found {
			partition.Info.Edges[v1] = append(partition.Info.Edges[v1], v2)
		}

		if msg.Query != "" {
			partition.Info.QueryCache = append(partition.Info.QueryCache, msg.Query)
		}
		partition.mu.Unlock()

		if msg.Query != "" {
			results, cypherErr = runCypher(msg.Query, nil)
			if cypherErr != nil {
				enc.Encode(common.Message{Type: "Error", Result: cypherErr.Error()})
				return
			}
		}
	case common.DeleteVertex:
		partition.mu.Lock()
		newVerts := []int{}
		for _, vid := range partition.Info.Vertices {
			if vid != msg.Vertex.ID {
				newVerts = append(newVerts, vid)
			}
		}
		partition.Info.Vertices = newVerts
		delete(partition.Info.Edges, msg.Vertex.ID)
		for from, tos := range partition.Info.Edges {
			newTos := []int{}
			for _, to := range tos {
				if to != msg.Vertex.ID {
					newTos = append(newTos, to)
				}
			}
			partition.Info.Edges[from] = newTos
		}

		if _, exists := partition.Info.Ghosts[msg.Vertex.ID]; exists {
			delete(partition.Info.Ghosts, msg.Vertex.ID)
			newGV := make([]int, 0, len(partition.Info.GhostVertices)-1)
			for _, v := range partition.Info.GhostVertices {
				if v != msg.Vertex.ID {
					newGV = append(newGV, v)
				}
			}
			partition.Info.GhostVertices = newGV
		}
		partition.mu.Unlock()

		if msg.Query != "" {
			results, cypherErr = runCypher(msg.Query, nil)
			if cypherErr != nil {
				enc.Encode(common.Message{Type: "Error", Result: cypherErr.Error()})
				return
			}
		}
	case common.DeleteEdge:
		partition.mu.Lock()
		if edges, ok := partition.Info.Edges[msg.Edge.V1]; ok {
			for i, id := range edges {
				if id == msg.Edge.V2 {
					partition.Info.Edges[msg.Edge.V1] = append(edges[:i], edges[i+1:]...)
					break
				}
			}
		}
		partition.mu.Unlock()

		if msg.Query != "" {
			results, cypherErr = runCypher(msg.Query, nil)
			if cypherErr != nil {
				enc.Encode(common.Message{Type: "Error", Result: cypherErr.Error()})
				return
			}
		}
	case common.UpdateGhostStatus:

		partition.mu.Lock()

		partition.mu.Unlock()

		if msg.Query != "" {

			results, cypherErr = runCypher(msg.Query, map[string]interface{}{"id": msg.Vertex.ID})
			if cypherErr != nil {
				enc.Encode(common.Message{Type: "Error", Result: cypherErr.Error()})
				return
			}
		}
	case common.AddGhostVertex:
		partition.mu.Lock()
		vid := msg.Vertex.ID
		if _, exists := partition.Info.Ghosts[vid]; !exists {
			partition.Info.Ghosts[vid] = struct{}{}
			partition.Info.GhostVertices = append(partition.Info.GhostVertices, vid)
			log.Println("Query", msg.Query)
			log.Printf("Worker added ghost vertex %d to local metadata.", vid)

			if _, ok := partition.Info.Edges[vid]; !ok {
				partition.Info.Edges[vid] = []int{}
			}
		}
		partition.mu.Unlock()

		if msg.Query != "" {
			results, cypherErr = runCypher(msg.Query, nil)
			if cypherErr != nil {
				enc.Encode(common.Message{Type: "Error", Result: cypherErr.Error()})
				return
			}
		}
	case common.RemoveGhostVertex:
		partition.mu.Lock()
		if _, exists := partition.Info.Ghosts[msg.Vertex.ID]; exists {
			delete(partition.Info.Ghosts, msg.Vertex.ID)
			newGV := make([]int, 0, len(partition.Info.GhostVertices)-1)
			for _, v := range partition.Info.GhostVertices {
				if v != msg.Vertex.ID {
					newGV = append(newGV, v)
				}
			}
			partition.Info.GhostVertices = newGV
			log.Printf("Worker removed ghost vertex %d from local metadata.", msg.Vertex.ID)
		}
		partition.mu.Unlock()

		if msg.Query != "" {
			results, cypherErr = runCypher(msg.Query, nil)
			if cypherErr != nil {
				enc.Encode(common.Message{Type: "Error", Result: cypherErr.Error()})
				return
			}
		}
	case common.Match:
		if msg.Query != "" {
			results, cypherErr = runCypher(msg.Query, nil)
			if cypherErr != nil {
				enc.Encode(common.Message{Type: "Error", Result: cypherErr.Error()})
				return
			}
		}
	case common.MatchVertex:
		Query := fmt.Sprintf("MATCH (N) WHERE N.vertex_id = %d RETURN N", msg.Vertex.ID)
		results, cypherErr = runCypher(Query, nil)
		if cypherErr != nil {
			enc.Encode(common.Message{Type: "Error", Result: cypherErr.Error()})
			return
		}
	case common.ExecQuery:
		if msg.Query != "" {
			results, cypherErr = runCypher(msg.Query, nil)
			if cypherErr != nil {
				enc.Encode(common.Message{Type: "Error", Result: cypherErr.Error()})
				return
			}
		}

	case common.FindLocalPartialMatches:
		if len(msg.Queries) > 0 {
			var batchResults []common.LocalPartialMatch
			for _, q := range msg.Queries {
				matches, err := findLocalPartialMatches(q, msg.Partition)
				if err != nil {
					cypherErr = err
					break
				}
				batchResults = append(batchResults, matches...)
			}
			if cypherErr == nil {
				partialMatches = batchResults
			}
		} else if msg.Query != "" {
			partialMatches, cypherErr = findLocalPartialMatches(msg.Query, msg.Partition)
		}

		if cypherErr != nil {
			enc.Encode(common.Message{Type: "Error", Result: cypherErr.Error()})
			return
		}
	default:
		log.Printf("Unknown message type: %s", msg.Type)
		enc.Encode(common.Message{Type: "Error", Result: fmt.Sprintf("Unknown message type: %s", msg.Type)})
		return
	}

	response := common.Message{Type: common.Ack}
	if len(partialMatches) > 0 {
		response.Type = common.PartialMatchResult
		response.PartialMatches = partialMatches
	}
	if len(results) > 0 {
		response.Results = results
	}

	enc.Encode(response)
}

func convertNeo4jNodeToMap(node neo4j.Node) map[string]interface{} {
	props := make(map[string]interface{})
	for k, v := range node.Props {
		props[k] = v
	}

	var vertexID int64 = -1
	if id, ok := node.Props["vertex_id"].(int64); ok {
		vertexID = id
	}

	return map[string]interface{}{
		"ID":     vertexID,
		"Labels": node.Labels,
		"Props":  props,
	}
}

func runCypher(cypher string, params map[string]interface{}) ([]map[string]interface{}, error) {
	if neo4jDriver == nil {
		return nil, fmt.Errorf("neo4j driver not initialized")
	}
	ctx := context.Background()
	session := neo4jDriver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	log.Printf("Executing Cypher query: %s", cypher)
	result, err := session.Run(ctx, cypher, params)
	if err != nil {
		log.Printf("Cypher query failed: %v", err)
		return nil, err
	}

	var res []map[string]interface{}
	for result.Next(ctx) {
		record := result.Record()
		rec := make(map[string]interface{})
		for _, key := range record.Keys {
			val, _ := record.Get(key)
			rec[key] = val
		}
		res = append(res, rec)
	}

	if result.Err() != nil {
		log.Printf("Error processing Cypher result: %v", result.Err())
		return nil, result.Err()
	}
	return res, nil
}

func getVertexIDFromNode(node neo4j.Node) int {
	if vertexID, exists := node.Props["vertex_id"]; exists {
		if id, ok := vertexID.(int64); ok {
			return int(id)
		}
	}
	return -1
}
func parseNodeVariablesFromQuery(query string) (string, string, string, error) {
	re := regexp.MustCompile(`MATCH\s*\(([^):{\s]+)[^)]*\)\s*-\s*\[([^\]:{}\s]*)?[^\]]*\]\s*->\s*\(([^):{\s]+)[^)]*\)`)
	matches := re.FindStringSubmatch(query)
	if len(matches) < 4 {
		return "", "", "", nil
	}
	return matches[1], matches[3], matches[2], nil
}

func isRecordValidForPartition(fromID, toID int) bool {
	partition.mu.Lock()
	defer partition.mu.Unlock()

	isFromInternal := false
	for _, vID := range partition.Info.Vertices {
		if vID == fromID {
			isFromInternal = true
			break
		}
	}

	if !isFromInternal {
		return false
	}

	isToInternal := false
	for _, vID := range partition.Info.Vertices {
		if vID == toID {
			isToInternal = true
			break
		}
	}

	_, isToGhost := partition.Info.Ghosts[toID]

	return isToInternal || isToGhost
}

func findLocalPartialMatches(query string, fragmentID int) ([]common.LocalPartialMatch, error) {
	if neo4jDriver == nil {
		return nil, fmt.Errorf("neo4j driver not initialized")
	}

	fromVar, toVar, relVar, err := parseNodeVariablesFromQuery(query)
	if err != nil {
		log.Printf("Could not parse query variables: %v", err)
	}

	ctx := context.Background()
	session := neo4jDriver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	log.Printf("Finding local partial matches with query: %s in fragment %d", query, fragmentID)

	result, err := session.Run(ctx, query, nil)
	if err != nil {
		log.Printf("Failed to find local partial matches: %v", err)
		return nil, err
	}

	var partialMatches []common.LocalPartialMatch

	for result.Next(ctx) {
		record := result.Record()

		if fromVar != "" && toVar != "" {
			fromVal, fromExists := record.Get(fromVar)
			toVal, toExists := record.Get(toVar)

			if fromExists && toExists {
				if fromNode, ok1 := fromVal.(neo4j.Node); ok1 {
					if toNode, ok2 := toVal.(neo4j.Node); ok2 {
						fromID := getVertexIDFromNode(fromNode)
						toID := getVertexIDFromNode(toNode)
						if !isRecordValidForPartition(fromID, toID) {
							continue
						}
					}
				}
			}
		}

		partialMatch := extractPartialMatchFromRecord(record, fragmentID, fromVar, toVar, relVar)
		partialMatches = append(partialMatches, partialMatch)
	}

	if result.Err() != nil {
		log.Printf("Error processing partial match results: %v", result.Err())
		return nil, result.Err()
	}

	return partialMatches, nil
}
func extractPartialMatchFromRecord(record *neo4j.Record, fragmentID int, fromVar, toVar, relVar string) common.LocalPartialMatch {
	matchFunction := make(map[string]interface{})
	internalVerticesSet := make(map[int]bool)
	extendedVerticesSet := make(map[int]bool)

	var internalVertices []int
	var extendedVertices []int
	var crossingEdges []common.Edge

	for _, key := range record.Keys {
		value, exists := record.Get(key)
		if !exists {
			continue
		}

		if node, ok := value.(neo4j.Node); ok {
			vertexID := getVertexIDFromNode(node)
			if _, exists := matchFunction[key]; !exists {
				matchFunction[key] = vertexID
			}

			if !internalVerticesSet[vertexID] && !extendedVerticesSet[vertexID] {
				isGhost := false
				for _, label := range node.Labels {
					if label == "Ghost" {
						isGhost = true
						break
					}
				}

				if isGhost {
					extendedVerticesSet[vertexID] = true
					extendedVertices = append(extendedVertices, vertexID)
				} else {
					internalVerticesSet[vertexID] = true
					internalVertices = append(internalVertices, vertexID)
				}
			}
		}
	}

	if relVar != "" {
		if relValue, exists := record.Get(relVar); exists {
			if _, ok := relValue.(neo4j.Relationship); ok {
				fromID := matchFunction[fromVar].(int)
				toID := matchFunction[toVar].(int)

				isFromInternal := internalVerticesSet[fromID]
				isToExtended := extendedVerticesSet[toID]

				if isFromInternal && isToExtended {
					crossingEdges = append(crossingEdges, common.Edge{V1: fromID, V2: toID})
				}
			}
		}
	}

	return common.LocalPartialMatch{
		FragmentID:       fragmentID,
		MatchFunction:    matchFunction,
		CrossingEdges:    crossingEdges,
		InternalVertices: internalVertices,
		ExtendedVertices: extendedVertices,
	}
}
