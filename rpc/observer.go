package rpc

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"magnapinna/api"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// TODO convert observability functions to gRPC interceptors
type observer struct {
	errs       *prometheus.CounterVec
	requests   *prometheus.CounterVec
	throughput *prometheus.CounterVec
	active     prometheus.Gauge
	logger     *zap.SugaredLogger
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
		throughput: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "grpc_errors",
			Help: "Counter of gRPC errors by calling context in application.",
		}, []string{"caller"}),
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

func (o *observer) ObserveThroughput(context string, length int) {
	o.throughput.WithLabelValues(context).Add(float64(length))
}

func (o *observer) UnaryObserver() func(ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler) (interface{}, error) {

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		switch info.FullMethod {
		case "/Magnapinna/Register":
			//TODO fix this and below to get ID (maybe?) or move elsewhere
			r, ok := req.(api.Registration)
			if !ok {
				return nil, fmt.Errorf("wrong type for gRPC endpoint: expected Registration, got %T", req)
			}
			o.ObserveClientAddition(r.Identifier)
		}

		resp, err := handler(ctx, req)
		o.ObserveGRPCCall(info.FullMethod, err)
		return resp, err
	}
}

func (o *observer) StreamObserver() func(srv interface{},
	stream grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler) error {

	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		var i interface{}
		err := stream.RecvMsg(i)
		if err != nil {
			return fmt.Errorf("failed to receive gRPC message: %w", err)
		}

		length := 0
		switch info.FullMethod {
		case "/Magnapinna/StartSession":
			//TODO fix this and below to get ID (maybe?) or move elsewhere
			r, ok := i.(api.Command)
			if !ok {
				return fmt.Errorf("wrong type for gRPC endpoint: expected Registration, got %T", i)
			}
			length = len(r.Contents)
		case "/Magnapinna/JoinCluster":
			r, ok := i.(api.Output)
			if !ok {
				return fmt.Errorf("wrong type for gRPC endpoint: expected Registration, got %T", i)
			}
			length = len(r.Contents)
		}
		o.ObserveThroughput(info.FullMethod, length)

		err = handler(srv, stream)
		o.ObserveGRPCCall(info.FullMethod, err)
		return err
	}
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
