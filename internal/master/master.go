package master

import (
	"container/internal/common"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

type Coordinator struct {
	Cohorts []string
}

func AddServer(md *MasterMetaData, addr string) int {
	md.mu.Lock()
	defer md.mu.Unlock()
	idx := len(md.WorkerAddrs) + len(md.FreeWorkers)
	md.FreeWorkers = append(md.FreeWorkers, addr)
	return idx
}

func ParseCohortList(cohorts string) []string {
	fields := strings.Split(cohorts, ",")
	var result []string
	for _, addr := range fields {
		if addr := strings.TrimSpace(addr); addr != "" {
			result = append(result, addr)
		}
	}
	return result
}

func NewCoordinator(cohorts []string) *Coordinator {
	return &Coordinator{Cohorts: cohorts}
}

func sendToWorker(addr string, msg common.Message) (common.Message, error) {
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return common.Message{}, err
	}
	defer conn.Close()

	enc := json.NewEncoder(conn)
	dec := json.NewDecoder(conn)
	if err := enc.Encode(msg); err != nil {
		return common.Message{}, err
	}
	var reply common.Message
	if err := dec.Decode(&reply); err != nil {
		return common.Message{}, err
	}
	return reply, nil
}

func (c *Coordinator) Run2PC(ctx context.Context, prepareMsg common.Message, addresses []string) error {
	results := make([]string, len(addresses))
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i, addr := range addresses {
		wg.Add(1)
		go func(idx int, a string) {
			defer wg.Done()
			ctx2, cancel := context.WithTimeout(ctx, 2*time.Second)
			defer cancel()
			reply, err := sendToWorker(a, prepareMsg)
			mu.Lock()
			if err != nil || reply.Type != common.Ack {
				results[idx] = common.MsgNo
			} else {
				results[idx] = common.MsgAck
			}
			mu.Unlock()
			_ = ctx2
		}(i, addr)
	}
	wg.Wait()

	ok := true
	for _, r := range results {
		if r != common.MsgAck {
			ok = false
			break
		}
	}

	commitType := common.MsgCommit
	if !ok {
		commitType = common.MsgRollback
	}
	commitMsg := common.Message{Type: commitType}

	wg = sync.WaitGroup{}
	for _, addr := range addresses {
		wg.Add(1)
		go func(a string) {
			defer wg.Done()
			_, _ = sendToWorker(a, commitMsg)
		}(addr)
	}
	wg.Wait()
	if !ok {
		return fmt.Errorf("prepare phase failed, rolled back")
	}
	return nil
}
