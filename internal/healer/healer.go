//go:build linux

package healer

import (
	appconfig "SchedLens/internal/config"
	"SchedLens/internal/metrics"
	"fmt"
	"syscall"
	"time"
)

type Healer struct {
	lastReniced map[int]time.Time
	boostedPIDs map[int]bool
	cfg         appconfig.HealerConfig
}

func NewHealer(cfg appconfig.HealerConfig) *Healer {
	return &Healer{
		lastReniced: make(map[int]time.Time),
		boostedPIDs: make(map[int]bool),
		cfg:         cfg,
	}
}

func (h *Healer) Heal(results []metrics.MetricResult) {
	if !h.cfg.Enabled {
		return
	}

	for _, r := range results {
		if r.IsStarved {
			h.maybeHeal(r.PID, r.Name)
		} else {
			h.maybeRollback(r.PID, r.Name)
		}
	}
}

func (h *Healer) maybeHeal(pid int, name string) {
	// Get the current nice value--> If not zero then dont heal (user policy)
	currentNice, err := syscall.Getpriority(syscall.PRIO_PROCESS, pid)
	if err != nil && currentNice != 0 {
		return
	}

	// Check cooldown — don't renice if we did it recently
	lastTime, exists := h.lastReniced[pid]
	if exists && time.Since(lastTime).Seconds() < float64(h.cfg.CooldownSeconds) {
		return
	}

	// Boost the process
	err = syscall.Setpriority(syscall.PRIO_PROCESS, pid, h.cfg.ReniceValue)
	if err != nil {
		return
	}

	h.lastReniced[pid] = time.Now()
	h.boostedPIDs[pid] = true
	fmt.Printf("[HEALER] Boosted PID %d (%s) → nice %d\n", pid, name, h.cfg.ReniceValue) //copy pasted
}

func (h *Healer) maybeRollback(pid int, name string) {
	// Only roll back if we boosted
	if !h.boostedPIDs[pid] {
		return
	}
	// Process recovered — restore to 0
	err := syscall.Setpriority(syscall.PRIO_PROCESS, pid, 0)
	if err != nil {
		return
	}

	delete(h.boostedPIDs, pid)
	delete(h.lastReniced, pid)
	fmt.Printf("[HEALER] Rolled back PID %d (%s) → nice 0\n", pid, name)
}

// Cleanup removes dead processes from our maps --> If the process is dead in middle then we just have to delete those maps
func (h *Healer) Cleanup(currentPIDs map[int]bool) {
	for pid := range h.boostedPIDs {
		if !currentPIDs[pid] {
			delete(h.boostedPIDs, pid)
			delete(h.lastReniced, pid)
		}
	}
}
