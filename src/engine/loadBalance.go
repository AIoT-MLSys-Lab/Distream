package engine

import (
	"time"
)

type LoadBalancer struct {
	ends           []*End
	ImbalanceIndex float64
}

func NewLoadBalancer(ends []*End, V float64) *LoadBalancer {
	return &LoadBalancer{ends, V}
}

func ComputeImbalanceIndex(workloads []float64, capacities []float64) float64 {
	max := 0.0
	sum := 0.0
	N := len(workloads)
	for i := 0; i < N; i++ {
		tmp := workloads[i] / capacities[i]
		if tmp > max {
			max = tmp
		}
		sum += tmp
	}
	if sum > 0 {
		a := max / (sum / float64(N))
		return a - 1
	} else {
		return 0.0
	}
}

func ComputeImbalanceIndexByEnds(ends []*End) float64 {
	max := 0.0
	sum := 0.0
	N := len(ends)
	for _, end := range ends {
		tmp := float64(len(end.InQueue)) / end.Capacity
		if tmp > max {
			max = tmp
		}
		sum += tmp
	}
	if sum > 0 {
		a := max / (sum / float64(N))
		return a - 1
	} else {
		return 0.0
	}
}

func ComputeAveragedNormalizedWorkload(ends []*End) float64 {

	sum := 0.0
	N := len(ends)
	for _, end := range ends {
		tmp := float64(len(end.InQueue)) / end.Capacity
		sum += tmp
	}
	return sum / float64(N)
}

func (lb *LoadBalancer) RunInBackground() {
	go lb.loadBalanceLoop()
}

func (lb *LoadBalancer) loadBalanceLoop() {
	for {
		time.Sleep(time.Millisecond * 500)
		migrations := lb.schedule()
		for from_i := range migrations {
			for to_j, num := range migrations[from_i] {
				if num != 0 {
					if num>15 {
						num=15
					}
					go lb.migrate(from_i, to_j, num)
				}
			}
		}
	}
}

func (lb *LoadBalancer) migrate(sourceEndIndex int, targetEndIndex int, num int) {
	forceComplete := time.After(100 * time.Millisecond)
	for i := 0; i < num; i++ {
		select {
		case q := <-lb.ends[sourceEndIndex].InQueue:
			lb.ends[targetEndIndex].InQueue <- q
		case <-forceComplete:
			return
		}
	}
}

func (lb *LoadBalancer) schedule() [][]int {

	workloads := make([]float64, len(lb.ends))
	capacities := make([]float64, len(lb.ends))
	migrations := make([][]int, len(lb.ends))
	for i := range migrations {
		migrations[i] = make([]int, len(lb.ends))
	}

	for i, end := range lb.ends {

		workloads[i] = float64(len(end.InQueue))
		capacities[i] = end.Capacity
	}

	v := ComputeImbalanceIndex(workloads, capacities)
	for v > lb.ImbalanceIndex {

		//find the largest and smallest
		max := -1.0
		max_idx := -1
		min := 10000.0
		min_idx := -1
		N := len(workloads)
		for i := 0; i < N; i++ {
			tmp := workloads[i] / capacities[i]
			if tmp > max {
				max = workloads[i] / capacities[i]
				max_idx = i
			}
			if tmp < min {
				min = workloads[i] / capacities[i]
				min_idx = i
			}
		}
		if (workloads[max_idx] - workloads[min_idx]) < 4 {
			break
		}
		//accumulate the migration
		num_to_migrate := 2
		migrations[max_idx][min_idx] += int(num_to_migrate)

		//new workload distribution
		workloads[max_idx] -= float64(num_to_migrate)
		workloads[min_idx] += float64(num_to_migrate)

		v = ComputeImbalanceIndex(workloads, capacities)
	}

	return migrations

}
