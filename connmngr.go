package netmanager

import "sync"

func (n *netManager) NewConnManager(network string, address string) *connManager {
	mngr := &connManager{network: network, addr: address}
	mngr.netManager = n
	mngr.ReadbufLen = DefaultReadBufLen
	mngr.perfmetrics = &perfmetrics{}
	n.connManager = append(n.connManager, mngr)
	mngr.lisMap = make(listenAddrMap)
	mngr.inbound = make(streamMap)
	mngr.outbound = make(streamMap)
	return mngr
}

func (mngr *connManager) addConn(c *StreamConn) {
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

func (mngr *connManager) delConn(c *StreamConn) {
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
