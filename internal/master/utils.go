package master

import (
	"container/internal/common"
	"context"
	"fmt"
	"math"
	"strings"
)

func calculateImbalance(md *MasterMetaData) float64 {
	if md.NumPartitions == 0 {
		return 0
	}

	loads := make([]float64, 0, md.NumPartitions)
	totalLoad := 0.0

	for _, partition := range md.PartitionInfoMap {
		load := float64(len(partition.Vertices))
		loads = append(loads, load)
		totalLoad += load
	}

	if len(loads) == 0 {
		return 0
	}

	meanLoad := totalLoad / float64(len(loads))
	variance := 0.0

	for _, load := range loads {
		variance += math.Pow(load-meanLoad, 2)
	}

	return math.Sqrt(variance / float64(len(loads)))
}

func findMinimumLoadPartition(md *MasterMetaData) int {
	if md.NumPartitions == 0 {
		return -1
	}

	minLoad := int(^uint(0) >> 1)
	selectedPartition := -1

	for partIdx, partition := range md.PartitionInfoMap {
		load := len(partition.Vertices)
		if load < minLoad {
			minLoad = load
			selectedPartition = partIdx
		}
	}

	return selectedPartition
}

func addEdgeToMeta(pi *common.PartitionInfo, vfrom, vto int) {
	found := false
	for _, eid := range pi.Edges[vfrom] {
		if eid == vto {
			found = true
			break
		}
	}
	if !found {
		if pi.Edges[vfrom] == nil {
			pi.Edges[vfrom] = []int{}
		}
		pi.Edges[vfrom] = append(pi.Edges[vfrom], vto)
	}
}

func appendUnique(slice []int, item int) []int {
	for _, existing := range slice {
		if existing == item {
			return slice
		}
	}
	return append(slice, item)
}

func updateSummary(md *MasterMetaData, partitionIndex int, v *common.Vertex) {
	if md.PartitionInfoMap[partitionIndex] == nil {
		md.PartitionInfoMap[partitionIndex] = &common.PartitionInfo{
			Vertices:      []int{},
			Edges:         make(map[int][]int),
			Ghosts:        make(map[int]struct{}),
			GhostVertices: []int{},
		}
	}

	found := false
	for _, existingVID := range md.PartitionInfoMap[partitionIndex].Vertices {
		if existingVID == v.ID {
			found = true
			break
		}
	}
	if !found {
		md.PartitionInfoMap[partitionIndex].Vertices = append(md.PartitionInfoMap[partitionIndex].Vertices, v.ID)
		md.NumVertices++
	}

	if md.PartitionInfoMap[partitionIndex].Edges[v.ID] == nil {
		md.PartitionInfoMap[partitionIndex].Edges[v.ID] = []int{}
	}

	if md.EdgeInfoMap[partitionIndex] == nil {
		md.EdgeInfoMap[partitionIndex] = make(map[int][]int)
	}
	if md.EdgeInfoMap[partitionIndex][v.ID] == nil {
		md.EdgeInfoMap[partitionIndex][v.ID] = []int{}
	}
}

func deleteVertex(md *MasterMetaData, v *common.Vertex, partitionID int) {
	if partition, exists := md.PartitionInfoMap[partitionID]; exists {

		for i, vid := range partition.Vertices {
			if vid == v.ID {
				partition.Vertices = append(partition.Vertices[:i], partition.Vertices[i+1:]...)
				md.NumVertices--
				break
			}
		}

		if edges, exists := partition.Edges[v.ID]; exists {
			md.NumEdges -= len(edges)
			delete(partition.Edges, v.ID)
		}

		if md.EdgeInfoMap[partitionID] != nil {
			delete(md.EdgeInfoMap[partitionID], v.ID)
		}

		for vid, edges := range partition.Edges {
			newEdges := []int{}
			for _, targetVID := range edges {
				if targetVID != v.ID {
					newEdges = append(newEdges, targetVID)
				} else {
					md.NumEdges--
				}
			}
			partition.Edges[vid] = newEdges
		}

		if md.EdgeInfoMap[partitionID] != nil {
			for vid, edges := range md.EdgeInfoMap[partitionID] {
				newEdges := []int{}
				for _, targetVID := range edges {
					if targetVID != v.ID {
						newEdges = append(newEdges, targetVID)
					}
				}
				md.EdgeInfoMap[partitionID][vid] = newEdges
			}
		}
	}
}

