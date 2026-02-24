package master

import (
	"container/internal/common"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
)

func EnableCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	}
}

func ServeIndex(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/index.html")
}

func ServeStatic(w http.ResponseWriter, r *http.Request) {
	staticPath := "web"
	http.StripPrefix("/static/", http.FileServer(http.Dir(staticPath))).ServeHTTP(w, r)

}

func HandleInfo(md *MasterMetaData) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		md.mu.Lock()
		defer md.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(md)
	}
}

func HandleWorkerInfo(md *MasterMetaData) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		workerIDStr := r.URL.Query().Get("id")
		workerID, err := strconv.Atoi(workerIDStr)
		if err != nil {
			http.Error(w, "Invalid worker ID", http.StatusBadRequest)
			return
		}

		addr, ok := md.WorkerAddrs[workerID]
		if !ok {
			http.Error(w, "Worker not found", http.StatusNotFound)
			return
		}

		msg := common.Message{Type: common.GetWorkerInfo}
		reply, err := sendToWorker(addr, msg)
		if err != nil {
			http.Error(w, "Failed to get info from worker: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(reply.PartitionInfo)
	}
}

func HandleCypher(coord *Coordinator, md *MasterMetaData) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Cannot read body", http.StatusInternalServerError)
			return
		}
		cypherText := string(body)

		upperQuery := strings.ToUpper(cypherText)
		isMatch := strings.Contains(upperQuery, "MATCH")
		isCreate := strings.Contains(upperQuery, "CREATE")
		isDelete := strings.Contains(upperQuery, "DELETE")

		if isMatch && !isCreate && !isDelete {
			queryStructure, err := parseQuery(cypherText)
			if err != nil {
				http.Error(w, "Failed to parse query structure: "+err.Error(), http.StatusInternalServerError)
				return
			}

			results, err := ProcessMatchQuery(coord, md, cypherText, nil)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			graphData, err := GetGraphData(coord, md, results, queryStructure)
			if err != nil {
				http.Error(w, "Failed to build graph data for visualization: "+err.Error(), http.StatusInternalServerError)
				return
			}

			json.NewEncoder(w).Encode(graphData)
			return
		}

		_, status, err := ProcessAndAddCypherQuery(coord, md, cypherText)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]string{"status": status})
	}
}

func HandleUpload(coord *Coordinator, md *MasterMetaData) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		file, _, err := r.FormFile("queriesFile")
		if err != nil {
			http.Error(w, "File upload error", http.StatusBadRequest)
			return
		}
		defer file.Close()

		content, err := io.ReadAll(file)
		if err != nil {
			http.Error(w, "Cannot read file", http.StatusInternalServerError)
			return
		}

		cypherText := strings.TrimSpace(string(content))
		queries := strings.Split(cypherText, ";")

		var errors []string
		var status string
		for _, rawQuery := range queries {
			query := strings.TrimSpace(rawQuery)
			if query == "" {
				continue
			}
			if _, status, err = ProcessAndAddCypherQuery(coord, md, query); err != nil {
				errors = append(errors, err.Error())
			}
			fmt.Println(status)

		}
		if len(errors) > 0 {
			http.Error(w, strings.Join(errors, "\n"), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{
			"status":            "success",
			"message":           "File processed successfully!",
			"queries_processed": strconv.Itoa(len(queries)),
		})
	}
}

type GraphData struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

type GraphNode struct {
	ID    int                    `json:"id"`
	Label string                 `json:"label"`
	Props map[string]interface{} `json:"props"`
}

type GraphEdge struct {
	From  int                    `json:"from"`
	To    int                    `json:"to"`
	Label string                 `json:"label"`
	Props map[string]interface{} `json:"props"`
}

func convertToInt(val interface{}) (int, bool) {
	switch v := val.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return int(i), true
		}
	case string:
		if i, err := strconv.Atoi(v); err == nil {
			return i, true
		}
	}
	return 0, false
}

