package master

import (
	"container/internal/common"
	"encoding/json"
	"fmt"
	"net"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
)

var (
	vertexIDCounter int
	vertexIDMutex   sync.Mutex
)

func getNextVertexID() int {
	vertexIDMutex.Lock()
	defer vertexIDMutex.Unlock()
	vertexIDCounter++
	return vertexIDCounter
}

func ProcessAndAddCypherQuery(coord *Coordinator, md *MasterMetaData, query string) ([]map[string]interface{}, string, error) {

	trees, subqueries := BuildQueryTrees([]string{query})
	if len(trees) == 0 {
		return nil, "error", fmt.Errorf("failed to parse query")
	}

	if strings.Contains(query, "CREATE") {
		for _, sq := range subqueries {
			generated := GenerateQueriesFromBGPs(sq)
			for _, singleQuery := range generated {

				variables := extractBGPVariables(singleQuery)

				for _, v := range variables {
					nodePat := findNodePattern(singleQuery, v)
					labelsMap, nodeType := extractLabelsAndType(nodePat)

					vertexID := getOrCreateVertexID(labelsMap, nodeType)

					vertex := &common.Vertex{
						ID:     vertexID,
						Labels: labelsMap,
						Type:   nodeType,
					}
					if err := ProcessAddVertex2PC(coord, md, vertex, ""); err != nil {
						continue
					}
				}

				edges := extractBGPEdges(singleQuery)

				if len(edges) > 0 && len(variables) > 1 {
					for _, e := range edges {
						fromVar, toVar := e[0], e[1]

						fromNodePat := findNodePattern(singleQuery, fromVar)
						fromLabelsMap, fromNodeType := extractLabelsAndType(fromNodePat)

						toNodePat := findNodePattern(singleQuery, toVar)
						toLabelsMap, toNodeType := extractLabelsAndType(toNodePat)

						edgeLabels, edgeType := extractEdgeLabelsAndType(singleQuery, fromVar, toVar)

						v1 := &common.Vertex{
							Labels: fromLabelsMap,
							Type:   fromNodeType,
						}
						v2 := &common.Vertex{
							Labels: toLabelsMap,
							Type:   toNodeType,
						}

						matchQuery1 := buildVertexExistenceQuery(v1)
						matchQuery2 := buildVertexExistenceQuery(v2)
						results1, err := ProcessMatchQuery(coord, md, matchQuery1, nil)
						if err != nil {
							fmt.Errorf("Problem with edge vertex from")
							continue
						}
						results2, err := ProcessMatchQuery(coord, md, matchQuery2, nil)
						if err != nil {
							fmt.Errorf("Problem with edge vertex to")
							continue
						}

						v1Id, err := extractVertexIDFromResults(results1, "n")
						if err != nil {

						}
						v2Id, err := extractVertexIDFromResults(results2, "n")

						if err != nil {

						}

						if err := ProcessAddEdge2PC(coord, md, v1Id, v2Id, singleQuery, edgeLabels, edgeType, v1, v2); err != nil {
							return nil, "error", err
						}
					}
				}
			}
		}
	}

	if strings.Contains(strings.ToUpper(query), "DELETE") {
		return processDeleteQueries(coord, md, subqueries, query)
	}
	if strings.Contains(strings.ToUpper(query), "MATCH") && !strings.Contains(strings.ToUpper(query), "CREATE") && !strings.Contains(strings.ToUpper(query), "DELETE") {
		finalReuslts, _ := ProcessMatchQuery(coord, md, query, subqueries)
		return finalReuslts, "match", nil
		// fmt.Println(finalReuslts)
	}

	return nil, "create", nil
}

func getOrCreateVertexID(labelsMap map[string]interface{}, nodeType string) int {
	newID := getNextVertexID()
	return newID
}

