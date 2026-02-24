package master

import (
	"container/internal/common"
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
)

type MasterMetaData struct {
	mu               sync.Mutex
	PartitionInfoMap map[int]*common.PartitionInfo
	EdgeInfoMap      map[int]map[int][]int
	NumEdges         int
	NumVertices      int
	NumPartitions    int
	WorkerAddrs      map[int]string
	Threshold        float64
	FreeWorkers      []string

	ToleranceParameter   float64
	DestinationParameter float64
}

func NewMasterMetaData() *MasterMetaData {
	return &MasterMetaData{
		PartitionInfoMap:     make(map[int]*common.PartitionInfo),
		EdgeInfoMap:          make(map[int]map[int][]int),
		WorkerAddrs:          make(map[int]string),
		Threshold:            common.ThresholdFactor,
		FreeWorkers:          []string{},
		ToleranceParameter:   10.0,
		DestinationParameter: 5.0,
	}
}

func ProcessAddVertex2PC(coord *Coordinator, md *MasterMetaData, v *common.Vertex, query string) error {
	vNoEdges := &common.Vertex{
		ID:     v.ID,
		Edges:  []int{},
		Labels: v.Labels,
		Type:   v.Type,
	}
	matchQuery := buildVertexExistenceQuery(v)
	results, err := ProcessMatchQuery(coord, md, matchQuery, nil)
	if err != nil {
		return fmt.Errorf("existence check failed: %v", err)
	}

	if len(results) > 0 {
		return fmt.Errorf("vertex already exists: %v", results[0]["n.vertex_id"])
	}

	var partIdx int
	var createOrPromoteQuery string

	thresholdValue := calculateAddingThreshold(md)

	if float64(common.MAXCAP) <= thresholdValue {

		if md.NumPartitions < len(md.WorkerAddrs) {
			idx, addr, found := assignNewPartition(md)
			if found {
				log.Printf("Adding new partition %d on worker %s", idx, addr)
				partIdx = idx
			} else {

				partIdx = handleVertexAssignmentWithoutNewPartition(md, vNoEdges)
			}
		} else {
			partIdx = handleVertexAssignmentWithoutNewPartition(md, vNoEdges)
		}
	} else {
		partIdx = handleVertexAssignmentWithoutNewPartition(md, vNoEdges)
	}

	createOrPromoteQuery = buildCreateVertexQuery(v)

	addr := md.WorkerAddrs[partIdx]
	msg := common.Message{
		Type:      common.AddVertex,
		Vertex:    vNoEdges,
		Query:     createOrPromoteQuery,
		Partition: partIdx,
	}

	if err := coord.Run2PC(context.Background(), msg, []string{addr}); err != nil {
		return fmt.Errorf("2PC failed for adding vertex %d on partition %d: %w", v.ID, partIdx, err)
	}

	updateSummary(md, partIdx, vNoEdges)

	return nil
}

func calculateAddingThreshold(md *MasterMetaData) float64 {
	if md.NumPartitions == 0 {
		return 0
	}
	return float64(md.NumEdges) / float64(md.NumPartitions)
}
func handleVertexAssignmentWithoutNewPartition(md *MasterMetaData, v *common.Vertex) int {
	if md.NumPartitions > 0 {
		sigma := calculateImbalance(md)
		if sigma > md.Threshold*float64(common.MAXCAP) {

			return assignVertexByConnections(md, v)
		} else {

			return findMinimumLoadPartition(md)
		}
	} else {

		idx, _, found := assignNewPartition(md)
		if found {
			return idx
		}
		return 0
	}
}

