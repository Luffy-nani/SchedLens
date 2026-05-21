package exporter

import (
	"SchedLens/internal/metrics"
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/prometheus"
	api "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"
)

type OtelExporter struct {
	meter           api.Meter
	fairnessGauge   api.Float64ObservableGauge
	starvationGauge api.Float64ObservableGauge
	switchRateGauge api.Float64ObservableGauge
	waitCounter     api.Float64Counter
	cpuCounter      api.Float64Counter

	mu      sync.Mutex
	current []metrics.MetricResult
}

// Gauges use callbacks because OpenTelemetry pulls their values during metric collection.
// the function for counter update is outside of this because we have to call it manually
func NewOtelExporter() (*OtelExporter, error) {

	//Boiler plate--> To set up the prometheus exporter and create meter for metrics
	exporter, err := prometheus.New()
	if err != nil {
		return nil, err
	}
	provider := metric.NewMeterProvider(metric.WithReader(exporter))
	meter := provider.Meter("schedlens")

	// All metrics
	fairnessGauge, _ := meter.Float64ObservableGauge("schedlens_fairness_score")
	starvationGauge, _ := meter.Float64ObservableGauge("schedlens_is_starved")
	switchRateGauge, _ := meter.Float64ObservableGauge("schedlens_switch_rate")
	waitCounter, _ := meter.Float64Counter("schedlens_wait_time_delta")
	cpuCounter, _ := meter.Float64Counter("schedlens_cpu_time_delta")

	// Exporter instance that stores current metric state and instruments.
	e := &OtelExporter{
		meter:           meter,
		fairnessGauge:   fairnessGauge,
		starvationGauge: starvationGauge,
		switchRateGauge: switchRateGauge,
		waitCounter:     waitCounter,
		cpuCounter:      cpuCounter,
	}

	// callback
	meter.RegisterCallback(func(ctx context.Context, o api.Observer) error {
		e.mu.Lock()
		defer e.mu.Unlock()
		for _, r := range e.current {
			labels := api.WithAttributes(
				attribute.String("pid", fmt.Sprintf("%d", r.PID)),
				attribute.String("name", r.Name),
			)
			o.ObserveFloat64(fairnessGauge, r.FairnessScore, labels) //returns the fariness gauge metris

			//Changes the boolean starved to an int to show in the metrics(as grafana can only show numbers in graphs)
			starvedVal := 0.0
			if r.IsStarved {
				starvedVal = 1.0
			}
			o.ObserveFloat64(starvationGauge, starvedVal, labels)   //returns the starvation gauge
			o.ObserveFloat64(switchRateGauge, r.SwitchRate, labels) //returns the switchrate gauge
		}
		return nil

	}, fairnessGauge, starvationGauge, switchRateGauge)

	return e, nil
}

// Function for counter update
func (e *OtelExporter) Update(results []metrics.MetricResult) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.current = results

	// Counters are pushed directly
	for _, r := range results {
		labels := api.WithAttributes(
			attribute.String("pid", fmt.Sprintf("%d", r.PID)),
			attribute.String("name", r.Name),
		)
		e.waitCounter.Add(context.Background(), float64(r.WaitTimeDelta), labels)
		e.cpuCounter.Add(context.Background(), float64(r.CPUTimeDelta), labels)
	}
}

// Start exposes /metrics endpoint
func (e *OtelExporter) Start(port int) {
	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}