func processDeleteQueries(coord *Coordinator, md *MasterMetaData, subqueries []*Subquery, originalQuery string) ([]map[string]interface{}, string, error) {
	for _, sq := range subqueries {
		generated := GenerateQueriesFromBGPs(sq)

		for _, singleQuery := range generated {
			if strings.Contains(strings.ToUpper(singleQuery), "DELETE") {

				if strings.Contains(strings.ToUpper(singleQuery), "DETACH DELETE") {

					singleQuery = strings.TrimSpace(singleQuery)

					nodePat, _ := parseQuery(singleQuery)

					nd := findNodePattern(singleQuery, nodePat.DeleteClause)

					labelsMap, nodeType := extractLabelsAndType(nd)

					v1 := &common.Vertex{
						Labels: labelsMap,
						Type:   nodeType,
					}
					matchQuery1 := buildVertexExistenceQuery(v1)

					results1, err := ProcessMatchQuery(coord, md, matchQuery1, nil)
					if err != nil {
						fmt.Errorf("Problem with edge vertex from")
						continue
					}

					v1Id, err := extractVertexIDFromResults(results1, "n")
					if err != nil {
						fmt.Errorf("Problem with edge vertex from")
						continue
					}
					v1.ID = v1Id
					if err := ProcessDeleteVertex2PCAll(coord, md, v1); err != nil {
						return nil, "error", err
					}

				}
				if strings.Contains(strings.ToUpper(singleQuery), "DELETE") && strings.Contains(singleQuery, "->") {

					singleQuery = strings.TrimSpace(singleQuery)

					routPat, _ := parseQuery(singleQuery)

					vars := strings.Split(routPat.DeleteClause, ", ")

					var NodePat []string
					for _, v := range vars {
						fp := findNodePattern(singleQuery, v)
						if fp != "" {
							NodePat = append(NodePat, fp)
						}
					}
					v1 := &common.Vertex{}
					v2 := &common.Vertex{}

					for i, v := range NodePat {
						LabelsMap, NodeType := extractLabelsAndType(v)
						if i == 0 {
							v1 = &common.Vertex{
								Labels: LabelsMap,
								Type:   NodeType,
							}
						} else {
							v2 = &common.Vertex{
								Labels: LabelsMap,
								Type:   NodeType,
							}
						}

					}

					matchQuery1 := buildVertexExistenceQuery(v1)
					matchQuery2 := buildVertexExistenceQuery(v2)
					results1, err := ProcessMatchQuery(coord, md, matchQuery1, nil)
					if err != nil {
						fmt.Errorf("Problem with edge vertex from")
						continue
					}

					v1Id, err := extractVertexIDFromResults(results1, "n")
					if err != nil {
						fmt.Errorf("Problem with edge vertex from")
						continue
					}
					v1.ID = v1Id

					results2, err := ProcessMatchQuery(coord, md, matchQuery2, nil)
					if err != nil {
						fmt.Errorf("Problem with edge vertex from")
						continue
					}

					v2Id, err := extractVertexIDFromResults(results2, "n")
					if err != nil {
						fmt.Errorf("Problem with edge vertex from")
						continue
					}
					v2.ID = v2Id

					if err := ProcessDeleteEdge2PC(coord, md, v1.ID, v2.ID, "", map[string]interface{}{}); err != nil {
						return nil, "error", err
					}
				}
			}
		}
	}
	return nil, "Delete", nil
}
func extractEdgeLabelsAndType(query, fromVar, toVar string) (map[string]interface{}, string) {
	labels := make(map[string]interface{})
	var edgeType string

	pattern := fmt.Sprintf(`\(%s[^)]*\)\s*-\[\s*([^]]*)\s*\]->\s*\(%s[^)]*\)`, fromVar, toVar)
	re := regexp.MustCompile(pattern)
	match := re.FindStringSubmatch(query)

	if len(match) > 1 {
		edgeInfo := match[1]

		typeRe := regexp.MustCompile(`:([^\s:{]+)`)
		typeMatch := typeRe.FindStringSubmatch(edgeInfo)
		if len(typeMatch) > 1 {
			edgeType = typeMatch[1]
		}

		propsRe := regexp.MustCompile(`\{([^}]+)\}`)
		propsMatch := propsRe.FindStringSubmatch(edgeInfo)
		if len(propsMatch) > 1 {
			parts := strings.Split(propsMatch[1], ",")
			for _, part := range parts {
				kv := strings.SplitN(strings.TrimSpace(part), ":", 2)
				if len(kv) == 2 {
					key := strings.TrimSpace(kv[0])
					val := strings.Trim(strings.TrimSpace(kv[1]), `"`)
					labels[key] = val
				}
			}
		}
	}

	return labels, edgeType
}