func assignVertexByConnections(md *MasterMetaData, v *common.Vertex) int {
	maxConnections := -1
	selectedPartition := -1
	partitionsWithMaxConnections := []int{}

	for partIdx, partition := range md.PartitionInfoMap {
		connectionCount := 0

		for _, edge := range v.Edges {
			for _, vertexInPartition := range partition.Vertices {
				if vertexInPartition == edge {
					connectionCount++
				}
			}
		}

		if connectionCount > maxConnections {
			maxConnections = connectionCount
			selectedPartition = partIdx
			partitionsWithMaxConnections = []int{partIdx}
		} else if connectionCount == maxConnections && connectionCount > 0 {
			partitionsWithMaxConnections = append(partitionsWithMaxConnections, partIdx)
		}
	}

	if len(partitionsWithMaxConnections) > 1 {
		return findMinimumLoadPartitionFromList(md, partitionsWithMaxConnections)
	}

	if maxConnections == -1 {
		return findMinimumLoadPartition(md)
	}

	return selectedPartition
}
func findMinimumLoadPartitionFromList(md *MasterMetaData, partitions []int) int {
	if len(partitions) == 0 {
		return findMinimumLoadPartition(md)
	}

	minLoad := int(^uint(0) >> 1)
	selectedPartition := partitions[0]

	for _, partIdx := range partitions {
		if partition, ok := md.PartitionInfoMap[partIdx]; ok {
			load := len(partition.Vertices)
			if load < minLoad {
				minLoad = load
				selectedPartition = partIdx
			}
		}
	}

	return selectedPartition
}

func ProcessAddEdge2PC(coord *Coordinator, md *MasterMetaData, v1, v2 int, query string, edgeLabels map[string]interface{}, edgeType string, vertex1 *common.Vertex, vertex2 *common.Vertex) error {

	v1Id := -1
	v2Id := -1

	v1Exists, err := checkVertexExistence(coord, md, v1)
	if err != nil {
		return err
	}

	v2Exists, err := checkVertexExistence(coord, md, v2)
	if err != nil {
		return err
	}
	if v2Exists == nil {
		return fmt.Errorf("destination vertex %d does not exist", v2)
	}

	v1Id, err = extractVertexIDFromResults(v1Exists, "n")
	if err != nil {

		return fmt.Errorf("balumba")
	}
	v2Id, err = extractVertexIDFromResults(v2Exists, "n")
	if err != nil {

		return fmt.Errorf("balumba")
	}
	if v1Id != -1 && v2Id != -1 {
		edgeExists, err := checkEdgeExistence(coord, md, v1Id, v2Id, edgeType, edgeLabels)
		if edgeExists {
			return fmt.Errorf("edge between %d and %d already exists", v1, v2)
		}
		if err != nil {
			return fmt.Errorf("something bad", v1, v2)
		}

	}
	md.mu.Lock()
	p1 := findVertexPartition(md, v1Id)
	p2 := findVertexPartition(md, v2Id)

	isV1Ghost := false
	isV2Ghost := false
	if p1 != -1 {
		if pi, ok := md.PartitionInfoMap[p1]; ok {
			if pi.Ghosts != nil {
				_, isV1Ghost = pi.Ghosts[v1]
			}
		}
	}

	if p2 != -1 {
		if pi, ok := md.PartitionInfoMap[p2]; ok {
			if pi.Ghosts != nil {
				_, isV2Ghost = pi.Ghosts[v2]
			}
		}
	}
	md.mu.Unlock()

	if p1 == -1 || p2 == -1 {
		return fmt.Errorf("failed to determine partitions for vertices %d or %d", v1, v2)
	}

	generateEdgeQuery, _ := buildCreateEdgeQuery(query, v1Id, v2Id)

	if p1 == p2 {

		md.mu.Lock()
		addEdgeToMeta(md.PartitionInfoMap[p1], v1, v2)
		md.EdgeInfoMap[p1][v1] = appendUnique(md.EdgeInfoMap[p1][v1], v2)
		md.NumEdges++
		md.mu.Unlock()

		msg := common.Message{
			Type:      common.AddEdge,
			Edge:      &common.Edge{V1: v1, V2: v2},
			Query:     generateEdgeQuery,
			Partition: p1,
		}
		return coord.Run2PC(context.Background(), msg, []string{md.WorkerAddrs[p1]})

	} else {

		var targetPartition int

		ghostVertex := &common.Vertex{}

		if !isV1Ghost && !isV2Ghost {

			targetPartition = p1

			ghostVertex = &common.Vertex{
				ID:     v2,
				Labels: vertex2.Labels,
			}

		} else if !isV1Ghost && isV2Ghost {

			targetPartition = p1

			ghostVertex = &common.Vertex{
				ID:     v2,
				Labels: vertex2.Labels,
			}

		} else if isV1Ghost && !isV2Ghost {

			targetPartition = p2

			ghostVertex = &common.Vertex{
				ID:     v1,
				Labels: vertex1.Labels,
			}

		} else {

			return fmt.Errorf("cannot create edge between two ghost vertices %d and %d", v1, v2)
		}

		md.mu.Lock()
		isGhostInTarget := false
		isRealInTarget := false
		if pi, ok := md.PartitionInfoMap[targetPartition]; ok {

			if pi.Ghosts != nil {
				_, isGhostInTarget = pi.Ghosts[ghostVertex.ID]
			}

			for _, vid := range pi.Vertices {
				if vid == ghostVertex.ID {
					isRealInTarget = true
					break
				}
			}
		}
		md.mu.Unlock()

		if !isGhostInTarget && !isRealInTarget {
			ghostInfo, err := getVertexDetails(coord, md, ghostVertex.ID)
			if err != nil {

				ghostInfo = &common.Vertex{ID: ghostVertex.ID, Labels: ghostInfo.Labels, Type: ""}
			}

			addGhostVertex(md, targetPartition, ghostVertex.ID)

			err = addGhostToWorker(coord, md, targetPartition, ghostVertex.ID, ghostInfo.Labels, ghostInfo.Type)
			if err != nil {
				return fmt.Errorf("failed to add ghost vertex %d to partition %d: %w", ghostVertex, targetPartition, err)
			}

		}

		md.mu.Lock()
		addEdgeToMeta(md.PartitionInfoMap[targetPartition], v1, v2)
		if md.EdgeInfoMap[targetPartition] == nil {
			md.EdgeInfoMap[targetPartition] = make(map[int][]int)
		}
		md.EdgeInfoMap[targetPartition][v1] = appendUnique(md.EdgeInfoMap[targetPartition][v1], v2)
		md.NumEdges++
		md.mu.Unlock()

		msg := common.Message{
			Type:      common.AddEdge,
			Edge:      &common.Edge{V1: v1, V2: v2},
			Query:     generateEdgeQuery,
			Partition: targetPartition,
		}

		err := coord.Run2PC(context.Background(), msg, []string{md.WorkerAddrs[targetPartition]})
		if err != nil {
			return fmt.Errorf("failed to add edge %d->%d in partition %d: %w", v1, v2, targetPartition, err)
		}
	}

	return nil
}