func GetGraphData(coord *Coordinator, md *MasterMetaData, results []map[string]interface{}, queryStructure *QueryStructure) (*GraphData, error) {
	nodeSet := make(map[int]GraphNode)
	edgeSet := make(map[string]GraphEdge)
	allNodeIDs := make(map[int]bool)

	for _, resultRow := range results {
		for _, value := range resultRow {
			if id, ok := convertToInt(value); ok {
				allNodeIDs[id] = true
			}
		}
	}

	for nodeID := range allNodeIDs {
		func(id int) {
			details, err := getNodeDetails(coord, md, id)
			if err != nil {
				nodeSet[id] = GraphNode{ID: id, Label: "Vertex", Props: map[string]interface{}{"error": "not found"}}
			} else {
				nodeSet[id] = details
			}
		}(nodeID)
	}

	for _, rel := range queryStructure.Relationships {
		fromVar := rel.FromNode.Variable
		toVar := rel.ToNode.Variable
		for _, resultRow := range results {
			fromIDVal, fromOk := resultRow[fromVar]
			toIDVal, toOk := resultRow[toVar]
			if fromOk && toOk {
				fromID, fromConverted := convertToInt(fromIDVal)
				toID, toConverted := convertToInt(toIDVal)
				if fromConverted && toConverted {
					edgeKey := fmt.Sprintf("%d->%d", fromID, toID)
					if _, exists := edgeSet[edgeKey]; !exists {
						details, err := getEdgeDetails(coord, md, fromID, toID)
						label := rel.RelType
						props := rel.Properties
						if err == nil && details != nil {
							if t, ok := details["type"].(string); ok && t != "" {
								label = t
							}
							if p, ok := details["props"].(map[string]interface{}); ok {
								props = p
							}
						} else {
							log.Printf("Could not get details for edge %d->%d: %v. Using info from query.", fromID, toID, err)
						}

						edgeSet[edgeKey] = GraphEdge{
							From:  fromID,
							To:    toID,
							Label: label,
							Props: props,
						}
					}
				}
			}
		}
	}

	graphData := &GraphData{
		Nodes: make([]GraphNode, 0, len(nodeSet)),
		Edges: make([]GraphEdge, 0, len(edgeSet)),
	}
	for _, node := range nodeSet {
		graphData.Nodes = append(graphData.Nodes, node)
	}
	for _, edge := range edgeSet {
		graphData.Edges = append(graphData.Edges, edge)
	}
	return graphData, nil
}

func getNodeDetails(coord *Coordinator, md *MasterMetaData, nodeID int) (GraphNode, error) {
	pID := findVertexPartition(md, nodeID)
	if pID == -1 {
		return GraphNode{}, fmt.Errorf("partition not found for vertex %d", nodeID)
	}

	addr, ok := md.WorkerAddrs[pID]
	if !ok {
		return GraphNode{}, fmt.Errorf("worker address not found for partition %d", pID)
	}

	msg := common.Message{
		Type:   common.GetNodeDetails,
		Vertex: &common.Vertex{ID: nodeID},
	}

	reply, err := sendToWorker(addr, msg)
	if err != nil {
		return GraphNode{}, fmt.Errorf("worker communication error for node %d: %w", nodeID, err)
	}

	if len(reply.Results) == 0 || reply.Results[0]["n"] == nil {
		return GraphNode{}, fmt.Errorf("no node data received from worker for ID %d", nodeID)
	}

	nodeData, ok := reply.Results[0]["n"].(map[string]interface{})
	if !ok {
		return GraphNode{}, fmt.Errorf("unexpected node data format from worker: %T", reply.Results[0]["n"])
	}

	labelStr := "Vertex"
	if labels, ok := nodeData["Labels"].([]interface{}); ok && len(labels) > 0 {
		if l, ok := labels[0].(string); ok {
			labelStr = l
		}
	}

	props, _ := nodeData["Props"].(map[string]interface{})

	return GraphNode{
		ID:    nodeID,
		Label: labelStr,
		Props: props,
	}, nil
}
func getEdgeDetails(coord *Coordinator, md *MasterMetaData, fromID, toID int) (map[string]interface{}, error) {
	partitionID := findVertexPartition(md, fromID)
	if partitionID == -1 {
		return nil, fmt.Errorf("partition for source vertex %d not found", fromID)
	}

	addr, ok := md.WorkerAddrs[partitionID]
	if !ok {
		return nil, fmt.Errorf("worker address for partition %d not found", partitionID)
	}

	msg := common.Message{
		Type: common.GetEdgeDetails,
		Edge: &common.Edge{V1: fromID, V2: toID},
	}

	reply, err := sendToWorker(addr, msg)
	if err != nil {
		return nil, fmt.Errorf("worker communication error for edge %d->%d: %w", fromID, toID, err)
	}

	if len(reply.Results) > 0 {
		details := reply.Results[0]
		if details != nil {
			return details, nil
		}
	}

	return nil, fmt.Errorf("no valid edge data returned from worker for edge %d->%d", fromID, toID)
}