func buildCreateEdgeQuery(singleQuery string, from int, to int) (string, error) {

	query := strings.TrimSpace(singleQuery)

	upperQuery := strings.ToUpper(query)
	matchIdx := strings.Index(upperQuery, "MATCH")
	createIdx := strings.Index(upperQuery, "CREATE")

	if matchIdx == 0 && createIdx > matchIdx {

		edgeVars := extractBGPEdges(singleQuery)
		if len(edgeVars) == 0 {
			return "", fmt.Errorf("cannot extract edge variables from pattern: %s", singleQuery)
		}
		fmt.Println(singleQuery)
		fromVar := edgeVars[0][0]
		toVar := edgeVars[0][1]

		fromID := from
		toID := to

		relTypeRe := regexp.MustCompile(`-\[\s*[^:]*:\s*(\w+)\s*[^]]*\]->`)
		relMatch := relTypeRe.FindStringSubmatch(singleQuery)

		edgeType := "CONNECTED_TO"
		if len(relMatch) > 1 {
			edgeType = relMatch[1]
		}
		// fmt.Println("aboba")
		// fmt.Println(fmt.Sprintf(
		// 	"MATCH (%s), (%s) WHERE %s.vertex_id = %d AND %s.vertex_id = %d CREATE (%s)-[:%s]->(%s)",
		// 	fromVar, toVar, fromVar, fromID, toVar, toID, fromVar, edgeType, toVar,
		// ))
		// fmt.Println()
		return fmt.Sprintf(
			"MATCH (%s), (%s) WHERE %s.vertex_id = %d AND %s.vertex_id = %d CREATE (%s)-[:%s]->(%s)",
			fromVar, toVar, fromVar, fromID, toVar, toID, fromVar, edgeType, toVar,
		), nil
	}

	if matchIdx != -1 && matchIdx < createIdx {
		return singleQuery, nil
	}

	nodePattern := regexp.MustCompile(`\(\s*([a-zA-Z]\w*)\s*(\{[^\}]*\})?\s*\)`)
	vars := make(map[string]bool)

	matches := nodePattern.FindAllStringSubmatch(singleQuery, -1)
	for _, match := range matches {
		if len(match) >= 2 {
			varName := match[1]
			vars[varName] = true
		}
	}

	matchParts := make([]string, 0, len(vars))
	whereParts := make([]string, 0, len(vars))

	matchParts = append(matchParts, fmt.Sprintf("(n1)"))

	matchParts = append(matchParts, fmt.Sprintf("(n2)"))

	whereParts = append(whereParts, fmt.Sprintf("n1.vertex_id = %d", from))
	whereParts = append(whereParts, fmt.Sprintf("n2.vertex_id = %d", to))

	matchClause := "MATCH " + strings.Join(matchParts, ", ")
	whereClause := "WHERE " + strings.Join(whereParts, " AND ")

	returnRegex := regexp.MustCompile(`(?i)\bRETURN\b`)
	loc := returnRegex.FindStringIndex(singleQuery)

	var createClause, returnClause string
	if loc != nil {
		createClause = strings.TrimSpace(singleQuery[:loc[0]])
		returnClause = strings.TrimSpace(singleQuery[loc[0]:])
	} else {
		createClause = singleQuery
		returnClause = ""
	}

	var builder strings.Builder
	builder.WriteString(matchClause)
	builder.WriteString(" ")
	builder.WriteString(whereClause)
	builder.WriteString(" ")
	builder.WriteString(createClause)
	if returnClause != "" {
		builder.WriteString(" ")
		builder.WriteString(returnClause)
	}

	return builder.String(), nil
}

func findNodePattern(query, varName string) string {
	re := regexp.MustCompile(`\(` + varName + `[^)]*\)`)
	match := re.FindString(query)
	return match
}

func extractBGPEdges(bgp string) [][2]string {
	re := regexp.MustCompile(`(?i)\(\s*(\w*)\s*\)\s*(<?-?\s*\[\s*(\w+)?\s*(:\s*\w+)?\s*(\{[^}]*\})?\s*\]\s*-?>?)\s*\(\s*(\w*)\s*\)`)
	matches := re.FindStringSubmatch(bgp)

	var out [][2]string

	if len(matches) >= 7 {
		from := matches[1]
		arrow := matches[2]
		to := matches[6]

		if strings.HasPrefix(arrow, "<-") {
			out = append(out, [2]string{to, from})
		} else {
			out = append(out, [2]string{from, to})
		}
	}

	return out
}

func extractBGPVariables(bgp string) []string {
	var variables []string
	varSet := make(map[string]bool)

	nodeRe := regexp.MustCompile(`\(\s*([a-zA-Z]\w*)\s*[^)]*\)`)
	nodeMatches := nodeRe.FindAllStringSubmatch(bgp, -1)

	for _, match := range nodeMatches {
		if len(match) > 1 {
			varName := match[1]
			if !varSet[varName] {
				variables = append(variables, varName)
				varSet[varName] = true
			}
		}
	}

	edgeRe := regexp.MustCompile(`\[\s*([a-zA-Z]\w*)\s*[^]]*\]`)
	edgeMatches := edgeRe.FindAllStringSubmatch(bgp, -1)

	for _, match := range edgeMatches {
		if len(match) > 1 {
			varName := match[1]
			if !varSet[varName] {
				variables = append(variables, varName)
				varSet[varName] = true
			}
		}
	}

	return variables
}