func deleteEdge(md *MasterMetaData, partitionID int, v1, v2 int) {
	if partition, exists := md.PartitionInfoMap[partitionID]; exists {
		if edges, exists := partition.Edges[v1]; exists {
			newEdges := []int{}
			edgeFound := false
			for _, targetVID := range edges {
				if targetVID != v2 {
					newEdges = append(newEdges, targetVID)
				} else {
					edgeFound = true
				}
			}
			if edgeFound {
				partition.Edges[v1] = newEdges
				md.NumEdges--
			}
		}

		if md.EdgeInfoMap[partitionID] != nil {
			if edges, exists := md.EdgeInfoMap[partitionID][v1]; exists {
				newEdges := []int{}
				for _, targetVID := range edges {
					if targetVID != v2 {
						newEdges = append(newEdges, targetVID)
					}
				}
				md.EdgeInfoMap[partitionID][v1] = newEdges
			}
		}
	}
}

func findVertexPartition(md *MasterMetaData, vID int) int {
	for pi, part := range md.PartitionInfoMap {
		for _, vid := range part.Vertices {
			if vid == vID {
				return pi
			}
		}
	}
	return -1
}

func assignNewPartition(md *MasterMetaData) (int, string, bool) {
	if len(md.FreeWorkers) == 0 {
		return -1, "", false
	}

	idx := len(md.WorkerAddrs)
	addr := md.FreeWorkers[0]
	md.FreeWorkers = md.FreeWorkers[1:]
	md.WorkerAddrs[idx] = addr
	md.PartitionInfoMap[idx] = &common.PartitionInfo{
		Vertices:      []int{},
		Edges:         make(map[int][]int),
		Ghosts:        make(map[int]struct{}),
		GhostVertices: []int{},
	}
	md.EdgeInfoMap[idx] = make(map[int][]int)
	md.NumPartitions++
	return idx, addr, true
}

func removeGhostVertex(md *MasterMetaData, partitionIdx int, ghostID int) {
	pi := md.PartitionInfoMap[partitionIdx]
	newGV := make([]int, 0, len(pi.GhostVertices))
	for _, v := range pi.GhostVertices {
		if v != ghostID {
			newGV = append(newGV, v)
		}
	}
	pi.GhostVertices = newGV
	delete(pi.Ghosts, ghostID)
}

func addGhostVertex(md *MasterMetaData, partition int, vid int) {
	md.mu.Lock()
	defer md.mu.Unlock()
	pi, ok := md.PartitionInfoMap[partition]
	if !ok {
		pi = &common.PartitionInfo{
			Vertices:      []int{},
			Edges:         make(map[int][]int),
			GhostVertices: []int{},
			Ghosts:        make(map[int]struct{}),
		}
		md.PartitionInfoMap[partition] = pi
	} else {
		if pi.Ghosts == nil {
			pi.Ghosts = make(map[int]struct{})
		}
		if pi.Edges == nil {
			pi.Edges = make(map[int][]int)
		}
	}

	if _, exists := pi.Ghosts[vid]; !exists {
		pi.Ghosts[vid] = struct{}{}
		pi.GhostVertices = append(pi.GhostVertices, vid)

		if _, ok := pi.Edges[vid]; !ok {
			pi.Edges[vid] = []int{}
		}
	}
}

func addGhostToWorker(coord *Coordinator, md *MasterMetaData, partition int, vid int, labels map[string]interface{}, nodeType string) error {
	addr := md.WorkerAddrs[partition]

	labelStr := ""
	if nodeType != "" {
		labelStr = ":" + nodeType
	}
	labelStr += ":Ghost"

	props := make([]string, 0, len(labels)+1)
	props = append(props, fmt.Sprintf("vertex_id: %d", vid))
	for k, v := range labels {
		if k != "vertex_id" {
			props = append(props, fmt.Sprintf("%s: \"%v\"", k, v))
		}
	}
	fmt.Println()
	fmt.Println(props)
	createQuery := fmt.Sprintf("CREATE (n:Ghost {%s})", strings.Join(props, ", "))
	fmt.Println(createQuery)
	fmt.Println()

	msg := common.Message{
		Type:      common.AddGhostVertex,
		Vertex:    &common.Vertex{ID: vid, Labels: labels, Type: nodeType},
		Partition: partition,
		Query:     createQuery,
	}
	return coord.Run2PC(context.Background(), msg, []string{addr})
}
