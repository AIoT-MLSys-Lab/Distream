package engine

import (
	"fmt"
	"math/rand"
	"time"
)


var STAGE_DURATIONS = map[string][]int{
	"car":    []int{3, 8, 8, 8},
	"person": []int{3, 8, 8},
	"other":  []int{3, 8},
}


var ENDEDGE_COMPUTE_RATIO = 7

// Query struct
type Query struct {
	BeginTime    time.Time
	EndTime      time.Time
	ObjectClass  string
	CurrentPoint int
	SrcEnd       int
	InferModels  []int
}

func NewQuery(objectClass string, end int) *Query {
	tmp := make([]int, len(STAGE_DURATIONS[objectClass]))
	for i := range tmp {
		tmp[i] = -1
	}
	return &Query{BeginTime: time.Now(), ObjectClass: objectClass, CurrentPoint: 0, SrcEnd: end, InferModels: tmp}
}

func (q Query) String() string {
	return fmt.Sprintf("%v in CurrentPoint %v", q.ObjectClass, q.CurrentPoint)
}

func (q *Query) getExecuteStageFullStochastic(par float64) (toParPoint int) {
	if rand.Float64() > par {
		toParPoint = 0
	} else {
		toParPoint = q.getFinalPar()
	}
	return toParPoint
}

func (q *Query) getExecuteStageSemiStochastic(par float64) (toParPoint int) {

	// par0 --- <stage0> ---- par1 --- <stage1> --- par2 --- <stage2> --- par3
	if par == 0 {
		toParPoint = 0
		return toParPoint
	}
	accuTime := make([]float64, len(STAGE_DURATIONS[q.ObjectClass])+1)
	for i := 0; i < q.getFinalPar()+1; i++ {
		if i == 0 {
			accuTime[i] = 0.0
		} else {
			accuTime[i] = accuTime[i-1] + float64(STAGE_DURATIONS[q.ObjectClass][i-1])
		}
	}
	leftParPoint := 0
	rightParPoint := 0
	for i := 0; i < q.getFinalPar()+1; i++ {
		stagePar := accuTime[i] / accuTime[len(accuTime)-1]
		if stagePar >= par {
			leftParPoint = i - 1
			rightParPoint = i
			break
		}
	}
	//leftParPoint* x + rightParPoint*(1-x) = (total*Par - rightParPoint) /leftParPoint - rightParPoint
	leftProb := (accuTime[len(accuTime)-1]*par - accuTime[rightParPoint]) / (accuTime[leftParPoint] - accuTime[rightParPoint])

	if rand.Float64() <= leftProb {
		toParPoint = leftParPoint
	} else {
		toParPoint = rightParPoint
	}
	return toParPoint
}

func (q *Query) getFinalPar() (finalPar int) {
	switch q.ObjectClass {
	case "car":
		finalPar = len(STAGE_DURATIONS["car"])
	case "person":
		finalPar = len(STAGE_DURATIONS["person"])
	case "other":
		finalPar = len(STAGE_DURATIONS["other"])
	default:
		panic(fmt.Errorf("Error Type"))
	}
	return finalPar
}