func extractLabelsAndType(nodePattern string) (map[string]interface{}, string) {
	labels := make(map[string]interface{})
	var nodeType string

	typeRe := regexp.MustCompile(`:([a-zA-Z]\w*)`)
	typeMatches := typeRe.FindAllStringSubmatch(nodePattern, -1)
	if len(typeMatches) > 0 {

		nodeType = typeMatches[0][1]
	}

	propsRe := regexp.MustCompile(`\{([^}]+)\}`)
	propsMatch := propsRe.FindStringSubmatch(nodePattern)
	if len(propsMatch) > 1 {
		propsStr := propsMatch[1]
		propRe := regexp.MustCompile(`([a-zA-Z]\w*)\s*:\s*("([^"]*)"|(\d+(?:\.\d+)?)|(\w+))`)
		propMatches := propRe.FindAllStringSubmatch(propsStr, -1)

		for _, match := range propMatches {
			if len(match) >= 2 {
				key := match[1]
				valueStr := match[2]

				if strings.HasPrefix(valueStr, "\"") && strings.HasSuffix(valueStr, "\"") {

					labels[key] = strings.Trim(valueStr, "\"")
				} else if strings.Contains(valueStr, ".") {

					if val, err := strconv.ParseFloat(valueStr, 64); err == nil {
						labels[key] = val
					} else {
						labels[key] = valueStr
					}
				} else if val, err := strconv.Atoi(valueStr); err == nil {

					labels[key] = val
				} else if valueStr == "true" || valueStr == "false" {

					labels[key] = valueStr == "true"
				} else {

					labels[key] = valueStr
				}
			}
		}
	}

	return labels, nodeType
}

func GenerateFragmentedQueries(query string, md *MasterMetaData) (map[int][]string, error) {
	fragmentQueries := make(map[int][]string)

	trees, _ := BuildQueryTrees([]string{query})
	if len(trees) == 0 {
		return nil, fmt.Errorf("failed to parse query")
	}

	queryStructure, err := parseQuery(query)
	if err != nil {
		return nil, fmt.Errorf("failed to parse query structure: %v", err)
	}

	for fragmentID, partitionInfo := range md.PartitionInfoMap {
		queries := buildFragmentQueries(queryStructure, fragmentID, partitionInfo, md)
		if len(queries) > 0 {
			fragmentQueries[fragmentID] = queries
		}
	}

	return fragmentQueries, nil
}

func formatVertexList(vertices []int) string {
	strVertices := make([]string, len(vertices))
	for i, v := range vertices {
		strVertices[i] = fmt.Sprintf("%d", v)
	}
	return "[" + strings.Join(strVertices, ",") + "]"
}

func ProcessMatchQuery(coord *Coordinator, md *MasterMetaData, query string, subqueries []*Subquery) ([]map[string]interface{}, error) {

	queryStructure, _ := parseQuery(query)
	if queryStructure.QueryType == "multiple_paths" {
		return processMultiplePathsQuery(coord, md, query)
	} else {
		fragmentQueries, err := GenerateFragmentedQueries(query, md)
		if err != nil {
			return nil, fmt.Errorf("failed to generate fragmented queries: %v", err)
		}

		allPartialMatches, err := executePartialEvaluation(coord, md, fragmentQueries)
		if err != nil {
			return nil, fmt.Errorf("failed to execute partial evaluation: %v", err)
		}

		finalResults, err := assembleCrossingMatches(allPartialMatches, queryStructure)
		if err != nil {
			return nil, fmt.Errorf("failed to assemble crossing matches: %v", err)
		}

		return finalResults, nil
	}
}

