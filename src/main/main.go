package main

import (
	"Distream/src/engine"
	"encoding/gob"
	"time"
)

func main() {
	// new Type needs to be registered
	gob.Register(engine.Query{})

	numEnds := 6
	numGPUS := 1

	addr := "localhost:3212"
	edge := engine.NewEdge(addr, numGPUS)

	// start server
	edge.RunInBackground()

	// create ends

	ends := make([]*engine.End, numEnds)
	for i := 0; i < numEnds; i++ {
		ends[i] = engine.NewEnd(i, "campus", addr)
	}

	//
	for i := 0; i < numEnds; i++ {
		parPoint := float64(len(ends)) / float64(engine.ENDEDGE_COMPUTE_RATIO*numGPUS+len(ends)) //Static
		//parPoint := 0	//Server-Only
		//parPoint := 1.0 //camera-only
		//parPoint := 0.1 //baseline
		ends[i].BaseParPoint = float64(parPoint)
	}

	// start ends
	for i := 0; i < numEnds; i++ {
		ends[i].RunInBackground()
	}

	tracker := engine.NewTracker(ends, edge)
	tracker.RunInBackground()
	engine.NewLoadBalancer(ends, 0.2).RunInBackground() //TODO: TWO-SIDE LOAD BALANCE
	engine.NewPartitioner(ends, edge).RunInBackground()

	//engine.NewAdapter(ends, edge).RunInBackground()

	<-time.After(5 * 60 * time.Second)

	tracker.Summary()
	//tracker.OutputWorkloads("./exp/capmus_baseline_workload.json")

}
