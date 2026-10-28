package discord

import (
	"context"
	"net"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// gatewayReadTimeout is well past Discord's ~41s heartbeat interval. A healthy gateway
	// always sends a heartbeat ACK within it.
	gatewayReadTimeout  = 2 * time.Minute
	gatewayWriteTimeout = 30 * time.Second
)

// gatewayDialer bounds every gateway read and write. discordgo sets no deadlines, so a
// connection that stops sending, even mid-handshake inside Session.Open, blocks forever while
// holding the session lock and its reconnect loop never runs.
func gatewayDialer() *websocket.Dialer {
	dialer := *websocket.DefaultDialer
	dialer.NetDialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		conn, err := new(net.Dialer).DialContext(ctx, network, addr)
		if err != nil {
			return nil, err
		}
		return deadlineConn{conn}, nil
	}
	return &dialer
}

type deadlineConn struct {
	net.Conn
}

func (c deadlineConn) Read(p []byte) (int, error) {
	if err := c.Conn.SetReadDeadline(time.Now().Add(gatewayReadTimeout)); err != nil {
		return 0, err
	}
	return c.Conn.Read(p)
}

func (c deadlineConn) Write(p []byte) (int, error) {
	if err := c.Conn.SetWriteDeadline(time.Now().Add(gatewayWriteTimeout)); err != nil {
		return 0, err
	}
	return c.Conn.Write(p)
}
