package rpc

import (
	"context"
	"magnapinna/api"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/oauth"
)

// Client...
type Client struct {
	grpc     api.MagnapinnaClient
	observer *observer
	id       string
	lease    time.Duration
	timeout  time.Duration
	ctx      context.Context
}

type ClientOpts struct {
	Addr    string
	ID      string
	Timeout time.Duration
	Token   oauth2.Token
	Context context.Context
}

func NewClient(opts ClientOpts) (*Client, error) {
	conn, err := grpc.Dial(opts.Addr, grpc.WithPerRPCCredentials(oauth.NewOauthAccess(&opts.Token)))
	if err != nil {
		return nil, err
	}

	return &Client{
		grpc:    api.NewMagnapinnaClient(conn),
		id:      opts.ID,
		ctx:     opts.Context,
		timeout: opts.Timeout,
	}, nil
}

// TODO double-check that GRPC will correctly handle the timeout
func (c *Client) Register() (*api.Lease, error) {
	ctx, cancel := context.WithTimeout(c.ctx, c.timeout)
	defer cancel()

	return c.grpc.Register(ctx, &api.Registration{
		Duration:   int32(c.lease.Seconds()),
		Identifier: c.id,
	})
}

func (c *Client) CheckRegistration() (*api.Lease, error) {
	ctx, cancel := context.WithTimeout(c.ctx, c.timeout)
	defer cancel()
	return c.grpc.CheckRegistration(ctx, &api.Registration{
		Duration:   int32(c.lease.Seconds()),
		Identifier: c.id,
	})
}

func (c *Client) Deregister() (*api.Lease, error) {
	ctx, cancel := context.WithTimeout(c.ctx, c.timeout)
	defer cancel()
	return c.grpc.Deregister(ctx, &api.Registration{
		Duration:   int32(c.lease.Seconds()),
		Identifier: c.id,
	})
}

func (c *Client) JoinCluster(ctx context.Context, opts ...grpc.CallOption) (api.Magnapinna_JoinClusterClient, error) {
	return nil, nil
}

func (c *Client) StartSession(ctx context.Context, opts ...grpc.CallOption) (api.Magnapinna_StartSessionClient, error) {
	return nil, nil
}
