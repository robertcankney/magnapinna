package rpc

import (
	"github.com/prometheus/client_golang/prometheus"
)

type observer struct {
	errs prometheus.CounterVec
}

// Describe sends the super-set of all possible descriptors of metrics
// collected by this Collector to the provided channel and returns once
// the last descriptor has been sent. The sent descriptors fulfill the
// consistency and uniqueness requirements described in the Desc
// documentation.
//
// It is valid if one and the same Collector sends duplicate
// descriptors. Those duplicates are simply ignored. However, two
// different Collectors must not send duplicate descriptors.
//
// Sending no descriptor at all marks the Collector as “unchecked”,
// i.e. no checks will be performed at registration time, and the
// Collector may yield any Metric it sees fit in its Collect method.
//
// This method idempotently sends the same descriptors throughout the
// lifetime of the Collector. It may be called concurrently and
// therefore must be implemented in a concurrency safe way.
//
// If a Collector encounters an error while executing this method, it
// must send an invalid descriptor (created with NewInvalidDesc) to
// signal the error to the registry.
func (o *observer) Describe(_ chan<- *prometheus.Desc) {
	panic("not implemented") // TODO: Implement
}

// Collect is called by the Prometheus registry when collecting
// metrics. The implementation sends each collected metric via the
// provided channel and returns once the last metric has been sent. The
// descriptor of each sent metric is one of those returned by Describe
// (unless the Collector is unchecked, see above). Returned metrics that
// share the same descriptor must differ in their variable label
// values.
//
// This method may be called concurrently and must therefore be
// implemented in a concurrency safe way. Blocking occurs at the expense
// of total performance of rendering all registered metrics. Ideally,
// Collector implementations support concurrent readers.
func (o *observer) Collect(_ chan<- prometheus.Metric) {
	panic("not implemented") // TODO: Implement
}

func (o *observer) ObserveJoinFailure(err error) {

}

func (o *observer) ObserveStartFailure(err error) {

}

func (o *observer) ObserveCommandFailure(err error) {

}

func (o *observer) ObserveOutputFailure(err error) {

}
