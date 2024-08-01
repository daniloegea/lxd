//go:build freebsd
// +build freebsd

package ucred

import (
	"context"
	"fmt"
	"net"

	"github.com/canonical/lxd/lxd/endpoints/listeners"
	"github.com/canonical/lxd/lxd/request"
)

// ErrNotUnixSocket is returned when the underlying connection isn't a unix socket.
var ErrNotUnixSocket = fmt.Errorf("Connection isn't a unix socket")

// GetCred returns the credentials from the remote end of a unix socket.
func GetCred(conn *net.UnixConn) (*interface{}, error) {
	return nil, nil
}

// GetConnFromContext extracts the connection from the request context on a HTTP listener.
func GetConnFromContext(ctx context.Context) net.Conn {
	return ctx.Value(request.CtxConn).(net.Conn)
}

// GetCredFromContext extracts the unix credentials from the request context on a HTTP listener.
func GetCredFromContext(ctx context.Context) (*interface{}, error) {
	conn := GetConnFromContext(ctx)
	unixConnPtr, ok := conn.(*net.UnixConn)
	if !ok {
		bufferedUnixConnPtr, ok := conn.(listeners.BufferedUnixConn)
		if !ok {
			return nil, ErrNotUnixSocket
		}

		unixConnPtr = bufferedUnixConnPtr.Unix()
	}

	return GetCred(unixConnPtr)
}