func buildVertexExistenceQuery(vertex *common.Vertex) string {
	var conditions []string
	for k, v := range vertex.Labels {
		conditions = append(conditions, fmt.Sprintf("%s: \"%v\"", k, v))
	}

	return fmt.Sprintf("MATCH (n:%s{%s}) RETURN n", vertex.Type, strings.Join(conditions, ", "))
}

func checkVertexExistence(coord *Coordinator, md *MasterMetaData, vertexID int) ([]map[string]interface{}, error) {
	query := fmt.Sprintf("MATCH (n {vertex_id: %d}) RETURN n", vertexID)
	results, err := ProcessMatchQuery(coord, md, query, nil)
	if err != nil {
		return nil, err
	}
	return results, nil
}

func checkEdgeExistence(coord *Coordinator, md *MasterMetaData, v1, v2 int, edgeType string, labels map[string]interface{}) (bool, error) {
	query := fmt.Sprintf("MATCH (a {vertex_id: %d})-[r]->(b {vertex_id: %d}) RETURN r", v1, v2)
	results, err := ProcessMatchQuery(coord, md, query, nil)
	return len(results) > 0, err
}

func buildCreateVertexQuery(v *common.Vertex) string {
	labelStr := ""
	if v.Type != "" {
		labelStr = ":" + v.Type
	}

	props := make([]string, 0, len(v.Labels)+1)
	props = append(props, fmt.Sprintf("vertex_id: %d", v.ID))
	for k, val := range v.Labels {
		props = append(props, fmt.Sprintf("%s: \"%v\"", k, val))
	}
	fmt.Println(fmt.Sprintf("CREATE (n%s {%s})", labelStr, strings.Join(props, ", ")))
	return fmt.Sprintf("CREATE (n%s {%s})", labelStr, strings.Join(props, ", "))
}

