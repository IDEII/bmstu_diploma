package master

import (
	"container/internal/common"
	"fmt"
	"regexp"
	"strings"
)

type QueryStructure struct {
	Nodes          []NodePattern
	Relationships  []RelationshipPattern
	ReturnClause   string
	Paths          [][]RelationshipPattern
	Properties     map[string]interface{}
	QueryType      string
	DeleteClause   string
	IsDetachDelete bool
	DeleteTargets  []string
}

type NodePattern struct {
	Variable   string
	Labels     []string
	Properties map[string]interface{}
}

type RelationshipPattern struct {
	FromNode   NodePattern
	ToNode     NodePattern
	RelType    string
	Variable   string
	Properties map[string]interface{}
	Direction  string
}

func parseQuery(query string) (*QueryStructure, error) {
	structure := &QueryStructure{
		Nodes:         []NodePattern{},
		Relationships: []RelationshipPattern{},
		Paths:         [][]RelationshipPattern{},
		Properties:    make(map[string]interface{}),
		DeleteTargets: []string{},
	}

	query = strings.TrimSpace(query)
	upperQuery := strings.ToUpper(query)

	detachDeleteIndex := strings.Index(upperQuery, "DETACH DELETE")
	deleteIndex := strings.Index(upperQuery, "DELETE")

	var deleteStart int = -1
	var hasDelete bool = false

	if detachDeleteIndex != -1 {
		structure.IsDetachDelete = true
		deleteStart = detachDeleteIndex
		hasDelete = true
		structure.DeleteClause = strings.TrimSpace(query[detachDeleteIndex+13:])
	} else if deleteIndex != -1 {
		structure.IsDetachDelete = false
		deleteStart = deleteIndex
		hasDelete = true
		structure.DeleteClause = strings.TrimSpace(query[deleteIndex+6:])
	}

	var returnIndex int = -1
	if !hasDelete {
		returnIndex = strings.Index(upperQuery, "RETURN")
		if returnIndex != -1 {
			structure.ReturnClause = strings.TrimSpace(query[returnIndex+6:])
		}
	}

	if hasDelete {
		err := parseDeleteTargets(structure.DeleteClause, structure)
		if err != nil {
			return nil, err
		}
	}

	matchIndex := strings.Index(upperQuery, "MATCH")
	var matchPart string
	if matchIndex != -1 {
		var endIndex int
		if hasDelete && deleteStart != -1 {
			endIndex = deleteStart
		} else if returnIndex != -1 {
			endIndex = returnIndex
		} else {
			endIndex = len(query)
		}
		matchPart = strings.TrimSpace(query[matchIndex+5 : endIndex])
	}

	if strings.Count(matchPart, "->")+strings.Count(matchPart, "<-") >= 2 {
		structure.QueryType = "multiple_paths"
		err := parseMultiplePathsQuery(matchPart, structure)
		if err != nil {
			return nil, err
		}
	} else if strings.Contains(matchPart, "-[") && strings.Contains(matchPart, "]->") {
		structure.QueryType = "relationship"
		err := parseRelationshipQuery(matchPart, structure)
		if err != nil {
			return nil, err
		}
	} else {
		structure.QueryType = "node_only"
		err := parseNodeOnlyQuery(matchPart, structure)
		if err != nil {
			return nil, err
		}
	}

	if hasDelete {
		if structure.IsDetachDelete {
			structure.QueryType = "detach_delete"
		} else {
			structure.QueryType = "delete"
		}
	}

	return structure, nil
}

func parseDeleteTargets(deleteClause string, structure *QueryStructure) error {
	deleteClause = strings.TrimSpace(deleteClause)

	targets := strings.Split(deleteClause, ",")

	for _, target := range targets {
		target = strings.TrimSpace(target)
		if target != "" {
			structure.DeleteTargets = append(structure.DeleteTargets, target)
		}
	}

	return nil
}
func parseMultiplePathsQuery(matchPart string, structure *QueryStructure) error {

	pathStrings := strings.Split(matchPart, ",")

	allNodes := make(map[string]NodePattern)

	for _, pathStr := range pathStrings {
		pathStr = strings.TrimSpace(pathStr)

		relRe := regexp.MustCompile(`\(([^)]+)\)\s*-\[([^\]]*)\]->\s*\(([^)]+)\)`)
		relMatch := relRe.FindStringSubmatch(pathStr)

		if len(relMatch) > 3 {
			fromNode := parseNodePattern(relMatch[1])
			toNode := parseNodePattern(relMatch[3])

			allNodes[fromNode.Variable] = fromNode
			allNodes[toNode.Variable] = toNode

			rel := RelationshipPattern{
				FromNode:  fromNode,
				ToNode:    toNode,
				Direction: "->",
			}

			if relMatch[2] != "" {
				rel.Variable, rel.RelType, rel.Properties = parseRelationshipPattern(relMatch[2])
			}

			structure.Relationships = append(structure.Relationships, rel)

			path := []RelationshipPattern{rel}
			structure.Paths = append(structure.Paths, path)
		}
	}

	for _, node := range allNodes {
		structure.Nodes = append(structure.Nodes, node)
	}

	return nil
}

