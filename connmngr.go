package netmanager

import (
	netdataframe "github.com/AgentCoop/net-dataframe"
	"net"
)

func (n *netManager) NewConnManager(network string, address string, opts *ConnManagerOptions) *connManager {
	mngr := &connManager{network: network, addr: address}
	mngr.netManager = n
	mngr.perfmetrics = &perfmetrics{}
	n.connManager = append(n.connManager, mngr)
	mngr.lisMap = make(listenAddrMap)
	mngr.streammap = make(streamMap)
	if opts == nil {
		opts = &ConnManagerOptions{
			InboundLimit: -1,
			ReadbufLen: DefaultReadBufLen,
		}
	}
	mngr.readbufLen = opts.ReadbufLen
	mngr.inboundLimit = opts.InboundLimit
	return mngr
}

func (mngr *connManager) NewStreamConn(conn net.Conn, typ ConnType) *stream {
	stream := &stream{conn: conn, typ: typ, state: InUseConn}
	stream.initChans()
	stream.connManager = mngr
	stream.frame = netdataframe.NewDataFrame()
	stream.readbuf = make([]byte, mngr.readbufLen)
	mngr.addConn(stream)
	return stream
}

func (mngr *connManager) GetConnTotalCount(state StreamState, typ ConnType) int {
	mngr.streammapmux.RLock()
	defer mngr.streammapmux.RUnlock()
	var totalcnt int
	for _, stream := range mngr.streammap {
		if stream.typ == typ && stream.GetState() == state {
			totalcnt++
		}
	}
	return totalcnt
}

func (mngr *connManager) addConn(s *stream) {
	mngr.streammapmux.Lock()
	defer mngr.streammapmux.Unlock()
	mngr.streammap[s.Key()] = s
}

func (mngr *connManager) delConn(s *stream) {
	mngr.streammapmux.Lock()
	defer mngr.streammapmux.Unlock()
	delete(mngr.streammap, s.Key())
	s.conn.Close()
}


func (mngr *connManager) GetInboundLimit() int {
	return mngr.inboundLimit
}