func ProcessDeleteVertex2PCAll(coord *Coordinator, md *MasterMetaData, v *common.Vertex) error {

	var matchQuery string
	if v.Type != "" && len(v.Labels) > 0 {

		labelStr := ":" + v.Type
		whereConditions := []string{fmt.Sprintf("n.vertex_id = %d", v.ID)}
		for k, val := range v.Labels {
			whereConditions = append(whereConditions, fmt.Sprintf("n.%s = \"%v\"", k, val))
		}
		matchQuery = fmt.Sprintf("MATCH (n%s {vertex_id: %d}) RETURN n", labelStr, v.ID)
	} else if v.Type != "" {

		matchQuery = fmt.Sprintf("MATCH (n:%s {vertex_id: %d}) RETURN n", v.Type, v.ID)
	} else if v.ID > -1 {

		matchQuery = fmt.Sprintf("MATCH (n {vertex_id: %d}) RETURN n", v.ID)
	} else {
		matchQuery = fmt.Sprintf("MATCH (n) RETURN n")
	}

	results, err := ProcessMatchQuery(coord, md, matchQuery, nil)
	if err != nil {
		return fmt.Errorf("failed to find vertex %d for deletion: %w", v.ID, err)
	}
	if len(results) == 0 {
		return fmt.Errorf("vertex %d not found for deletion", v.ID)
	}

	v_Id, err := extractVertexIDFromResultsArray(results, "n")
	if err != nil {

		return fmt.Errorf("balumba")
	}

	for _, id := range v_Id {
		var deleteQuery string
		deleteQuery = fmt.Sprintf("MATCH (n) WHERE n.vertex_id = %d DETACH DELETE n", id)
		var affectedPartitions []int

		for pID := range md.PartitionInfoMap {

			part := md.PartitionInfoMap[pID]
			found := false

			for _, vid := range part.Vertices {
				if vid == id {
					found = true
					break
				}
			}

			if !found {
				if _, exists := part.Ghosts[id]; exists {
					found = true
				}
			}

			if found {
				affectedPartitions = append(affectedPartitions, pID)
			}
		}

		if len(affectedPartitions) == 0 {
			return fmt.Errorf("vertex %d not found in any partition metadata", v.ID)
		}

		var participants []string

		for i := range md.WorkerAddrs {
			participants = append(participants, md.WorkerAddrs[i])
		}

		msg := common.Message{
			Type:   common.DeleteVertex,
			Vertex: &common.Vertex{ID: id},
			Query:  deleteQuery,
		}

		if err := coord.Run2PC(context.Background(), msg, participants); err != nil {
			return fmt.Errorf("2PC failed for deleting vertex %d: %w", v.ID, err)
		}
		p1 := findVertexPartition(md, id)

		deleteVertex(md, &common.Vertex{ID: id}, p1)
		for _, pID := range affectedPartitions {

			if part, exists := md.PartitionInfoMap[p1]; exists {
				if _, isGhost := part.Ghosts[id]; isGhost {
					removeGhostVertex(md, pID, id)
					newGV := make([]int, 0, len(part.GhostVertices)-1)
					for _, gvid := range part.GhostVertices {
						if gvid != id {
							newGV = append(newGV, gvid)
						}
					}
					part.GhostVertices = newGV
				}
			}
		}

	}

	return nil
}