func parseNodeOnlyQuery(matchPart string, structure *QueryStructure) error {

	nodeRe := regexp.MustCompile(`\(([^)]+)\)`)
	nodeMatch := nodeRe.FindStringSubmatch(matchPart)

	if len(nodeMatch) > 1 {
		node := parseNodePattern(nodeMatch[1])
		structure.Nodes = append(structure.Nodes, node)
	}

	return nil
}

func parseRelationshipQuery(matchPart string, structure *QueryStructure) error {

	relRe := regexp.MustCompile(`\(([^)]+)\)\s*-\[([^\]]*)\]->\s*\(([^)]+)\)`)
	relMatch := relRe.FindStringSubmatch(matchPart)

	if len(relMatch) > 3 {
		fromNode := parseNodePattern(relMatch[1])
		toNode := parseNodePattern(relMatch[3])

		rel := RelationshipPattern{
			FromNode:  fromNode,
			ToNode:    toNode,
			Direction: "->",
		}

		if relMatch[2] != "" {
			rel.Variable, rel.RelType, rel.Properties = parseRelationshipPattern(relMatch[2])
		}

		structure.Relationships = append(structure.Relationships, rel)
		structure.Nodes = append(structure.Nodes, fromNode, toNode)
	}

	return nil
}

func parseNodePattern(nodeStr string) NodePattern {
	node := NodePattern{
		Properties: make(map[string]interface{}),
	}

	nodeStr = strings.TrimSpace(nodeStr)

	propsRe := regexp.MustCompile(`\{([^}]+)\}`)
	propsMatch := propsRe.FindStringSubmatch(nodeStr)
	if len(propsMatch) > 1 {
		node.Properties = parseProperties(propsMatch[1])

		nodeStr = propsRe.ReplaceAllString(nodeStr, "")
	}

	parts := strings.Split(nodeStr, ":")
	if len(parts) > 0 {
		node.Variable = strings.TrimSpace(parts[0])
	}

	for i := 1; i < len(parts); i++ {
		label := strings.TrimSpace(parts[i])
		if label != "" {
			node.Labels = append(node.Labels, label)
		}
	}

	return node
}

func parseRelationshipPattern(relStr string) (string, string, map[string]interface{}) {
	variable := ""
	relType := ""
	properties := make(map[string]interface{})

	relStr = strings.TrimSpace(relStr)

	propsRe := regexp.MustCompile(`\{([^}]+)\}`)
	propsMatch := propsRe.FindStringSubmatch(relStr)
	if len(propsMatch) > 1 {
		properties = parseProperties(propsMatch[1])

		relStr = propsRe.ReplaceAllString(relStr, "")
	}

	parts := strings.Split(relStr, ":")
	if len(parts) > 0 && strings.TrimSpace(parts[0]) != "" {
		variable = strings.TrimSpace(parts[0])
	}

	if len(parts) > 1 {
		relType = strings.TrimSpace(parts[1])
	}

	return variable, relType, properties
}

func buildNodeQuery(node NodePattern, partition *common.PartitionInfo, returnClause string) string {
	var parts []string

	matchPart := fmt.Sprintf("MATCH (%s)", node.Variable)
	parts = append(parts, matchPart)

	var whereConditions []string

	whereConditions = append(whereConditions, fmt.Sprintf("%s.vertex_id IN %v", node.Variable, formatVertexList(partition.Vertices)))

	for _, label := range node.Labels {
		whereConditions = append(whereConditions, fmt.Sprintf("%s:%s", node.Variable, label))
	}

	for key, value := range node.Properties {
		whereConditions = append(whereConditions, fmt.Sprintf("%s.%s = %s", node.Variable, key, formatPropertyValue(value)))
	}

	if len(whereConditions) > 0 {
		whereClause := "WHERE " + strings.Join(whereConditions, " AND ")
		parts = append(parts, whereClause)
	}

	if returnClause != "" {
		parts = append(parts, "RETURN "+returnClause)
	} else {
		parts = append(parts, fmt.Sprintf("RETURN %s", node.Variable))
	}

	return strings.Join(parts, " ")
}

