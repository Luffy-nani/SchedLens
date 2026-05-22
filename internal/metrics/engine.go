package metrics

import (
	"SchedLens/internal/proc"
	"time"
)

type MetricResult struct {
	PID             int
	Name            string
	FairnessScore   float64
	IsStarved       bool
	StarvationTicks int
	WaitTimeDelta   uint64
	CPUTimeDelta    uint64
	SwitchRate      float64
}

type Engine struct {
	starvationTicks map[int]int // PID → consecutive starved checks
} //We use a struct here because we want to maintain state across calls to Calculate (specifically for starvation detection)--> to remember the number of consecutive ticks a process has been starved, we need to store that in the Engine struct so that it persists across calls to Calculate

func NewEngine() *Engine {
	return &Engine{
		starvationTicks: make(map[int]int),
	}
} //constructor for Engine that initializes the starvationTicks map

// here we used proc.ProccessStatus instead of just ProcessessStatus because proc is the package name(see the imports) thats why to use that struct we've to do like this
func (e *Engine) Calculate(current, previous []proc.ProcessStatus, timeDelta time.Duration) []MetricResult {
	prevMap := make(map[int]proc.ProcessStatus)
	for _, p := range previous {
		prevMap[p.PID] = p
	}

	var totalCPU uint64
	for _, curr := range current {
		if prev, ok := prevMap[curr.PID]; ok {
			totalCPU += curr.CPUTime - prev.CPUTime
		}
	}

	fairShare := float64(totalCPU) / float64(len(current))

	var results []MetricResult
	for _, curr := range current {
		prev, ok := prevMap[curr.PID]
		if !ok {
			continue
		}

		cpuDelta := curr.CPUTime - prev.CPUTime
		waitDelta := curr.WaitTime - prev.WaitTime
		switchDelta := curr.Switches - prev.Switches

		// Fairness score
		var fairnessScore float64
		if fairShare > 0 {
			fairnessScore = float64(cpuDelta) / fairShare
			if fairnessScore > 1.0 {
				fairnessScore = 1.0
			}
		}

		// Starvation detection with consecutive ticks
		var isStarved bool
		if curr.State == "R" && waitDelta > 500*1e6 && cpuDelta < 1000 {
			e.starvationTicks[curr.PID]++
			// Only flag after 3 consecutive checks — avoids false positives
			if e.starvationTicks[curr.PID] >= 3 {
				isStarved = true
			}
		} else {
			// Process recovered — reset ticks
			e.starvationTicks[curr.PID] = 0
			isStarved = false
		}

		// Switch rate
		var switchRate float64
		seconds := timeDelta.Seconds()
		if seconds > 0 {
			switchRate = float64(switchDelta) / seconds
		}

		results = append(results, MetricResult{
			PID:             curr.PID,
			Name:            curr.Name,
			FairnessScore:   fairnessScore,
			IsStarved:       isStarved,
			StarvationTicks: e.starvationTicks[curr.PID],
			WaitTimeDelta:   waitDelta,
			CPUTimeDelta:    cpuDelta,
			SwitchRate:      switchRate,
		})
	}
	return results
}
