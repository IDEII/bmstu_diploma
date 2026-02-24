package common

const (
	MAXCAP          = 20
	ThresholdFactor = 0.1
	PortMaster      = ":9000"
)

type Vertex struct {
	ID     int
	Edges  []int
	Labels map[string]interface{}
	Type   string
}
type Edge struct {
	V1     int                    `json:"v1"`
	V2     int                    `json:"v2"`
	Labels map[string]interface{} `json:"labels,omitempty"`
	Type   string                 `json:"type,omitempty"`
}

type Message struct {
	Type            string                   `json:"type"`
	Vertex          *Vertex                  `json:"vertex,omitempty"`
	Edge            *Edge                    `json:"edge,omitempty"`
	Query           string                   `json:"query,omitempty"`
	Queries         []string                 `json:"queries,omitempty"`
	Partition       int                      `json:"partition,omitempty"`
	Variables       []string                 `json:"variables,omitempty"`
	Result          string                   `json:"result,omitempty"`
	Results         []map[string]interface{} `json:"results,omitempty"`
	PartialMatches  []LocalPartialMatch      `json:"partial_matches,omitempty"`
	FragmentQueries map[int]string           `json:"fragment_queries,omitempty"`
	EdgeLabels      map[string]interface{}   `json:"edge_labels,omitempty"`
	EdgeType        string                   `json:"edge_type,omitempty"`
	PartitionInfo   *PartitionInfo           `json:"partition_info,omitempty"`
}

type MessageType string

const (
	AddVertex               = "AddVertex"
	AddEdge                 = "AddEdge"
	DeleteVertex            = "DeleteVertex"
	DeleteEdge              = "DeleteEdge"
	Match                   = "Match"
	MatchVertex             = "MatchVertex"
	UpdateGhostStatus       = "UpdateGhostStatus"
	Ack                     = "Ack"
	MsgPrepare              = "PREPARE"
	MsgCommit               = "COMMIT"
	MsgRollback             = "ROLLBACK"
	MsgAck                  = "ACK"
	MsgNo                   = "NO"
	AddGhostVertex          = "AddGhostVertex"
	RemoveGhostVertex       = "RemoveGhostVertex"
	ExecQuery               = "ExecQuery"
	FindLocalPartialMatches = "FindLocalPartialMatches"
	AssembleMatches         = "AssembleMatches"
	PartialMatchResult      = "PartialMatchResult"
	GetWorkerInfo           = "GetWorkerInfo"
	GetNodeDetails          = "GetNodeDetails"
	GetEdgeDetails          = "GetEdgeDetails"
)

type LocalPartialMatch struct {
	FragmentID       int                    `json:"fragment_id"`
	MatchFunction    map[string]interface{} `json:"match_function"`
	CrossingEdges    []Edge                 `json:"crossing_edges"`
	InternalVertices []int                  `json:"internal_vertices"`
	ExtendedVertices []int                  `json:"extended_vertices"`
}

type PartitionInfo struct {
	Vertices      []int            `json:"vertices"`
	Edges         map[int][]int    `json:"edges"`
	GhostVertices []int            `json:"ghost_vertices"`
	Ghosts        map[int]struct{} `json:"ghosts"`
	QueryCache    []string         `json:"query_cache"`
}

type VariableIDMap map[string]int

func (m VariableIDMap) Add(varname string, vid int) {
	m[varname] = vid
}

func (m VariableIDMap) Get(varname string) (int, bool) {
	vid, ok := m[varname]
	return vid, ok
}
func (m VariableIDMap) Pop(vid int) {
	for v, i := range m {
		if i == vid {
			delete(m, v)
		}
	}
}
