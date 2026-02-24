package main

import (
	"container/internal/common"
	"container/internal/master"
	"container/internal/worker"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

var neo4jDriver neo4j.DriverWithContext

func main() {
	cmd := flag.String("cmd", "master", "start as 'master' or 'worker'")
	addr := flag.String("addr", ":9000", "listen address")
	cohorts := flag.String("cohorts", "localhost:9001,localhost:9002,localhost:9003", "comma-separated cohort addresses (master only)")
	neo4jPort := flag.Int("neo4j-port", 7687, "neo4j bolt port for this worker")
	neo4jHTTPPort := flag.Int("neo4j-http-port", 7474, "neo4j http port for this worker")
	workerID := flag.Int("id", 1, "worker instance id")
	webAddr := flag.String("web-addr", ":8080", "web interface address")

	// go run main.go -cmd worker -addr :9001 -id 1 -neo4j-port 7688 -neo4j-http-port 7475
	// go run main.go -cmd worker -addr :9002 -id 2 -neo4j-port 7689 -neo4j-http-port 7476
	// go run main.go -cmd worker -addr :9003 -id 3 -neo4j-port 7690 -neo4j-http-port 7477
	// go run main.go -cmd=master -cohorts=localhost:9001,localhost:9002,localhost:9003
	flag.Parse()

	switch *cmd {
	case "master":
		RunAsMaster(*addr, *cohorts, *webAddr)
	case "worker":
		RunAsWorker(*addr, *neo4jPort, *neo4jHTTPPort, *workerID)

	default:
		fmt.Fprintln(os.Stderr, "unknown cmd")
		os.Exit(1)
	}
}

func isNeo4jRunning(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func startNeo4jDocker(instanceID, boltPort, httpPort int) error {
	containerName := fmt.Sprintf("neo4j-local%d", instanceID)
	version := "5.17"
	password := "testtest"

	exec.Command("docker", "rm", "-f", containerName).Run()

	cmd := exec.Command("docker", "run", "-d", "--name", containerName,
		"-e", "NEO4J_AUTH=neo4j/"+password,
		"-p", fmt.Sprintf("%d:7474", httpPort),
		"-p", fmt.Sprintf("%d:7687", boltPort),
		"neo4j:"+version,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
func RunAsWorker(addr string, neo4jPort, neo4jHTTPPort, workerID int) {
	if !isNeo4jRunning(neo4jPort) {
		fmt.Printf("Neo4j на порту %d не запущен — запускаю через Docker...\n", neo4jPort)
		if err := startNeo4jDocker(workerID, neo4jPort, neo4jHTTPPort); err != nil {
			log.Fatal("Не удалось запустить neo4j:", err)
		}
		time.Sleep(3 * time.Second)
	} else {
		fmt.Printf("Neo4j на %d уже запущен.\n", neo4jPort)
	}
	uri := fmt.Sprintf("bolt://localhost:%d", neo4jPort)
	user := "neo4j"
	pass := "testtest"
	if _, err := worker.InitNeo4j(uri, user, pass); err != nil {
		log.Fatal("Ошибка инициализации Neo4j:", err)
	}
	log.Printf("Starting worker at %s (neo4j @ %d)", addr, neo4jPort)
	if err := worker.StartCohortServer(addr); err != nil {
		log.Fatal(err)
	}
}

func RunAsMaster(addr, cohorts string, webAddr string) {

	cohortList := master.ParseCohortList(cohorts)
	coordinator := master.NewCoordinator(cohortList)

	metadata := master.NewMasterMetaData()
	metadata.NumPartitions = len(cohortList)
	for i, waddr := range cohortList {
		metadata.WorkerAddrs[i] = waddr
		metadata.PartitionInfoMap[i] = &common.PartitionInfo{Vertices: []int{}, Edges: make(map[int][]int)}
		metadata.EdgeInfoMap[i] = make(map[int][]int)
	}
	http.HandleFunc("/", master.ServeIndex)
	http.HandleFunc("/static/", master.ServeStatic)
	http.HandleFunc("/api/info", master.EnableCORS(master.HandleInfo(metadata)))
	http.HandleFunc("/api/worker-info", master.EnableCORS(master.HandleWorkerInfo(metadata)))
	http.HandleFunc("/api/cypher", master.EnableCORS(master.HandleCypher(coordinator, metadata)))
	http.HandleFunc("/api/upload", master.EnableCORS(master.HandleUpload(coordinator, metadata)))

	go func() {
		log.Printf("Starting web interface at %s", webAddr)
		if err := http.ListenAndServe(webAddr, nil); err != nil {
			log.Fatalf("Failed to start web server: %v", err)
		}
	}()
	select {}

}

func calculateAverageLoad(md *master.MasterMetaData) float64 {
	total := 0
	for _, pi := range md.PartitionInfoMap {
		total += len(pi.Edges)
	}
	if md.NumPartitions == 0 {
		return 0
	}
	return float64(total) / float64(md.NumPartitions)
}
