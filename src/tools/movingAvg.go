package tools

import (
	"sync"
	"time"
)

type MovingAvg struct {
	WindowLenghth int
	Store         []float64
	cache         float64
	mux           sync.Mutex
	UpdateChan    chan float64
}

func (movAvg *MovingAvg) update(x float64) {
	movAvg.mux.Lock()
	if len(movAvg.Store) >= movAvg.WindowLenghth {
		movAvg.Store = movAvg.Store[1:]
	}
	movAvg.Store = append(movAvg.Store, x)
	sum := 0.0
	for _, a := range movAvg.Store {
		sum += a
	}
	movAvg.cache = sum / float64(len(movAvg.Store))
	movAvg.mux.Unlock()
}

func (movAvg *MovingAvg) Value() float64 {
	return movAvg.cache
}

func (movAvg *MovingAvg) reset() {
	movAvg.mux.Lock()
	movAvg.Store = nil
	movAvg.mux.Unlock()
}

func (movAvg *MovingAvg) UpdateMovAvgLoop(timeOutInMS int) {
	for {
		select {
		case value := <-movAvg.UpdateChan:
			movAvg.update(value)
		case <-time.After(time.Duration(timeOutInMS) * time.Millisecond):
			movAvg.reset()
		}
	}
}
