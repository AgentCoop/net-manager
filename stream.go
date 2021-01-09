package netmanager

import (
	netdataframe "github.com/AgentCoop/net-dataframe"
	"net"
	"sync"
)

type ConnType int
type StreamState int
type DataKind int

const (
	Inbound ConnType = iota
	Outbound
)

const (
	InUseConn StreamState = iota
	IdleConn
)

const (
	DataFrameKind DataKind = iota
	DataRawKind
)

func (s ConnType) String() string {
	return [...]string{"Inbound", "Outbound"}[s]
}

func (s StreamState) String() string {
	return [...]string{"InUse", "Idle"}[s]
}

type stream struct {
	conn net.Conn

	writeChan             chan interface{}
	writeSyncChan         chan int
	chanclose             sync.Once
	readChan              chan interface{}
	newConnChan           chan struct{}
	connCloseChan         chan struct{}
	recvDataFrameChan     chan netdataframe.DataFrame
	recvDataFrameSyncChan chan struct{}
	recvRawChan           chan []byte
	recvRawSyncChan       chan struct{}

	connManager *connManager
	statemux    sync.RWMutex
	state       StreamState
	typ         ConnType
	dataKind    DataKind
	frame       netdataframe.DataFrame
	readbuf     []byte
	connclose   sync.Once
}

func (s *stream) Read() <-chan interface{} {
	return s.readChan
}

func (s *stream) Write() chan<- interface{} {
	return s.writeChan
}

func (s *stream) WriteSync() int {
	return <-s.writeSyncChan
}

func (s *stream) RecvDataFrame() <-chan netdataframe.DataFrame {
	return s.recvDataFrameChan
}

func (s *stream) RecvDataFrameSync() {
	s.recvDataFrameSyncChan <- struct{}{}
}

func (s *stream) RecvRaw() <-chan []byte {
	return s.recvRawChan
}

func (s *stream) RecvRawSync() {
	s.recvRawSyncChan <- struct{}{}
}

func (s *stream) NewConn() <-chan struct{} {
	return s.newConnChan
}

func (s *stream) GetConnManager() ConnManager {
	return s.connManager
}

func (s *stream) GetConn() net.Conn {
	return s.conn
}

func (s *stream) IsConnected() bool {
	return connCheck(s.conn)
}

func (s *stream) CloseWithReuse() {
	s.closeChans()
	s.SetState(IdleConn)
}

func (s *stream) CanBeReused() bool {
	return s.GetState() == IdleConn && s.IsConnected()
}

func (s *stream) Refresh() {
	s.SetState(InUseConn)
	s.initChans()
}

func (s *stream) initChans() {
	s.writeChan = make(chan interface{})
	s.writeSyncChan = make(chan int)
	s.readChan = make(chan interface{})
	s.newConnChan = make(chan struct{}, 1)
	s.recvDataFrameChan = make(chan netdataframe.DataFrame)
	s.recvDataFrameSyncChan = make(chan struct{})
	s.recvRawChan = make(chan []byte)
	s.recvRawSyncChan = make(chan struct{})
}

func (s *stream) closeChanFn() {
	close(s.writeChan)
	close(s.writeSyncChan)
	close(s.readChan)
	close(s.newConnChan)
	close(s.recvDataFrameChan)
	close(s.recvDataFrameSyncChan)
	close(s.recvRawChan)
	close(s.recvRawSyncChan)
}

func (s *stream) closeChans() {
	s.chanclose.Do(s.closeChanFn)
}

func (c *stream) String() string {
	return c.conn.RemoteAddr().String() + " -> " + c.conn.LocalAddr().String()
}

func (c *stream) Key() string {
	return c.String()
}

func (s *stream) Close() {
	s.connclose.Do(func() {
		s.connManager.delConn(s)
		s.conn.Close()
	})
}

func (s *stream) GetState() StreamState {
	s.statemux.RLock()
	defer s.statemux.RUnlock()
	return s.state
}

func (s *stream) SetState(state StreamState) {
	s.statemux.Lock()
	defer s.statemux.Unlock()
	s.state = state
}
