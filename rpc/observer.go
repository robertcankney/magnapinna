package rpc

import (
	"io"
	"io/ioutil"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// TODO convert observability functions to gRPC interceptors
type observer struct {
	errs     *prometheus.CounterVec
	requests *prometheus.CounterVec
	active   prometheus.Gauge
	logger   *zap.SugaredLogger
}

func NewObserver(w io.Writer) *observer {
	conf := zap.NewProductionConfig()
	conf.OutputPaths = toZapKeys(w)
	l, err := conf.Build()
	if err != nil {
		panic("failed to build zap logger - this is a code issue")
	}
	return &observer{
		logger: l.Sugar(),
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "grpc_requests",
			Help: "Counter of gRPC requests by calling context in application.",
		}, []string{"caller"}),
		errs: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "grpc_errors",
			Help: "Counter of gRPC errors by calling context in application.",
		}, []string{"caller"}),
		active: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "active_sessions",
			Help: "Number of active Magnapinna client sessions.",
		}),
	}
}

// Describe is part of the implememtation of prometheus.Collector.
func (o *observer) Describe(desc chan<- *prometheus.Desc) {
	o.requests.Describe(desc)
	o.errs.Describe(desc)
	o.active.Describe(desc)
}

// Collect is part of the implememtation of prometheus.Collector.
func (o *observer) Collect(coll chan<- prometheus.Metric) {
	o.requests.Collect(coll)
	o.errs.Collect(coll)
	o.active.Collect(coll)
}

func (o *observer) ObserveGRPCCall(context string, err error) {
	if err != nil {
		o.errs.WithLabelValues(context).Inc()
		o.logger.Errorw("gRPC call failed", "err", err)
	}
	o.requests.WithLabelValues(context).Inc()
}

func (o *observer) ObserveClientAddition(id string) {
	o.logger.Infow("new client added", "client", id)
	o.active.Inc()
}

func (o *observer) ObserveClientDeletion(id string) {
	o.logger.Infow("client deleted", "client", id)
	o.active.Inc()
}

// toZapKeys is a shim to deal zap requiring specific strings for writing to stdout/err, or not writing at all
func toZapKeys(w io.Writer) []string {
	switch w {
	case ioutil.Discard:
		return []string{}
	default:
		return []string{"stdout"}
	}
}
