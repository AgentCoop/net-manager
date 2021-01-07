package netmanager

import (
	"fmt"
	"github.com/google/go-cmp/cmp"
	"net"
	"sync"
)

const (
	DefaultReadBufLen = 4096
)

type NetManager interface {
}

type streamMap map[string]*stream
type listenAddrMap map[string]net.Listener

type perfmetrics struct {
	bytesSent       uint64
	bytesReceived   uint64
}

type netManager struct {
	connManager []*connManager
	perfmetrics *perfmetrics // cumulative performance
}

type connManager struct {
	netManager *netManager
	inboundMux      sync.RWMutex
	inbound         streamMap
	outboundMux     sync.RWMutex
	outbound        streamMap

	ReadbufLen int32
	lisMap     listenAddrMap
	network    string
	addr       string

	tcpAddr *net.TCPAddr
	perfmetrics *perfmetrics
}

func NewNetworkManager() *netManager {
	n := &netManager{}
	return n
}

func (mngr *connManager) GetNetworkManager() NetManager {
	return mngr.netManager
}

func (n *netManager) reuseOrNewConn(endpoint *net.TCPAddr) (*stream, error) {
	for _, connMngr := range n.connManager {
		if cmp.Equal(connMngr.tcpAddr, endpoint) {
			for _, out := range connMngr.outbound {
				if out.CanBeReused() {
					fmt.Printf("Use idle connection\n")
					out.Refresh()
					return out, nil
				}
			}
		}
	}
	// If none were found open a new one
	conn, err := net.DialTCP(endpoint.Network(), nil, endpoint)
	if err != nil { return nil, err }

	connMngr := n.NewConnManager("", "")
	connMngr.tcpAddr = endpoint
	stream := connMngr.NewStreamConn(conn, Outbound)

	return stream,nil
}