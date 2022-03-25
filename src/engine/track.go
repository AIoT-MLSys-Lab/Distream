package engine

import (
	"encoding/json"
	"fmt"
	"gonum.org/v1/gonum/stat"
	"log"
	"math"
	"os"
	"sort"
	"time"
)

type Tracker struct {
	ends      []*End
	edge      *EdgeServer
	store     map[string][]float64
	workloads map[string][]int
}

func NewTracker(ends []*End, edge *EdgeServer) *Tracker {
	return &Tracker{ends: ends, edge: edge, store: make(map[string][]float64)}
}

func (tracker *Tracker) printWorkloadAndLatency() {
	for i, end := range tracker.ends {
		log.Printf("End%d's workload is %d", i, len(end.InQueue))
	}
	avgLatency, stdLatency := stat.MeanStdDev(tracker.edge.latencies, nil)

	log.Printf("Edge's workload is %d, QueryLatency %.2f +- %.2f",
		len(tracker.edge.InQueue), avgLatency, stdLatency)
}

func (tracker *Tracker) printImbalanceIndex() {
	log.Printf("Workload Imbalance Index v: %f", ComputeImbalanceIndexByEnds(tracker.ends))
}

func (tracker *Tracker) SummaryLatency(latencySLO float64) {
	sort.Float64s(tracker.edge.latencies)
	slo := 1.0
	for i := range tracker.edge.latencies {
		if tracker.edge.latencies[i] > latencySLO {
			slo = float64(i) / float64(len(tracker.edge.latencies))
			log.Printf("**********************latency SLO rate :%.2f", float64(i)/float64(len(tracker.edge.latencies)))
			break
		}
	}
	per1Idx := int(0.01 * float64(len(tracker.edge.latencies)))
	per25Idx := int(0.25 * float64(len(tracker.edge.latencies)))
	per50Idx := int(0.50 * float64(len(tracker.edge.latencies)))
	per75Idx := int(0.75 * float64(len(tracker.edge.latencies)))
	per99Idx := int(0.99 * float64(len(tracker.edge.latencies)))
	mean, std := stat.MeanStdDev(tracker.edge.latencies, nil)
	log.Printf("**********************Average latency: %.2f +- %.2f, latency SLO rate :%.2f, Percentile:(%.2f, %.2f, %.2f, %.2f, %.2f)",
		mean,
		std,
		slo,
		tracker.edge.latencies[per1Idx],
		tracker.edge.latencies[per25Idx],
		tracker.edge.latencies[per50Idx],
		tracker.edge.latencies[per75Idx],
		tracker.edge.latencies[per99Idx])
}

func (tracker *Tracker) SummaryThroughput() {
	sort.Float64s(tracker.store["monitoredTPs"])
	per1Idx := int(0.01 * float64(len(tracker.store["monitoredTPs"])))
	per25Idx := int(0.25 * float64(len(tracker.store["monitoredTPs"])))
	per50Idx := int(0.50 * float64(len(tracker.store["monitoredTPs"])))
	per75Idx := int(0.75 * float64(len(tracker.store["monitoredTPs"])))
	per99Idx := int(0.99 * float64(len(tracker.store["monitoredTPs"])))
	mean, std := stat.MeanStdDev(tracker.store["monitoredTPs"], nil)
	log.Printf("**********************Monitored TP :%.2f +- %.2f Percentile:(%.2f, %.2f, %.2f, %.2f, %.2f)",
		mean,
		std,
		tracker.store["monitoredTPs"][per1Idx],
		tracker.store["monitoredTPs"][per25Idx],
		tracker.store["monitoredTPs"][per50Idx],
		tracker.store["monitoredTPs"][per75Idx],
		tracker.store["monitoredTPs"][per99Idx])
	mean, std = stat.MeanStdDev(tracker.store["sysTPs"], nil)
	log.Printf("**********************System TP :%.2f +- %.2f",
		mean,
		std)
}

func (tracker *Tracker) PrintLoop() {
	workloadTick := time.Tick(time.Second * 10)
	imbalanceTick := time.Tick(time.Second * 10)
	tpTick := time.Tick(time.Second)
	sysTPTick := time.Tick(time.Second)

	lastNumProcess := 0
	lastNumRev := 0
	for {
		select {
		case <-workloadTick:
			tracker.printWorkloadAndLatency()
		case <-imbalanceTick:
			tracker.printImbalanceIndex()
		case <-tpTick:
			currentProcessedNum := tracker.edge.stat.Get("numProcessed")
			currentRevNum := tracker.edge.stat.Get("numRev")
			monitoredTP := float64(currentProcessedNum-lastNumProcess) / 1.0
			tracker.store["monitoredTPs"] = append(tracker.store["monitoredTPs"], monitoredTP)
			log.Printf("MonitoredReceived TP: %.2f, Monitored Overall TP: %.2f, AverageTP:%.2f +- %.2f",
				float64(currentRevNum-lastNumRev)/1.0,
				monitoredTP, stat.Mean(tracker.store["monitoredTPs"], nil),
				stat.StdDev(tracker.store["monitoredTPs"], nil))

			lastNumProcess = currentProcessedNum
			lastNumRev = currentRevNum

		case <-sysTPTick:
			TPEnds := 0.0
			for _, end := range tracker.ends {
				//log.Printf("end%d's tp : %.2f", end.id, end.getMaxTP())
				TPEnds += end.getMaxTP()
			}
			TPEdge := tracker.edge.getMaxTP()
			TPsys := math.Min(TPEdge, TPEnds)
			tracker.store["sysTPs"] = append(tracker.store["sysTPs"], TPsys)
			log.Printf("ends's Maximum TP:%.2f , edge's Maximum TP: %.2f, System Maximum TP:%.2f, Avg System TP:%.2f +- %.2f",
				TPEnds, TPEdge, TPsys, stat.Mean(tracker.store["sysTPs"], nil),
				stat.StdDev(tracker.store["sysTPs"], nil))
		}
	}
}

func (tracker *Tracker) RunInBackground() {
	go tracker.PrintLoop()
	go tracker.collectUtilization()
}

func (tracker *Tracker) collectUtilization() {
	collectTick := time.Tick(200 * time.Millisecond)
	tracker.workloads = make(map[string][]int)
	tracker.workloads["numGPU"] = []int{tracker.edge.numWorkers}
	tracker.workloads["ratio"] = []int{ENDEDGE_COMPUTE_RATIO}
	for {
		select {
		case <-collectTick:
			accumlatedOverpro := 0
			for _, end := range tracker.ends {
				tracker.workloads[fmt.Sprintf("end%d", end.id)] = append(tracker.workloads[fmt.Sprintf("end%d", end.id)],
					len(end.InQueue))
			}
			if len(tracker.edge.InQueue) < 4 {
				accumlatedOverpro += ENDEDGE_COMPUTE_RATIO * tracker.edge.numWorkers
			}
			tracker.workloads["edge"] = append(tracker.workloads[fmt.Sprintf("edge")], len(tracker.edge.InQueue))
		}
	}
}

func (tracker *Tracker) Summary() {
	tracker.SummaryThroughput()
	tracker.SummaryLatency(3.0)
	log.Printf("**********************Total Queries : %d", len(tracker.edge.finishedQueries))

}

func (tracker *Tracker) OutputWorkloads(filepath string) {
	jsonData, err := json.Marshal(tracker.workloads)
	jsonFile, err := os.Create(filepath)
	if err != nil {
		panic(err)
	}
	defer jsonFile.Close()

	jsonFile.Write(jsonData)
	jsonFile.Close()
	fmt.Println("JSON data written to ", jsonFile.Name())

}
