package netmanager

import (
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
	inboundLimit	int // limit number of inbound connections
	streammapmux    sync.RWMutex
	streammap       streamMap

	readbufLen int
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
			for _, stream := range connMngr.streammap {
				if stream.typ == Outbound && stream.CanBeReused() {
					stream.Refresh()
					return stream, nil
				} else {
					connMngr.delConn(stream)
				}
			}
		}
	}
	// If none were found open a new one
	conn, err := net.DialTCP(endpoint.Network(), nil, endpoint)
	if err != nil { return nil, err }

	connMngr := n.NewConnManager("", "", nil)
	connMngr.tcpAddr = endpoint
	stream := connMngr.NewStreamConn(conn, Outbound)

	return stream,nil
}