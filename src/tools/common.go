package tools

import "time"

func MakeTimestamp() float64 {
	return float64(time.Now().UnixNano()) / float64(time.Millisecond)
}

func Max(x, y float64) float64 {
	if x < y {
		return y
	} else {
		return x
	}
}
