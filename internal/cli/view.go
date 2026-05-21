package cli

import (
	"SchedLens/internal/metrics"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func Render(results []metrics.MetricResult) {
	fmt.Print("\033[2J\033[H") //This clears the terminal

	// FOR TITLES(copy pasted)
	fmt.Println("SchedLens v1.0 — Live Scheduler View")
	fmt.Println("Updated:", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Println()
	fmt.Printf("%-10s %-20s %-12s %-10s %-12s %-12s\n",
		"PID", "NAME", "FAIRNESS", "STARVED", "WAIT(ms)", "SWITCHES/s")
	fmt.Println("------------------------------------------------------------------------------------")

	for _, r := range results {
		starved := "NO"
		if r.IsStarved {
			starved = "YES.....BEWARE!!"
		}

		waitMs := float64(r.WaitTimeDelta) / 1_000_000 // Just converting in milli seconds

		fmt.Printf("%-10d %-20s %-12.2f %-10s %-12.2f %-12.2f\n",
			r.PID,
			r.Name,
			r.FairnessScore,
			starved,
			waitMs,
			r.SwitchRate,
		)
	}

	// TO handle CTR+C smoothly
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

}