func processMultiplePathsQuery(coord *Coordinator, md *MasterMetaData, query string) ([]map[string]interface{}, error) {
	// Сбор всех локализованных запросов со всех подзапросов
	allFragmentQueries := make(map[int][]string)

	queryStructure, _ := parseQuery(query)
	q := []string{query}
	_, subqueries := BuildQueryTrees(q)

	for _, sq := range subqueries {
		generated := GenerateQueriesFromBGPs(sq)
		for _, singleQuery := range generated {
			sq, _ := parseQuery(singleQuery)
			pathQuery := buildSinglePathQuery(sq)

			fragmentQueriesForPath, err := GenerateFragmentedQueriesForPath(pathQuery, md, sq)
			if err != nil {
				return nil, fmt.Errorf("failed to generate fragmented queries for path: %v", err)
			}

			// Группируем по ID фрагмента
			for fragmentID, queries := range fragmentQueriesForPath {
				allFragmentQueries[fragmentID] = append(allFragmentQueries[fragmentID], queries...)
			}
		}
	}

	allPathMatches, err := executePartialEvaluationBatch(coord, md, allFragmentQueries)
	if err != nil {
		return nil, fmt.Errorf("failed to execute partial evaluation for paths: %v", err)
	}

	// Сборка результатов
	finalResults, err := assembleCrossingMatches(allPathMatches, queryStructure)
	if err != nil {
		return nil, fmt.Errorf("failed to assemble crossing matches: %v", err)
	}

	return finalResults, nil
}

func GenerateFragmentedQueriesForPath(pathQuery string, md *MasterMetaData, queryStructure *QueryStructure) (map[int][]string, error) {
	fragmentQueries := make(map[int][]string)
	queryStructure, _ = parseQuery(pathQuery)
	for fragmentID, partition := range md.PartitionInfoMap {
		queries := buildFragmentQueries(queryStructure, fragmentID, partition, md)
		if len(queries) > 0 {
			fragmentQueries[fragmentID] = queries
		}
	}

	return fragmentQueries, nil
}

func executePartialEvaluation(coord *Coordinator, md *MasterMetaData, fragmentQueries map[int][]string) (map[int][]common.LocalPartialMatch, error) {
	allPartialMatches := make(map[int][]common.LocalPartialMatch)

	for fragmentID, queries := range fragmentQueries {
		workerAddr, exists := md.WorkerAddrs[fragmentID]
		if !exists {

			continue
		}

		var fragmentMatches []common.LocalPartialMatch

		for _, query := range queries {
			partialMatches, err := sendPartialMatchRequest(workerAddr, query, fragmentID)
			if err != nil {

				continue
			}
			fragmentMatches = append(fragmentMatches, partialMatches...)
		}

		allPartialMatches[fragmentID] = fragmentMatches

	}

	return allPartialMatches, nil
}

func executePartialEvaluationBatch(coord *Coordinator, md *MasterMetaData, allFragmentQueries map[int][]string) (map[int][]common.LocalPartialMatch, error) {
	allPartialMatches := make(map[int][]common.LocalPartialMatch)

	for fragmentID, queries := range allFragmentQueries {
		if len(queries) == 0 {
			continue
		}

		workerAddr, exists := md.WorkerAddrs[fragmentID]
		if !exists {
			continue
		}

		partialMatches, err := sendPartialMatchRequestBatch(workerAddr, queries, fragmentID)
		if err != nil {
			fmt.Printf("Error executing batch for fragment %d: %v\n", fragmentID, err)
			continue
		}

		allPartialMatches[fragmentID] = append(allPartialMatches[fragmentID], partialMatches...)

	}

	return allPartialMatches, nil
}

func sendPartialMatchRequestBatch(workerAddr string, queries []string, fragmentID int) ([]common.LocalPartialMatch, error) {
	conn, err := net.Dial("tcp", workerAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to worker %s: %v", workerAddr, err)
	}
	defer conn.Close()

	enc := json.NewEncoder(conn)
	dec := json.NewDecoder(conn)

	msg := common.Message{
		Type:      common.FindLocalPartialMatches,
		Queries:   queries,
		Partition: fragmentID,
	}

	if err := enc.Encode(msg); err != nil {
		return nil, fmt.Errorf("failed to encode message: %v", err)
	}

	var response common.Message
	if err := dec.Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	if response.Type == "Error" {
		return nil, fmt.Errorf("worker error: %s", response.Result)
	}

	return response.PartialMatches, nil
}

func sendPartialMatchRequest(workerAddr, query string, fragmentID int) ([]common.LocalPartialMatch, error) {
	conn, err := net.Dial("tcp", workerAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to worker %s: %v", workerAddr, err)
	}
	defer conn.Close()

	enc := json.NewEncoder(conn)
	dec := json.NewDecoder(conn)

	msg := common.Message{
		Type:      common.FindLocalPartialMatches,
		Query:     query,
		Partition: fragmentID,
	}

	if err := enc.Encode(msg); err != nil {
		return nil, fmt.Errorf("failed to encode message: %v", err)
	}

	var response common.Message
	if err := dec.Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	if response.Type == "Error" {
		return nil, fmt.Errorf("worker error: %s", response.Result)
	}

	return response.PartialMatches, nil
}

