package metrics

import (
	appconfig "SchedLens/internal/config"
	"SchedLens/internal/proc"
	"testing"
	"time"
)

func testConfig() appconfig.MetricsConfig {
	return appconfig.MetricsConfig{
		StarvationThresholdMs: 500,
		CpuDeltaThreshold:     1000,
		StarvationTicks:       3,
	}
}

func TestCalculate_FairnessScore(t *testing.T) {
	engine := NewEngine(testConfig())

	current := []proc.ProcessStatus{
		{PID: 1, Name: "p1", State: "S", CPUTime: 1000, WaitTime: 0, Switches: 0},
		{PID: 2, Name: "p2", State: "S", CPUTime: 0, WaitTime: 0, Switches: 0},
	}
	previous := []proc.ProcessStatus{
		{PID: 1, Name: "p1", State: "S", CPUTime: 0, WaitTime: 0, Switches: 0},
		{PID: 2, Name: "p2", State: "S", CPUTime: 0, WaitTime: 0, Switches: 0},
	}

	results := engine.Calculate(current, previous, 2*time.Second)

	if results[0].FairnessScore != 1.0 {
		t.Errorf("expected fairness 1.0, got %f", results[0].FairnessScore)
	}
	if results[1].FairnessScore != 0.0 {
		t.Errorf("expected fairness 0.0, got %f", results[1].FairnessScore)
	}
}

func TestCalculate_StarvationNotFlaggedForSleepingProcess(t *testing.T) {
	engine := NewEngine(testConfig())

	current := []proc.ProcessStatus{
		{PID: 1, Name: "p1", State: "S", CPUTime: 0, WaitTime: 1000 * 1e6, Switches: 0},
	}
	previous := []proc.ProcessStatus{
		{PID: 1, Name: "p1", State: "S", CPUTime: 0, WaitTime: 0, Switches: 0},
	}

	results := engine.Calculate(current, previous, 2*time.Second)

	if results[0].IsStarved {
		t.Error("sleeping process should not be flagged as starved")
	}
}

func TestCalculate_StarvationRequiresConsecutiveTicks(t *testing.T) {
	engine := NewEngine(testConfig())

	current := []proc.ProcessStatus{
		{PID: 1, Name: "p1", State: "R", CPUTime: 0, WaitTime: 1000 * 1e6, Switches: 0},
	}
	previous := []proc.ProcessStatus{
		{PID: 1, Name: "p1", State: "R", CPUTime: 0, WaitTime: 0, Switches: 0},
	}

	// first tick — should not be starved yet
	results := engine.Calculate(current, previous, 2*time.Second)
	if results[0].IsStarved {
		t.Error("should not be starved after only 1 tick")
	}

	// second tick
	results = engine.Calculate(current, previous, 2*time.Second)
	if results[0].IsStarved {
		t.Error("should not be starved after only 2 ticks")
	}

	// third tick — now it should be starved
	results = engine.Calculate(current, previous, 2*time.Second)
	if !results[0].IsStarved {
		t.Error("should be starved after 3 consecutive ticks")
	}
}

func TestCalculate_StarvationTicksResetOnRecovery(t *testing.T) {
	engine := NewEngine(testConfig())

	starving := []proc.ProcessStatus{
		{PID: 1, Name: "p1", State: "R", CPUTime: 0, WaitTime: 1000 * 1e6, Switches: 0},
	}
	recovered := []proc.ProcessStatus{
		{PID: 1, Name: "p1", State: "R", CPUTime: 5000, WaitTime: 0, Switches: 0},
	}
	base := []proc.ProcessStatus{
		{PID: 1, Name: "p1", State: "R", CPUTime: 0, WaitTime: 0, Switches: 0},
	}

	// two ticks of starvation
	engine.Calculate(starving, base, 2*time.Second)
	engine.Calculate(starving, base, 2*time.Second)

	// recovery tick
	engine.Calculate(recovered, base, 2*time.Second)

	// starvation again — ticks should have reset so not starved yet
	results := engine.Calculate(starving, base, 2*time.Second)
	if results[0].IsStarved {
		t.Error("ticks should have reset after recovery")
	}
}

func TestCalculate_SwitchRate(t *testing.T) {
	engine := NewEngine(testConfig())

	current := []proc.ProcessStatus{
		{PID: 1, Name: "p1", State: "S", CPUTime: 0, WaitTime: 0, Switches: 10},
	}
	previous := []proc.ProcessStatus{
		{PID: 1, Name: "p1", State: "S", CPUTime: 0, WaitTime: 0, Switches: 0},
	}

	results := engine.Calculate(current, previous, 2*time.Second)

	expected := 5.0 // 10 switches / 2 seconds
	if results[0].SwitchRate != expected {
		t.Errorf("expected switch rate %f, got %f", expected, results[0].SwitchRate)
	}
}