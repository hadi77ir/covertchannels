package channel

import (
	"github.com/hadi77ir/uoicmp/endpointstates"
	"net"
)

type DuplexChannel interface {
	net.PacketConn
	StateManager() *endpointstates.Manager
}