type PartialMatchPartition struct {
	QueryVertex string                     `json:"query_vertex"`
	Matches     []common.LocalPartialMatch `json:"matches"`
}

type JoinResult struct {
	MatchFunction map[string]interface{} `json:"match_function"`
	IsComplete    bool                   `json:"is_complete"`
}

func assembleCrossingMatches(allPartialMatches map[int][]common.LocalPartialMatch, queryStructure *QueryStructure) ([]map[string]interface{}, error) {
	var allMatches []common.LocalPartialMatch
	for _, matches := range allPartialMatches {
		allMatches = append(allMatches, matches...)
	}

	deduplicatedMatches := deduplicateMatches(allMatches)

	partitioning := createOptimalPartitioning(deduplicatedMatches)

	results, err := executePartitioningBasedJoin(partitioning)
	if err != nil {
		return nil, err
	}

	specificValues := extractSpecificNodeValues(queryStructure)

	var filteredResults []map[string]interface{}
	for _, result := range results {
		if matchesSpecificValues(result, specificValues) {
			filteredResults = append(filteredResults, result)
		}
	}

	return deduplicateFinalResults(filteredResults), nil

}

func extractSpecificNodeValues(queryStructure *QueryStructure) map[string]interface{} {
	specificValues := make(map[string]interface{})

	for _, node := range queryStructure.Nodes {
		for propName, propValue := range node.Properties {
			if propName == "id" {
				specificValues[node.Variable] = propValue
			}
		}
	}

	for _, rel := range queryStructure.Relationships {

		for propName, propValue := range rel.FromNode.Properties {
			if propName == "id" {
				specificValues[rel.FromNode.Variable] = propValue
			}
		}

		for propName, propValue := range rel.ToNode.Properties {
			if propName == "id" {
				specificValues[rel.ToNode.Variable] = propValue
			}
		}
	}

	return specificValues
}

func matchesSpecificValues(result map[string]interface{}, specificValues map[string]interface{}) bool {
	for variable, expectedValue := range specificValues {
		if actualValue, exists := result[variable]; exists {

			expectedStr := fmt.Sprintf("%v", expectedValue)
			actualStr := fmt.Sprintf("%v", actualValue)

			expectedStr = strings.Trim(expectedStr, "\"")
			actualStr = strings.Trim(actualStr, "\"")

			if expectedStr != actualStr {
				return false
			}
		}
	}
	return true
}

func deduplicateMatches(matches []common.LocalPartialMatch) []common.LocalPartialMatch {
	seen := make(map[string]bool)
	var result []common.LocalPartialMatch

	for _, match := range matches {

		key := createMatchKey(match.MatchFunction)
		if !seen[key] {
			seen[key] = true
			result = append(result, match)
		}
	}

	return result
}
func deduplicateFinalResults(results []map[string]interface{}) []map[string]interface{} {
	seen := make(map[string]bool)
	var deduplicated []map[string]interface{}

	for _, result := range results {
		key := createResultKey(result)
		if !seen[key] {
			seen[key] = true
			deduplicated = append(deduplicated, result)
		}
	}

	return deduplicated
}

func createResultKey(result map[string]interface{}) string {
	var keys []string
	for k := range result {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s:%v", k, result[k]))
	}
	return strings.Join(parts, "|")
}

func createMatchKey(matchFunction map[string]interface{}) string {

	var keys []string
	for k := range matchFunction {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s:%v", k, matchFunction[k]))
	}
	return strings.Join(parts, "|")
}
func createOptimalPartitioning(matches []common.LocalPartialMatch) []PartialMatchPartition {
	// Извлекаем все переменные запроса
	queryVariables := extractAllQueryVariables(matches)

	// Группируем совпадения по переменным, для которых есть внутренние вершины

	variableToMatches := make(map[string][]common.LocalPartialMatch)

	for _, variable := range queryVariables {
		for _, match := range matches {
			// Проверяем, есть ли внутренняя вершина для данной переменной
			// fmt.Println(match, variable)
			if hasInternalVertexForVariable(match, variable) {
				variableToMatches[variable] = append(variableToMatches[variable], match)
			}
		}
	}
	// Создаем частичные совпадения

	partitions := make([]PartialMatchPartition, 0)

	for variable, matchesForVar := range variableToMatches {
		if len(matchesForVar) > 0 {
			// Удаляем дубликаты

			deduplicatedMatches := deduplicateMatches(matchesForVar)
			if len(deduplicatedMatches) > 0 {
				partition := PartialMatchPartition{
					QueryVertex: variable,
					Matches:     deduplicatedMatches,
				}

				partitions = append(partitions, partition)
			}
		}
	}

	sort.Slice(partitions, func(i, j int) bool {
		return len(partitions[i].Matches) > len(partitions[j].Matches)
	})
	return partitions
}
func extractVertexIDFromResults(results []map[string]interface{}, variableName string) (int, error) {
	if len(results) == 0 {
		return -1, fmt.Errorf("no results found")
	}

	firstMap := results[0]
	if value, ok := firstMap[variableName]; ok {
		switch v := value.(type) {
		case int:
			return v, nil
		case string:
			if id, err := strconv.Atoi(v); err == nil {
				return id, nil
			}
			return -1, fmt.Errorf("cannot convert string %s to int", v)
		case float64:
			return int(v), nil
		default:
			return -1, fmt.Errorf("unexpected type %T for vertex ID", v)
		}
	}
	return -1, fmt.Errorf("key %s not found in results", variableName)
}

