package engine

import (
	"Distream/src/rpc"
	"Distream/src/tools"
	"fmt"
	"log"
	"sync"
	"time"
)

var MAX_EDGE_QUEUE = 1000

type EdgeServer struct {
	rpc               *rpc.RPCServer
	InQueue           chan Query
	Quit              chan bool
	stat              *tools.Stat
	processTimeMovAvg *tools.MovingAvg
	numWorkers        int
	tokenChan         chan string // for getting access to GPU
	model             *Model
	finishedQueries   []*Query
	latencies         []float64
}

// NewServer creates a new server
func NewEdge(addr string, numWorkers int) *EdgeServer {

	srv := rpc.NewServer(addr)
	edge := EdgeServer{rpc: srv,
		InQueue:           make(chan Query, MAX_EDGE_QUEUE),
		Quit:              make(chan bool),
		processTimeMovAvg: &tools.MovingAvg{WindowLenghth: 100, UpdateChan: make(chan float64)},
		stat:              &tools.Stat{make(map[string]int), &sync.RWMutex{}},
		numWorkers:        numWorkers,
		tokenChan:         make(chan string),
		model:             NewModel()}

	srv.Register("SubmitQuery", edge.SubmitQuery)
	return &edge
}

func (edge *EdgeServer) SubmitQuery(q Query) error {
	edge.InQueue <- q
	edge.stat.Incre("numRev", 1)
	return nil
}

func (edge *EdgeServer) InferLoop() {

	//inject initial tokens
	go func() {
		for i := 0; i < edge.numWorkers; i++ {
			edge.tokenChan <- fmt.Sprintf("%d", i+1)
		}
	}()

	for {
		select {
		//TODO: if batching is needed, then add baching module (using queues with timeout) between edge.InQueue and process.
		case query := <-edge.InQueue:
			token := <-edge.tokenChan
			go edge.processWithToken(&query, token)
		}
	}
}

func (edge *EdgeServer) processWithToken(query *Query, token string) {
	t1 := tools.MakeTimestamp()
	edge.process(query)
	t2 := tools.MakeTimestamp()

	edge.tokenChan <- token //give back token

	query.EndTime = time.Now()

	edge.finishedQueries = append(edge.finishedQueries, query)
	edge.latencies = append(edge.latencies, float64(query.EndTime.Sub(query.BeginTime).Microseconds())/1e6)
	edge.processTimeMovAvg.UpdateChan <- float64(t2 - t1)

	edge.stat.Incre("numProcessed", 1)

}

func (edge *EdgeServer) process(query *Query) {
	
	edge.processQuery(query)

}


func (edge *EdgeServer) getMaxTP() float64 {
	if edge.stat.Get("numProcessed") == 0 {
		return 0
	}
	if edge.processTimeMovAvg.Value() == 0.0 {
		return 50000.0
	} else {
		return 1000.0 / edge.processTimeMovAvg.Value() * float64(edge.numWorkers)
	}
}

func (edge *EdgeServer) printWorkload() {
	tick := time.Tick(time.Second)
	for {
		select {
		case <-tick:
			log.Println(len(edge.InQueue))
		}
	}
}
func (edge *EdgeServer) RunInBackground() {
	// start server to process request
	go edge.rpc.Run()
	time.Sleep(time.Millisecond * 500)
	go edge.InferLoop()
	go edge.processTimeMovAvg.UpdateMovAvgLoop(500)
	//go edge.printWorkloadAndLatency()
}
