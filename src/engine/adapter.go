package engine

import (
	"time"
)

type Adapter struct {
	ends  []*End
	edge  *EdgeServer
	store map[string][]float64
}

func NewAdapter(ends []*End, edge *EdgeServer) *Adapter {
	return &Adapter{ends: ends, edge: edge, store: make(map[string][]float64)}
}

func (adapter *Adapter) adaptLoop() {
	adaptTick := time.Tick(50 * time.Millisecond)

	for {
		select {
		case <-adaptTick:
			if len(adapter.edge.InQueue) > int(0.9*float64(MAX_EDGE_QUEUE)) {
				adapter.edge.model.downshift()

			} else if len(adapter.edge.InQueue) < int(0.1*float64(MAX_EDGE_QUEUE)) {
				adapter.edge.model.upshift()
			}

			for _, end := range adapter.ends {
				if len(end.InQueue) > int(0.9*float64(MAX_END_QUEUE)) {
					end.model.downshift()
				} else if len(end.InQueue) < int(0.1*float64(MAX_END_QUEUE)) {
					end.model.upshift()
				}
			}
		}
	}
}

func (adapter *Adapter) RunInBackground() {
	go adapter.adaptLoop()
}
