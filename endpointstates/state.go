package endpointstates

import "net"
import "go.uber.org/atomic"

type EndpointState struct {
	address        net.UDPAddr
	sequenceNumber *atomic.Int32
}

func (e *EndpointState) NewSequenceId() int32 {
	return e.sequenceNumber.Inc()
}

func (e *EndpointState) Reset() {
	e.sequenceNumber.Store(0)
}
