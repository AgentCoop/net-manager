package netmanager

import (
	netdataframe "github.com/AgentCoop/net-dataframe"
	"net"
	"sync"
)

type ConnType int
type ConnState int
type DataKind int

//type StreamConn interface {
//	Write() chan<- interface{}
//	WriteSync() int
//	NewConn() <-chan struct{}
//	RecvDataFrame() <-chan netdataframe.DataFrame
//	RecvDataFrameSync()
//	RecvRaw() <-chan []byte
//	RecvRawSync()
//	GetConnManager() ConnManager
//	GetDataKind() DataKind
//	SetDataKind(DataKind)
//}

const (
	Inbound ConnType = iota
	Outbound
)

const (
	InuseConn ConnState = iota
	IdleConn
)

const (
	DataFrameKind DataKind = iota
	DataRawKind
)

func (s ConnType) String() string {
	return [...]string{"Inbound", "Outbound"}[s]
}

func (s ConnState) String() string {
	return [...]string{"Active", "Closed"}[s]
}

type StreamConn struct {
	conn net.Conn

	writeChan     chan interface{}
	writeSyncChan chan int

	readChan        chan interface{}
	newConnChan     chan struct{}
	onConnCloseChan chan struct{}

	recvDataFrameChan     chan netdataframe.DataFrame
	recvDataFrameSyncChan chan struct{}
	recvRawChan     chan []byte
	recvRawSyncChan chan struct{}

	connManager *connManager
	State       ConnState
	typ         ConnType
	DataKind    DataKind
	df          netdataframe.DataFrame
	readbuf     []byte

	value   interface{}
	ValueMu sync.RWMutex

	closeOnce sync.Once
}

func (s *StreamConn) Read() <-chan interface{} {
	return s.readChan
}

func (s *StreamConn) Write() chan<- interface{} {
	return s.writeChan
}

func (s *StreamConn) WriteSync() int {
	return <-s.writeSyncChan
}

func (s *StreamConn) RecvDataFrame() <-chan netdataframe.DataFrame {
	return s.recvDataFrameChan
}

func (s *StreamConn) RecvDataFrameSync() {
	s.recvDataFrameSyncChan <- struct{}{}
}

func (s *StreamConn) RecvRaw() <-chan []byte {
	return s.recvRawChan
}

func (s *StreamConn) RecvRawSync() {
	s.recvRawSyncChan <- struct{}{}
}

func (s *StreamConn) NewConn() <-chan struct{} {
	return s.newConnChan
}

func (s *StreamConn) GetConnManager() ConnManager {
	return s.connManager
}

func (s *StreamConn) GetDataKind() DataKind {
	return s.DataKind
}

func (s *StreamConn) SetDataKind(kind DataKind) {
	s.DataKind = kind
}

func (s *StreamConn) GetConn() net.Conn {
	return s.conn
}

func (s *StreamConn) IsConnected() bool {
	return connCheck(s.conn)
}

func (mngr *connManager) NewStreamConn(conn net.Conn, typ ConnType) *StreamConn {
	stream := &StreamConn{conn: conn, typ: typ, State: InuseConn}

	stream.writeChan = make(chan interface{})
	stream.writeSyncChan = make(chan int)
	stream.readChan = make(chan interface{})

	stream.newConnChan = make(chan struct{}, 1)
	stream.onConnCloseChan = make(chan struct{}, 1)

	stream.recvDataFrameChan = make(chan netdataframe.DataFrame)
	stream.recvDataFrameSyncChan = make(chan struct{})

	stream.recvRawChan = make(chan []byte)
	stream.recvRawSyncChan = make(chan struct{})

	stream.connManager = mngr
	stream.df = netdataframe.NewDataFrame()
	stream.readbuf = make([]byte, mngr.ReadbufLen)

	mngr.addConn(stream)
	return stream
}

func (c *StreamConn) String() string {
	return c.conn.RemoteAddr().String() + " -> " + c.conn.LocalAddr().String()
}

func (c *StreamConn) Key() string {
	return c.String()
}

func (s *StreamConn) Close() {
	s.closeOnce.Do(func() {
		s.conn.Close()
	})
}
