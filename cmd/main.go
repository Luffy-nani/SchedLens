package main

import (
	"SchedLens/internal/proc"
	"fmt"
)

func main() {
	stats, err := proc.ReadAllProcesses()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	for _, s := range stats {
		fmt.Printf("PID: %d Name: %s WaitTime: %d\n", s.PID, s.Name, s.WaitTime)
	}
}
