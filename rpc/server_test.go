package rpc

import (
	"context"
	"io/ioutil"
	"magnapinna/api"
	"testing"
	"time"

	"google.golang.org/grpc"
)

// TODO - convert these to use stretchr/testify
type testRepository struct {
	store  func(context.Context, *api.Lease) error
	fetch  func(context.Context, *api.Registration) (*api.Lease, error)
	delete func(context.Context, *api.Lease) error
}

func (t *testRepository) StoreLease(ctx context.Context, lease *api.Lease) error {
	return t.store(ctx, lease)
}
func (t *testRepository) FetchLease(ctx context.Context, registration *api.Registration) (*api.Lease, error) {
	return t.fetch(ctx, registration)
}
func (t *testRepository) DeleteLease(ctx context.Context, lease *api.Lease) error {
	return t.delete(ctx, lease)
}

// delete and store have the same function signature so will be using the same functions for them
// positive cases have a very short sleep to test function cancellation
func leasePositive(ctx context.Context, lease *api.Lease) error {
	time.Sleep(time.Millisecond)
	select {
	case <-ctx.Done():
		return context.DeadlineExceeded
	default:
		return nil
	}
}

func fetchPositive(ctx context.Context, registration *api.Registration) (*api.Lease, error) {
	time.Sleep(time.Millisecond)
	select {
	case <-ctx.Done():
		return nil, context.DeadlineExceeded
	default:
		return &api.Lease{}, nil
	}
}

func leaseNegative(ctx context.Context, lease *api.Lease) error {
	return RepositoryError{}
}

func fetchNegative(ctx context.Context, registration *api.Registration) (*api.Lease, error) {
	return nil, RepositoryError{}
}

// tests for all unary server functions
// timeout is in microseconds to allow for ease of testing context timeouts against
// the repo's hardcoded 1 millisecond sleep
func TestUnaryFunctions(t *testing.T) {
	cases := []struct {
		valid   bool
		name    string
		lease   func(ctx context.Context, lease *api.Lease) error
		reg     func(ctx context.Context, registration *api.Registration) (*api.Lease, error)
		timeout int
		ls      *api.Lease
		rs      *api.Registration
	}{
		{
			name:    "positive case",
			valid:   true,
			timeout: 5000,
			lease:   leasePositive,
			reg:     fetchPositive,
			ls: &api.Lease{
				Identifier: "foo",
				Expiration: 1000,
			},
			rs: &api.Registration{
				Identifier: "foo",
				Duration:   1000,
			},
		},
		{
			name:    "timeout case",
			timeout: 1,
			lease:   leasePositive,
			reg:     fetchPositive,
			ls: &api.Lease{
				Identifier: "foo",
				Expiration: 1000,
			},
			rs: &api.Registration{
				Identifier: "foo",
				Duration:   1000,
			},
		},
		{
			name:    "invalid requests",
			timeout: 5000,
			lease:   leasePositive,
			reg:     fetchPositive,
			ls: &api.Lease{
				Identifier: "foo",
			},
			rs: &api.Registration{
				Duration: 1000,
			},
		},
		{
			name:    "negative case",
			timeout: 5000,
			lease:   leaseNegative,
			reg:     fetchNegative,
			ls: &api.Lease{
				Identifier: "foo",
				Expiration: 1000,
			},
			rs: &api.Registration{
				Identifier: "foo",
				Duration:   1000,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			srv := Server{
				srv:     &grpc.Server{},
				ctx:     context.Background(),
				timeout: time.Duration(c.timeout) * time.Microsecond,
				repository: &testRepository{
					store:  c.lease,
					fetch:  c.reg,
					delete: c.lease,
				},
				observer: NewObserver(ioutil.Discard),
			}

			t.Run("Register", func(t *testing.T) {
				ctx, cancel := context.WithTimeout(srv.ctx, srv.timeout)
				defer cancel()
				_, err := srv.Register(ctx, c.rs)

				if err != nil && c.valid {
					t.Errorf("unexpected failure: %w", err)
				}
				if err == nil && !c.valid {
					t.Error("did not fail as expected")
				}
			})

			t.Run("CheckRegistration", func(t *testing.T) {
				ctx, cancel := context.WithTimeout(srv.ctx, srv.timeout)
				defer cancel()
				_, err := srv.CheckRegistration(ctx, c.rs)

				if err != nil && c.valid {
					t.Errorf("unexpected failure: %w", err)
				}
				if err == nil && !c.valid {
					t.Error("did not fail as expected")
				}
			})

			t.Run("Deregister", func(t *testing.T) {
				ctx, cancel := context.WithTimeout(srv.ctx, srv.timeout)
				defer cancel()
				_, err := srv.Deregister(ctx, c.rs)

				if err != nil && c.valid {
					t.Errorf("unexpected failure: %w", err)
				}
				if err == nil && !c.valid {
					t.Error("did not fail as expected")
				}
			})

		})
	}
}
