package main

import (
	"fmt"
	"time"
)

func main_4() {
	timeout1 := time.After(5 * time.Second)

	for {
		select {
		case <-timeout1:
			fmt.Println("The end!")
			return
		case <-time.After(1 * time.Second):
			fmt.Println("There's no more time to this. Exiting!")
		}
	}

}