func extractAllQueryVariables(matches []common.LocalPartialMatch) []string {
	variableSet := make(map[string]bool)

	for _, match := range matches {
		for variable := range match.MatchFunction {
			variableSet[variable] = true
		}
	}

	variables := make([]string, 0, len(variableSet))
	for variable := range variableSet {
		variables = append(variables, variable)
	}

	return variables
}

func hasInternalVertexForVariable(match common.LocalPartialMatch, variable string) bool {

	vertexID, exists := match.MatchFunction[variable]
	if !exists {

		return false
	}

	var id int
	var ok bool

	if id, ok = vertexID.(int); ok {

	} else if idFloat, okFloat := vertexID.(float64); okFloat {

		id = int(idFloat)
		ok = true

	} else if idStr, okStr := vertexID.(string); okStr {

		var err error
		if id, err = strconv.Atoi(idStr); err == nil {
			ok = true

		} else {

		}
	} else {

		return false
	}

	if !ok {

		return false
	}

	for _, internalVertex := range match.InternalVertices {

		if internalVertex == id {

			return true
		}
	}

	return false
}

func executePartitioningBasedJoin(partitions []PartialMatchPartition) ([]map[string]interface{}, error) {
	if len(partitions) == 0 {
		return []map[string]interface{}{}, nil
	}
	// первое совпадение становится промежуточным результатом

	var currentResults []JoinResult
	for _, match := range partitions[0].Matches {
		currentResults = append(currentResults, JoinResult{
			MatchFunction: match.MatchFunction,
			IsComplete:    false,
		})
	}
	var finalResults []map[string]interface{}
	// Итеративное объединение с остальными совпадениям

	for i := 1; i < len(partitions); i++ {
		var newResults []JoinResult
		// Для каждого промежуточного результата

		for _, currentResult := range currentResults {
			hasJoin := false
			// Пытаемся объединить с каждым соответствием из текущего совпадения

			for _, partialMatch := range partitions[i].Matches {
				// Если объединить можно
				if areJoinable(currentResult, partialMatch) {
					joinedResult := joinPartialMatches(currentResult, partialMatch)
					// Проверяем полноту результата
					// То проверяем на полноту ответа, то есть наличия всех переменных запроса в совпадении
					if isCompleteMatch(joinedResult, partitions) {
						// и если ответ полный добавляем его в конечные результаты
						finalResults = append(finalResults, joinedResult.MatchFunction)
					} else {
						// если нет, то добавляем в новые промежуточные результаты
						newResults = append(newResults, joinedResult)
					}
					hasJoin = true

				}
			}
			// Если объединения не произошло, но результат частично полный, сохраняем его также в промежуточные результаты

			if !hasJoin {
				if isPartiallyComplete(currentResult, partitions, i) {
					newResults = append(newResults, currentResult)
				}
			}
		}
		// Обновляем старые промежуточные результаты, новыми для обхода по следующим совпадениям
		currentResults = newResults
	}
	// Добавляем оставшиеся полные совпадения

	for _, result := range currentResults {
		if isCompleteMatch(result, partitions) {
			finalResults = append(finalResults, result.MatchFunction)
		}
	}

	return finalResults, nil
}

func isPartiallyComplete(result JoinResult, partitions []PartialMatchPartition, currentPartitionIndex int) bool {

	processedVariables := make(map[string]bool)
	for i := 0; i <= currentPartitionIndex; i++ {
		processedVariables[partitions[i].QueryVertex] = true
	}

	for variable := range result.MatchFunction {
		if processedVariables[variable] {
			return true
		}
	}

	return false
}