func extractVertexIDFromResultsArray(results []map[string]interface{}, variableName string) ([]int, error) {
	if len(results) == 0 {
		return nil, fmt.Errorf("no results found")
	}

	var ids []int

	for i, resultMap := range results {
		if value, ok := resultMap[variableName]; ok {
			var id int

			switch v := value.(type) {
			case int:
				id = v
			case string:
				if convertedID, convErr := strconv.Atoi(v); convErr == nil {
					id = convertedID
				} else {
					return nil, fmt.Errorf("cannot convert string %s to int at index %d", v, i)
				}
			case float64:
				id = int(v)
			default:
				return nil, fmt.Errorf("unexpected type %T for vertex ID at index %d", v, i)
			}

			ids = append(ids, id)
		} else {
			return nil, fmt.Errorf("key %s not found in results at index %d", variableName, i)
		}
	}

	return ids, nil
}
func ProcessDeleteEdge2PC(coord *Coordinator, md *MasterMetaData, v1, v2 int, edgeType string, edgeLabels map[string]interface{}) error {

	v1Exists, err := checkVertexExistence(coord, md, v1)
	if err != nil {
		return fmt.Errorf("failed to check existence of vertex %d: %w", v1, err)
	}
	if len(v1Exists) == 0 {
		return fmt.Errorf("source vertex %d does not exist", v1)
	}

	v2Exists, err := checkVertexExistence(coord, md, v2)
	if err != nil {
		return fmt.Errorf("failed to check existence of vertex %d: %w", v2, err)
	}
	if len(v2Exists) == 0 {
		return fmt.Errorf("destination vertex %d does not exist", v2)
	}

	v1Id, err := extractVertexIDFromResults(v1Exists, "n")
	if err != nil {
		return fmt.Errorf("failed to extract vertex ID for vertex %d: %w", v1, err)
	}

	v2Id, err := extractVertexIDFromResults(v2Exists, "n")
	if err != nil {
		return fmt.Errorf("failed to extract vertex ID for vertex %d: %w", v2, err)
	}

	var findEdgeQuery string
	if edgeType != "" && len(edgeLabels) > 0 {
		whereConditions := make([]string, 0, len(edgeLabels))
		for k, v := range edgeLabels {
			whereConditions = append(whereConditions, fmt.Sprintf("r.%s = \"%v\"", k, v))
		}
		findEdgeQuery = fmt.Sprintf(
			"MATCH (n{vertex_id: %d})-[r:%s]->(g{vertex_id: %d}) WHERE %s RETURN n, r, g",
			v1Id, edgeType, v2Id, strings.Join(whereConditions, " AND "))
	} else if edgeType != "" {
		findEdgeQuery = fmt.Sprintf(
			"MATCH (n{vertex_id: %d})-[r:%s]->(g{vertex_id: %d}) RETURN n,r,g",
			v1Id, edgeType, v2Id)
	} else {
		findEdgeQuery = fmt.Sprintf(
			"MATCH (n{vertex_id: %d})-[r]->(g{vertex_id: %d}) RETURN n, r, g", v1Id, v2Id)
	}

	edgeResults, err := ProcessMatchQuery(coord, md, findEdgeQuery, nil)
	if err != nil {
		return fmt.Errorf("failed to find edge %d->%d for deletion: %w", v1Id, v2Id, err)
	}

	if len(edgeResults) == 0 {
		return fmt.Errorf("edge %d->%d not found for deletion", v1Id, v2Id)
	}

	md.mu.Lock()
	p1 := findVertexPartition(md, v1Id)
	p2 := findVertexPartition(md, v2Id)

	var affectedPartitions []int

	if p1 != -1 {
		affectedPartitions = appendUnique(affectedPartitions, p1)
	}
	if p2 != -1 && p1 != p2 {
		affectedPartitions = appendUnique(affectedPartitions, p2)
	}

	for pi, part := range md.PartitionInfoMap {
		hasV1 := false
		hasV2 := false

		for _, vid := range part.Vertices {
			if vid == v1Id {
				hasV1 = true
				break
			}
		}
		if !hasV1 {
			if _, ok := part.Ghosts[v1Id]; ok {
				hasV1 = true
			}
		}

		for _, vid := range part.Vertices {
			if vid == v2Id {
				hasV2 = true
				break
			}
		}
		if !hasV2 {
			if _, ok := part.Ghosts[v2Id]; ok {
				hasV2 = true
			}
		}

		if hasV1 && hasV2 {
			affectedPartitions = appendUnique(affectedPartitions, pi)
		}
	}
	md.mu.Unlock()

	if len(affectedPartitions) == 0 {
		return fmt.Errorf("no partitions found for vertices %d and %d", v1Id, v2Id)
	}

	var deleteEdgeQuery string
	if edgeType != "" && len(edgeLabels) > 0 {
		whereConditions := make([]string, 0, len(edgeLabels))
		for k, v := range edgeLabels {
			whereConditions = append(whereConditions, fmt.Sprintf("r.%s = \"%v\"", k, v))
		}
		deleteEdgeQuery = fmt.Sprintf(
			"MATCH (n{vertex_id: %d})-[r:%s]->(g{vertex_id: %d}) WHERE %s DELETE r",
			v1Id, edgeType, v2Id, strings.Join(whereConditions, " AND "))
	} else if edgeType != "" {
		deleteEdgeQuery = fmt.Sprintf(
			"MATCH (n{vertex_id: %d})-[r:%s]->(g{vertex_id: %d}) DELETE r",
			v1Id, edgeType, v2Id)
	} else {
		deleteEdgeQuery = fmt.Sprintf(
			"MATCH (n{vertex_id: %d})-[r]->(g{vertex_id: %d}) DELETE r", v1Id, v2Id)
	}

	var participants []string
	for _, pID := range affectedPartitions {
		if addr, ok := md.WorkerAddrs[pID]; ok {
			participants = append(participants, addr)
		}
	}

	if len(participants) == 0 {
		return fmt.Errorf("no active workers found for affected partitions")
	}

	msg := common.Message{
		Type:  common.DeleteEdge,
		Edge:  &common.Edge{V1: v1Id, V2: v2Id, Type: edgeType, Labels: edgeLabels},
		Query: deleteEdgeQuery,
	}

	if err := coord.Run2PC(context.Background(), msg, participants); err != nil {
		return fmt.Errorf("2PC failed for deleting edge %d->%d: %w", v1Id, v2Id, err)
	}

	md.mu.Lock()
	defer md.mu.Unlock()

	for _, pID := range affectedPartitions {
		deleteEdge(md, pID, v1Id, v2Id)

	}

	if md.NumEdges > 0 {
		md.NumEdges--
	}

	return nil
}