func buildRelationshipQuery(rel RelationshipPattern, partition *common.PartitionInfo, md *MasterMetaData, fragmentID int, returnClause string) string {
	var parts []string

	relPattern := fmt.Sprintf("[%s]", rel.Variable)
	matchPart := fmt.Sprintf("MATCH (%s)-%s%s(%s)",
		rel.FromNode.Variable, relPattern, rel.Direction, rel.ToNode.Variable)
	parts = append(parts, matchPart)

	var whereConditions []string

	for _, label := range rel.FromNode.Labels {
		whereConditions = append(whereConditions, fmt.Sprintf("%s:%s", rel.FromNode.Variable, label))
	}
	for _, label := range rel.ToNode.Labels {
		whereConditions = append(whereConditions, fmt.Sprintf("%s:%s", rel.ToNode.Variable, label))
	}

	for key, value := range rel.FromNode.Properties {
		whereConditions = append(whereConditions, fmt.Sprintf("%s.%s = %s", rel.FromNode.Variable, key, formatPropertyValue(value)))
	}
	for key, value := range rel.ToNode.Properties {
		whereConditions = append(whereConditions, fmt.Sprintf("%s.%s = %s", rel.ToNode.Variable, key, formatPropertyValue(value)))
	}

	if rel.RelType != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("type(%s) = '%s'", rel.Variable, rel.RelType))
	}

	for key, value := range rel.Properties {
		whereConditions = append(whereConditions, fmt.Sprintf("%s.%s = %s", rel.Variable, key, formatPropertyValue(value)))
	}

	if len(whereConditions) > 0 {
		whereClause := "WHERE " + strings.Join(whereConditions, " AND ")
		parts = append(parts, whereClause)
	}

	if returnClause != "" {
		parts = append(parts, "RETURN "+returnClause)
	} else {
		returnVars := []string{rel.FromNode.Variable, rel.Variable, rel.ToNode.Variable}
		parts = append(parts, "RETURN "+strings.Join(returnVars, ", "))
	}

	return strings.Join(parts, " ")
}

func buildSinglePathQuery(queryStructure *QueryStructure) string {
	if len(queryStructure.Relationships) == 0 {
		return ""
	}

	rel := queryStructure.Relationships[0]

	relPattern := fmt.Sprintf("[%s]", rel.Variable)
	if rel.RelType != "" {
		relPattern = fmt.Sprintf("[%s:%s]", rel.Variable, rel.RelType)
	}

	matchPart := fmt.Sprintf("MATCH (%s)-%s%s(%s)",
		rel.FromNode.Variable, relPattern, rel.Direction, rel.ToNode.Variable)

	var parts []string
	parts = append(parts, matchPart)

	var whereConditions []string

	for _, label := range rel.FromNode.Labels {
		whereConditions = append(whereConditions, fmt.Sprintf("%s:%s", rel.FromNode.Variable, label))
	}
	for _, label := range rel.ToNode.Labels {
		whereConditions = append(whereConditions, fmt.Sprintf("%s:%s", rel.ToNode.Variable, label))
	}

	for key, value := range rel.FromNode.Properties {
		whereConditions = append(whereConditions, fmt.Sprintf("%s.%s = %s", rel.FromNode.Variable, key, formatPropertyValue(value)))
	}
	for key, value := range rel.ToNode.Properties {
		whereConditions = append(whereConditions, fmt.Sprintf("%s.%s = %s", rel.ToNode.Variable, key, formatPropertyValue(value)))
	}

	for key, value := range rel.Properties {
		whereConditions = append(whereConditions, fmt.Sprintf("%s.%s = %s", rel.Variable, key, formatPropertyValue(value)))
	}

	if len(whereConditions) > 0 {
		whereClause := "WHERE " + strings.Join(whereConditions, " AND ")
		parts = append(parts, whereClause)
	}

	if queryStructure.ReturnClause != "" {
		parts = append(parts, "RETURN "+queryStructure.ReturnClause)
	} else {
		var returnVars []string

		if rel.FromNode.Variable != "" {
			returnVars = append(returnVars, rel.FromNode.Variable)
		}

		if rel.Variable != "" {
			returnVars = append(returnVars, rel.Variable)
		}

		if rel.ToNode.Variable != "" {
			returnVars = append(returnVars, rel.ToNode.Variable)
		}

		if len(returnVars) > 0 {
			parts = append(parts, "RETURN "+strings.Join(returnVars, ", "))
		}
	}

	return strings.Join(parts, " ")
}

func formatPropertyValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return fmt.Sprintf("'%s'", v)
	case int, int64:
		return fmt.Sprintf("%d", v)
	case float64:
		return fmt.Sprintf("%.2f", v)
	case bool:
		return fmt.Sprintf("%t", v)
	default:
		return fmt.Sprintf("'%v'", v)
	}
}
