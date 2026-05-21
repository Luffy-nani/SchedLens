package main

import (
	"SchedLens/internal/api"
	"SchedLens/internal/cli"
	"SchedLens/internal/exporter"
	"SchedLens/internal/metrics"
	"SchedLens/internal/proc"
	"SchedLens/internal/snapshot"
	"fmt"
	"time"
)

func main() {
	// Start OTel exporter
	exp, err := exporter.NewOtelExporter()
	if err != nil {
		fmt.Println("Error starting OTel exporter:", err)
		return
	}
	exp.Start(2222)

	// Connect MongoDB
	db, err := snapshot.NewMongoDB("mongodb://localhost:27017")
	if err != nil {
		fmt.Println("Error connecting to MongoDB:", err)
		return
	}

	// Start Gin API in background
	server := api.NewServer(db)
	go server.Run(":8080")

	// First snapshot
	snapshot1, _ := proc.ReadAllProcesses()
	fmt.Println("SchedLens started. Collecting data...")
	time.Sleep(2 * time.Second)

	// Main loop
	for {
		snapshot2, _ := proc.ReadAllProcesses()
		results := metrics.Calculate(snapshot2, snapshot1, 2*time.Second)

		// Fan-out to all three simultaneously
		go exp.Update(results)
		go db.Insert(results)
		cli.Render(results)

		snapshot1 = snapshot2
		time.Sleep(2 * time.Second)
	}
}