func getVertexDetails(coord *Coordinator, md *MasterMetaData, nodeID int) (*common.Vertex, error) {
	pID := findVertexPartition(md, nodeID)
	if pID == -1 {
		return &common.Vertex{}, fmt.Errorf("partition not found for vertex %d", nodeID)
	}

	addr, ok := md.WorkerAddrs[pID]
	if !ok {
		return &common.Vertex{}, fmt.Errorf("worker address not found for partition %d", pID)
	}

	msg := common.Message{
		Type:   common.GetNodeDetails,
		Vertex: &common.Vertex{ID: nodeID},
	}

	reply, err := sendToWorker(addr, msg)
	if err != nil {
		return &common.Vertex{}, fmt.Errorf("worker communication error for node %d: %w", nodeID, err)
	}

	if len(reply.Results) == 0 || reply.Results[0]["n"] == nil {
		return &common.Vertex{}, fmt.Errorf("no node data received from worker for ID %d", nodeID)
	}

	nodeData, ok := reply.Results[0]["n"].(map[string]interface{})
	if !ok {
		return &common.Vertex{}, fmt.Errorf("unexpected node data format from worker: %T", reply.Results[0]["n"])
	}

	labelStr := "Vertex"
	if labels, ok := nodeData["Labels"].([]interface{}); ok && len(labels) > 0 {
		if l, ok := labels[0].(string); ok {
			labelStr = l
		}
	}

	props, _ := nodeData["Props"].(map[string]interface{})

	return &common.Vertex{
		ID:     nodeID,
		Type:   labelStr,
		Labels: props,
	}, nil
}
