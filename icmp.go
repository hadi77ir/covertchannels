package channel

import (
	"encoding/binary"
	"github.com/hadi77ir/uoicmp/endpointstates"
	"github.com/valyala/bytebufferpool"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"net"
	"time"
)

const MaxPacketSize = 1500

// MaxPayloadSize equals to MaxPacketSize minus icmp header length + 4 for magic number
const MaxPayloadSize = 1468
const ProtocolICMP = 1

type ICMPChannel struct {
	endpoints  *endpointstates.Manager
	conn       *icmp.PacketConn
	isServer   bool
	localAddr  *net.UDPAddr
	magic      uint32
	magicBytes []byte
	bufferPool *bytebufferpool.Pool
}

// OpenChannel4 opens an ICMPv4 channel, enabling sending and receiving datagrams.
func OpenChannel4(magic uint32, isServer bool) (DuplexChannel, error) {
	conn, err := icmp.ListenPacket("ip4:icmp", "")
	if err != nil {
		return nil, err
	}

	val := &ICMPChannel{
		endpoints:  endpointstates.NewManager(),
		conn:       conn,
		isServer:   isServer,
		magic:      magic,
		bufferPool: &bytebufferpool.Pool{},
		localAddr:  &net.UDPAddr{IP: net.IP("0.0.0.0"), Port: 0},
	}
	val.magicBytes = make([]byte, 4)
	binary.BigEndian.PutUint32(val.magicBytes, magic)

	return val, nil
}

func (c *ICMPChannel) StateManager() *endpointstates.Manager {
	return c.endpoints
}

func (c *ICMPChannel) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	// find the target endpoint
	udpAddr, ok := addr.(*net.UDPAddr)
	if !ok {
		return 0, net.InvalidAddrError("only UDPAddr is allowed")
	}
	endpointState := c.endpoints.GetOrNew(*udpAddr)

	if len(p) > MaxPayloadSize {
		p = p[:MaxPayloadSize]
	}

	// prepend the magic number
	p = append(c.magicBytes, p...)

	body := &icmp.Echo{
		ID:   udpAddr.Port,
		Seq:  int(endpointState.NewSequenceId()),
		Data: p,
	}

	// the initiator (client) sends "echo requests" and the server responds with "echo replies".
	proto := ipv4.ICMPTypeEcho
	if c.isServer {
		proto = ipv4.ICMPTypeEcho
	}

	msg := &icmp.Message{
		Type: proto,
		Code: 0, // code must be zero, per RFC792
		Body: body,
	}

	msgBytes, err := msg.Marshal(nil)
	if err != nil {
		return 0, err
	}

	return c.conn.WriteTo(msgBytes, addr)
}

func (c *ICMPChannel) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	var noAddr net.Addr
	buffer := c.bufferPool.Get()
	// ensure buffer size is good
	if buffer.Len() != MaxPacketSize {
		buffer.B = make([]byte, MaxPacketSize)
	}
	defer c.bufferPool.Put(buffer)

	n, srcAddr, err := c.conn.ReadFrom(buffer.B)

	// drop redundant bytes
	buffer.B = buffer.B[:n]

	// in case of errors (such as timeout)...
	if err != nil {
		return 0, noAddr, err
	}
	// ...and empty packets
	if n <= 0 {
		return 0, noAddr, nil
	}

	message, err := icmp.ParseMessage(ProtocolICMP, buffer.B)

	// don't accept malformed icmp messages
	if err != nil {
		return 0, noAddr, net.UnknownNetworkError("malformed message")
	}
	srcIPAddr, ok := srcAddr.(*net.IPAddr)

	if !ok {
		return 0, srcAddr, net.InvalidAddrError("address should be convertible to IPAddr")
	}

	if body, ok := message.Body.(*icmp.Echo); ok {
		// if packet doesn't start with magic, drop
		packetMagic := binary.BigEndian.Uint32(body.Data)
		if packetMagic != c.magic {
			// ignore message
			return 0, srcAddr, nil
		}
		return copy(p, body.Data), &net.UDPAddr{IP: srcIPAddr.IP, Zone: srcIPAddr.Zone, Port: body.ID}, nil
	}

	// ignore message
	return 0, srcAddr, nil
}

func (c *ICMPChannel) LocalAddr() net.Addr {
	return c.localAddr
}
func (c *ICMPChannel) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}
func (c *ICMPChannel) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}
func (c *ICMPChannel) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}

func (c *ICMPChannel) Close() error {
	if err := c.conn.Close(); err != nil {
		return err
	}
	return nil
}
