package engine

import (
	"Distream/src/tools"
	"log"
	"time"
)

type Incident struct {
	timeStamp int64
	value     interface{}
}

type Partitioner struct {
	ends         []*End
	edge         *EdgeServer
	cache        map[string]Incident
	BaseParPoint float64
}

func NewPartitioner(ends []*End, edge *EdgeServer) *Partitioner {
	base := float64(len(ends)) / float64(ENDEDGE_COMPUTE_RATIO*edge.numWorkers+len(ends))
	return &Partitioner{ends: ends, edge: edge, cache: make(map[string]Incident),
		BaseParPoint: base}
}

func (partitioner *Partitioner) printLoop() {
	printTick := time.Tick(2000 * time.Millisecond)
	for {
		select {
		case <-printTick:
			for i, end := range partitioner.ends {
				log.Printf("end_%d ParPoint: %.2f = %.2f + %.2f workload %d", i,
					end.getParPoint(), end.BaseParPoint, end.FineGrainedOffset, len(end.InQueue))
			}
		}
	}
}

func (partitioner *Partitioner) scheduleBaseParPointLoop() {
	scheduleTick := time.Tick(250 * time.Millisecond)
	TPendsAvg := 0.0
	TPedgeAvg := 0.0
	smooth := 0.4
	for {
		select {
		case <-scheduleTick:
			TPEnds := 0.0
			for _, end := range partitioner.ends {
				//log.Printf("end%d's tp : %.2f", end.id, end.getMaxTP())
				if end.numProcessed != 0 {
					TPEnds += end.getMaxTP()
				}
			}
			TPendsAvg = smooth*TPendsAvg + (1-smooth)*TPEnds
			TPedgeAvg = smooth*TPedgeAvg + (1-smooth)*partitioner.edge.getMaxTP()

			if TPendsAvg > TPedgeAvg+float64(10*partitioner.edge.numWorkers) {
				if TPendsAvg > 1.5*TPedgeAvg {
					partitioner.BaseParPoint += 0.006
				} else if TPendsAvg > 1.3*TPedgeAvg {
					partitioner.BaseParPoint += 0.003
				} else {
					partitioner.BaseParPoint += 0.001
				}
			} else if TPendsAvg < TPedgeAvg-float64(10*partitioner.edge.numWorkers) {
				if 1.5*TPendsAvg < TPedgeAvg {
					partitioner.BaseParPoint -= 0.006
				} else if 1.3*TPendsAvg < TPedgeAvg {
					partitioner.BaseParPoint -= 0.003
				} else {
					partitioner.BaseParPoint -= 0.001
				}
			}
			if partitioner.BaseParPoint > 1 {
				partitioner.BaseParPoint = 1
			} else if partitioner.BaseParPoint < 0 {
				partitioner.BaseParPoint = 0
			}
			for _, end := range partitioner.ends {
				end.BaseParPoint = partitioner.BaseParPoint
			}
		}
	}
}

func (partitioner *Partitioner) scheduleFineGrainedParPointLoop() {
	scheduleTick := time.Tick(250 * time.Millisecond)

	for {
		select {
		case <-scheduleTick:
			wbar := ComputeAveragedNormalizedWorkload(partitioner.ends)
			for _, end := range partitioner.ends {
				w := float64(len(end.InQueue)) / end.Capacity
				tmp := 0.0
				if w == 0 {
					tmp = 1.0
				} else {
					tmp = (wbar - w) / tools.Max(wbar, w)
				}
				end.FineGrainedOffset = tmp * end.BaseParPoint
				//if tmp := (w - wbar) /  w ; tmp > 0.02 {
				//	end.FineGrainedOffset = - tmp * end.BaseParPoint
				//}
			}
		}
	}
}

func (partitioner *Partitioner) RunInBackground() {
	go partitioner.scheduleBaseParPointLoop()
	go partitioner.scheduleFineGrainedParPointLoop()
	//go partitioner.printLoop()
}
