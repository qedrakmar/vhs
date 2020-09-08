package capture

import (
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/go-errors/errors"
	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	"github.com/gramLabs/vhs/session"

	// See https://pkg.go.dev/github.com/google/gopacket?tab=doc#hdr-A_Final_Note
	_ "github.com/google/gopacket/layers"
)

// NewListener creates a new listener.
func NewListener(cap *Capture) Listener {
	return &listener{
		Capture: cap,
		packets: make(chan gopacket.Packet),
	}
}

// Listener listens for network traffic on a
// given address and port.
type Listener interface {
	Packets() <-chan gopacket.Packet
	Listen(*session.Context)
	Close()
}

type listener struct {
	Capture *Capture

	packets chan gopacket.Packet

	handleMu sync.Mutex
	handles  []*pcap.Handle
}

// Packets retrieves a channel for all packets
// captured by this listener.
func (l *listener) Packets() <-chan gopacket.Packet {
	return l.packets
}

// Listen starts listening.
func (l *listener) Listen(ctx *session.Context) {
	for _, i := range l.Capture.Interfaces {
		go func(ii pcap.Interface) {
			h, err := l.newHandle(ii)
			if err != nil {
				ctx.Errors <- err
				return
			}
			if h == nil {
				return
			}
			l.readPackets(ctx, h, h.LinkType())
		}(i)
	}
}

func (l *listener) newHandle(i pcap.Interface) (*pcap.Handle, error) {
	handle, err := l.newActiveHandler(i.Name)
	if err != nil {
		log.Printf("failed to create new inactive handle: %v\n", err)
		return nil, nil
	}

	if err := handle.SetBPFFilter(filter(l.Capture, i)); err != nil {
		return nil, errors.Errorf("failed to set filter: %w", err)
	}

	l.handleMu.Lock()
	l.handles = append(l.handles, handle)
	l.handleMu.Unlock()

	return handle, nil
}

func (l *listener) readPackets(ctx *session.Context, dataSource gopacket.PacketDataSource, decoder gopacket.Decoder) {
	source := gopacket.NewPacketSource(dataSource, decoder)
	source.Lazy = true
	source.NoCopy = true

	for {
		p, err := source.NextPacket()
		if errors.Is(err, io.EOF) {
			continue
		}
		if err != nil {
			fmt.Printf("read packet failed: %v\n", err)
			continue
		}
		l.packets <- p
	}
}

func (l *listener) newActiveHandler(name string) (*pcap.Handle, error) {
	inactive, err := pcap.NewInactiveHandle(name)
	if err != nil {
		return nil, errors.Errorf("failed to create inactive handle: %w", err)
	}

	defer inactive.CleanUp()

	inactive.SetPromisc(true)

	// TODO(andrewhare): configure these
	inactive.SetTimeout(2000 * time.Millisecond)
	inactive.SetImmediateMode(true)
	inactive.SetSnapLen(65536)

	handle, err := inactive.Activate()
	if err != nil {
		return nil, errors.Errorf("failed to activate %s: %w", name, err)
	}

	return handle, nil
}

// Close closes the listener and all open handles.
func (l *listener) Close() {
	l.handleMu.Lock()
	defer l.handleMu.Unlock()

	for _, handle := range l.handles {
		handle.Close()
	}
}
