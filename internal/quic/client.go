package quic

import (
	"context"
	"crypto/tls"
	"fmt"

	"github.com/quic-go/quic-go"
)

type Client struct {
	quicConfig *quic.Config
	tlsConfig  *tls.Config
}

func NewClient(tlsConfig *tls.Config, quicConfig *quic.Config) *Client {
	return &Client{
		tlsConfig:  tlsConfig,
		quicConfig: quicConfig,
	}
}

func (c *Client) Connect(ctx context.Context, addr string) (quic.Connection, error) {
	conn, err := quic.DialAddr(ctx, addr, c.tlsConfig, c.quicConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to dial QUIC address: %w", err)
	}

	// Connection established successfully
	return conn, nil
}

func (c *Client) SendStream(conn quic.Connection, data []byte) error {
	stream, err := conn.OpenStreamSync(context.Background())
	if err != nil {
		return fmt.Errorf("failed to open stream: %w", err)
	}
	defer stream.Close()

	_, err = stream.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write to stream: %w", err)
	}

	return nil
}

func (c *Client) Close(conn quic.Connection) error {
	if err := conn.CloseWithError(0, "closing connection"); err != nil {
		return fmt.Errorf("failed to close connection: %w", err)
	}
	return nil
}