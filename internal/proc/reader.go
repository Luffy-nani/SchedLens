package proc

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type ProcessStatus struct {
	PID       int
	Name      string
	State     string
	CPUTime   uint64
	WaitTime  uint64
	RunTime   uint64
	Switches  uint64
	Timestamp time.Time
}

func ReadAllProcesses() ([]ProcessStatus, error) {
	var processes []ProcessStatus

	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {

		if !entry.IsDir() {
			continue
		}
		pidStr := entry.Name()
		if _, err := strconv.Atoi(pidStr); err != nil {
			continue
		}
		processStat, err := readOne(pidStr)
		if err != nil {
			continue
		}

		processes = append(processes, processStat)
	}

	return processes, nil
}

func readOne(pidStr string) (ProcessStatus, error) {
	statData, err := os.ReadFile(filepath.Join("/proc", pidStr, "stat"))
	if err != nil {
		return ProcessStatus{}, err
	}
	statFields := strings.Fields(string(statData))
	if len(statFields) < 15 {
		return ProcessStatus{}, err
	}
	pid, _ := strconv.Atoi(pidStr)
	name := statFields[1]
	state := statFields[2]
	utime, _ := strconv.ParseUint(statFields[13], 10, 64)
	stime, _ := strconv.ParseUint(statFields[14], 10, 64)

	schedData, err := os.ReadFile(filepath.Join("/proc", pidStr, "schedstat"))
	if err != nil {
		return ProcessStatus{}, err
	}
	schedFields := strings.Fields(string(schedData)) //we change schedData to strings because the fileds method accepts only string input but schedData is raw binary bytes

	if len(schedFields) < 3 {
		return ProcessStatus{}, err
	}
	runTime, _ := strconv.ParseUint(schedFields[0], 10, 64)
	waitTime, _ := strconv.ParseUint(schedFields[1], 10, 64)
	switches, _ := strconv.ParseUint(schedFields[2], 10, 64)

	return ProcessStatus{
		PID:       pid,
		Name:      name,
		State:     state,
		CPUTime:   utime + stime,
		WaitTime:  waitTime,
		RunTime:   runTime,
		Switches:  switches,
		Timestamp: time.Now(),
	}, nil

}
