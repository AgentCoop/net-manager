package netmanager

import (
	"fmt"
	"github.com/AgentCoop/go-work"
	"github.com/google/go-cmp/cmp"
	"net"
	"sync"
)

const (
	DefaultReadBufLen = 4096
)

// Governs inbound/outbound connections for an TCP endpoint
type ConnManager interface {
	ConnectTask(job.JobInterface) (job.Init, job.Run, job.Finalize)
	AcceptTask(job.JobInterface) (job.Init, job.Run, job.Finalize)
	//ReadTask(job.JobInterface) (job.Init, job.Run, job.Finalize)
	//WriteTask(job.JobInterface) (job.Init, job.Run, job.Finalize)
	GetNetworkManager() NetManager
}

type NetManager interface {
	NewProxyConn(upstreamServer *ServerNet, downstream *StreamConn) *proxyConn
}

type streamMap map[string]*StreamConn
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

func (n *netManager) reuseOrNewConn(endpoint *net.TCPAddr) (*StreamConn, error) {
	for _, connMngr := range n.connManager {
		if cmp.Equal(connMngr.tcpAddr, endpoint) {
			for _, out := range connMngr.outbound {
				if out.Available() {
					fmt.Printf(" !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!! Usse idle conn\n")
					//ConnCheck(out.conn)
					out.Refresh()
					//out.State = InuseConn
					//n, err := out.conn.Write([]byte("hello"))
					//fmt.Printf("done sending to upstream %d err %s", n, err)
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