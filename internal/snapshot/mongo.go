package snapshot

import (
	"SchedLens/internal/metrics"
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type ProcessSnapshot struct {
	PID           int     `bson:"pid"`
	Name          string  `bson:"name"`
	FairnessScore float64 `bson:"fairness_score"`
	IsStarved     bool    `bson:"is_starved"`
	WaitTime      uint64  `bson:"wait_time"`
	CPUTime       uint64  `bson:"cpu_time"`
}

type SnapshotDocument struct {
	Timestamp time.Time         `bson:"timestamp"`
	Processes []ProcessSnapshot `bson:"processes"`
}

type MongoDB struct {
	client     *mongo.Client
	collection *mongo.Collection
}

func NewMongoDB(uri string) (*MongoDB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(opts) // v2: no ctx in Connect
	if err != nil {
		return nil, err
	}

	// Ping to confirm connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	collection := client.Database("schedlens").Collection("snapshots")

	return &MongoDB{
		client:     client,
		collection: collection,
	}, nil
}

func (m *MongoDB) Insert(results []metrics.MetricResult) error {
	processes := make([]ProcessSnapshot, 0, len(results))
	for _, r := range results {
		processes = append(processes, ProcessSnapshot{
			PID:           r.PID,
			Name:          r.Name,
			FairnessScore: r.FairnessScore,
			IsStarved:     r.IsStarved,
			WaitTime:      r.WaitTimeDelta,
			CPUTime:       r.CPUTimeDelta,
		})
	}

	doc := SnapshotDocument{
		Timestamp: time.Now(),
		Processes: processes,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := m.collection.InsertOne(ctx, doc)
	return err
}

func (m *MongoDB) Query(from, to time.Time) ([]SnapshotDocument, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{
		"timestamp": bson.M{
			"$gte": from,
			"$lte": to,
		},
	}

	cursor, err := m.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []SnapshotDocument
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	return results, nil
}
