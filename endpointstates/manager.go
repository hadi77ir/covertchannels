package endpointstates

import (
	"github.com/hadi77ir/timingoutmap"
	"go.uber.org/atomic"
	"net"
	"time"
)

type Manager struct {
	storage *timingoutmap.TimingoutMap[string, *EndpointState]
}

func NewManager() *Manager {
	return &Manager{
		storage: timingoutmap.New[string, *EndpointState](time.Duration(3)*time.Minute, true),
	}
}

func (c *Manager) Get(addr net.UDPAddr) (*EndpointState, error) {
	return c.storage.Get(addr.String())
}

func (c *Manager) GetOrNew(addr net.UDPAddr) *EndpointState {
	return c.storage.GetOrNew(addr.String(),
		&EndpointState{
			address:        addr,
			sequenceNumber: atomic.NewInt32(0),
		})
}
func (c *Manager) Clean() {
	c.storage.Clear()
}

func (c *Manager) CleanDead() {
	c.storage.CleanDead()
}