func areJoinable(result JoinResult, match common.LocalPartialMatch) bool {
	// проверяем, что одна переменная не сопоставлена разным вершинам

	for variable, resultValue := range result.MatchFunction {
		if matchValue, exists := match.MatchFunction[variable]; exists {
			if !valuesEqual(resultValue, matchValue) {
				return false
			}
		}
	}
	// Требуем наличие общих переменных для избежания объединения без общих переменных

	if len(result.MatchFunction) > 3 {
		hasCommonVariable := false
		for variable := range result.MatchFunction {
			if _, exists := match.MatchFunction[variable]; exists {
				hasCommonVariable = true
				break
			}
		}

		if !hasCommonVariable {
			return false
		}
	}

	return true
}
func valuesEqual(val1, val2 interface{}) bool {
	str1 := fmt.Sprintf("%v", val1)
	str2 := fmt.Sprintf("%v", val2)
	return str1 == str2
}

func joinPartialMatches(result JoinResult, match common.LocalPartialMatch) JoinResult {
	// Создаем новую функцию сопоставления, объединяя существующую и новую

	newMatchFunction := make(map[string]interface{})
	// Копируем существующие сопоставления переменных

	for variable, value := range result.MatchFunction {
		newMatchFunction[variable] = value
	}
	// Добавляем новые сопоставления
	// если fi(v) != NULL && fj(v) != NULL, то f(v) <- fi(v)
	for variable, value := range match.MatchFunction {
		newMatchFunction[variable] = value
	}
	return JoinResult{
		MatchFunction: newMatchFunction,
		IsComplete:    false,
	}
}
func isCompleteMatch(result JoinResult, partitions []PartialMatchPartition) bool {
	// Собираем все переменные из всех разделов

	allVariables := make(map[string]bool)
	for _, partition := range partitions {
		allVariables[partition.QueryVertex] = true
	}
	// Проверяем, что для каждой переменной есть сопоставление

	for variable := range allVariables {
		if _, exists := result.MatchFunction[variable]; !exists {
			return false
		}
	}

	return true
}

func parseProperties(propsStr string) map[string]interface{} {
	properties := make(map[string]interface{})

	propRe := regexp.MustCompile(`([a-zA-Z]\w*)\s*:\s*("([^"]*)"|(\d+(?:\.\d+)?)|(\w+))`)
	propMatches := propRe.FindAllStringSubmatch(propsStr, -1)

	for _, match := range propMatches {
		if len(match) >= 2 {
			key := match[1]
			valueStr := match[2]

			if strings.HasPrefix(valueStr, "\"") && strings.HasSuffix(valueStr, "\"") {
				properties[key] = strings.Trim(valueStr, "\"")
			} else if strings.Contains(valueStr, ".") {
				if val, err := strconv.ParseFloat(valueStr, 64); err == nil {
					properties[key] = val
				} else {
					properties[key] = valueStr
				}
			} else if val, err := strconv.Atoi(valueStr); err == nil {
				properties[key] = val
			} else if valueStr == "true" || valueStr == "false" {
				properties[key] = valueStr == "true"
			} else {
				properties[key] = valueStr
			}
		}
	}

	return properties
}

func hasRelevantRelationship(rel RelationshipPattern, partition *common.PartitionInfo, md *MasterMetaData, fragmentID int) bool {
	edges, exists := md.EdgeInfoMap[fragmentID]
	return exists && len(edges) > 0
}

func buildFragmentQueries(queryStructure *QueryStructure, fragmentID int, partition *common.PartitionInfo, md *MasterMetaData) []string {
	var queries []string

	if queryStructure.QueryType == "node_only" {

		for _, node := range queryStructure.Nodes {
			if len(partition.Vertices) > 0 {
				query := buildNodeQuery(node, partition, queryStructure.ReturnClause)
				if query != "" {
					queries = append(queries, query)
				}
			}
		}
	} else if queryStructure.QueryType == "relationship" {

		for _, rel := range queryStructure.Relationships {
			if hasRelevantRelationship(rel, partition, md, fragmentID) {
				query := buildRelationshipQuery(rel, partition, md, fragmentID, queryStructure.ReturnClause)
				if query != "" {
					queries = append(queries, query)
				}

			}
		}
	} else if queryStructure.QueryType == "multiple_paths" {

		for _, rel := range queryStructure.Relationships {
			if hasRelevantRelationship(rel, partition, md, fragmentID) {
				query := buildRelationshipQuery(rel, partition, md, fragmentID, queryStructure.ReturnClause)
				if query != "" {
					queries = append(queries, query)
				}

			}
		}
	}

	return queries
}
