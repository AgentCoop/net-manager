package netmanager

import (
	"github.com/AgentCoop/go-work"
	"sync"
	"net"
)

type ConnManager interface {
	ConnectTask(job.JobInterface) (job.Init, job.Run, job.Cancel)
	AcceptTask(job.JobInterface) (job.Init, job.Run, job.Cancel)
	ReadTask(job.JobInterface) (job.Init, job.Run, job.Cancel)
	WriteTask(job.JobInterface) (job.Init, job.Run, job.Cancel)
}

type NetManager interface {

}

type streamMap map[string]*streamConn
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
	inboundMux      sync.RWMutex
	inbound         streamMap
	outboundMux     sync.RWMutex
	outbound        streamMap

	ReadbufLen int32
	lisMap     listenAddrMap
	network    string
	addr       string

	perfmetrics *perfmetrics
}

func NewNetworkManager() *netManager {
	n := &netManager{}
	n.connManager = make([]*connManager, 1)
	return n
}

func (n *netManager) NewConnManager(network string, address string) *connManager {
	mngr := &connManager{network: network, addr: address}
	mngr.ReadbufLen = 4096
	mngr.perfmetrics = &perfmetrics{}
	n.connManager = append(n.connManager, mngr)
	mngr.lisMap = make(listenAddrMap)
	mngr.inbound = make(streamMap)
	mngr.outbound = make(streamMap)
	return mngr
}

func (mngr *connManager) addConn(c *streamConn) {
	var l *sync.RWMutex
	var connMap streamMap
	switch c.typ {
	case Inbound:
		l = &mngr.inboundMux
		connMap = mngr.inbound
	case Outbound:
		l = &mngr.outboundMux
		connMap = mngr.outbound
	}
	l.Lock()
	defer l.Unlock()
	connMap[c.Key()] = c
}

func (mngr *connManager) delConn(c *streamConn) {
	var l *sync.RWMutex
	var connMap streamMap
	switch c.typ {
	case Inbound:
		l = &mngr.inboundMux
		connMap = mngr.inbound
	case Outbound:
		l = &mngr.outboundMux
		connMap = mngr.outbound
	}
	l.Lock()
	defer l.Unlock()
	delete(connMap, c.Key())
	c.conn.Close()
}
