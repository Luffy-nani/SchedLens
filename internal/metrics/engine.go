package metrics

import (
	"SchedLens/internal/proc"
	"time"
)

type MetricResult struct {
	PID           int
	Name          string
	FairnessScore float64
	IsStarved     bool
	WaitTimeDelta uint64
	CPUTimeDelta  uint64
	SwitchRate    float64
}

// here we used proc.ProccessStatus instead of just ProcessessStatus because proc is the package name(see the imports) thats why to use that struct we've to do like this
func Calculate(current, previous []proc.ProcessStatus, timeDelta time.Duration) []MetricResult {
	prevMap := make(map[int]proc.ProcessStatus)
	for _, p := range previous {
		prevMap[p.PID] = p
	}

	var totalCPU uint64 //calculating the total CPU delta across all the processes
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
		cpuDelta := curr.CPUTime - prev.CPUTime //This is for one single process
		waitDelta := curr.WaitTime - prev.WaitTime
		switchDelta := curr.Switches - prev.Switches

		//Fairness score calculation for that process
		var fairnessScore float64
		if fairShare > 0 {
			fairnessScore = float64(cpuDelta) / fairShare
			if fairnessScore > 1.0 {
				fairnessScore = 1.0
			}
		}

		//Starvation detection for that processes
		// Note: We took threshold value as 500ms(converted to nano)--> can be changed later
		isStarved := waitDelta > 10_000 && cpuDelta < 1000

		//Switche rate per sec = switches/time_delta in seconds(input)
		var switchRate float64
		seconds := timeDelta.Seconds()
		if seconds > 0 {
			switchRate = float64(switchDelta) / seconds
		}

		results = append(results, MetricResult{
			PID:           curr.PID,
			Name:          curr.Name,
			FairnessScore: fairnessScore,
			IsStarved:     isStarved,
			WaitTimeDelta: waitDelta,
			CPUTimeDelta:  cpuDelta,
			SwitchRate:    switchRate,
		})
	}
	return results
}
