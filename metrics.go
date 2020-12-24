package main

import (
	"time"

	"github.com/povilasv/prommod"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	updateMetrics = map[string]prometheus.ObserverVec{}
)

func initMetrics() {
	ns := "hkrelay"
	prometheus.MustRegister(prommod.NewCollector(ns), prometheus.NewBuildInfoCollector())

	// on: On, acc: Accessory (identify event)
	for _, sub := range []string{"on", "acc"} {
		updateMetrics[sub+"UpdateDurationHist"] = promauto.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: ns,
			Subsystem: sub,
			Name:      "update_duration_seconds",
			Buckets:   []float64{.01, .05, .1, .25, .5, 1, 2.5, 5, 10},
		}, []string{"event"})
		updateMetrics[sub+"UpdateDurationSumm"] = promauto.NewSummaryVec(prometheus.SummaryOpts{
			Namespace:  ns,
			Subsystem:  sub,
			Name:       "update_duration_quantile_seconds",
			Objectives: map[float64]float64{0.1: 0.01, 0.5: 0.01, 0.9: 0.01, 0.99: 0.001, 0.999: 0.0001},
		}, []string{"event"})
	}
}

func observeUpdateDuration(sub, event string, start time.Time) {
	diff := time.Since(start)
	l := prometheus.Labels{"event": event}
	updateMetrics[sub+"UpdateDurationHist"].With(l).Observe(diff.Seconds())
	updateMetrics[sub+"UpdateDurationSumm"].With(l).Observe(diff.Seconds())
}
