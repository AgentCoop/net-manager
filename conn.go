package netmanager

import (
	job "github.com/AgentCoop/go-work"
	"net"
	"sync"
	netdataframe "github.com/AgentCoop/net-dataframe"
)

type ConnType int
type ConnState int
type DataKind int

const (
	Inbound ConnType = iota
	Outbound
)

const (
	Active ConnState = iota
	Closed
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

type streamConn struct {
	conn net.Conn

	writeChan chan interface{}
	writeDoneChan chan int

	readChan chan interface{}
	onNewConnChan chan struct{}
	onConnCloseChan chan struct{}

	onDataFrameChan chan []byte
	onDataFrameDoneChan chan struct{}

	onRawChan chan []byte
	onRawDoneChan chan struct{}

	connManager *connManager
	state ConnState
	typ ConnType
	DataKind DataKind
	df netdataframe.DataFrame
	readbuf []byte

	value   interface{}
	ValueMu sync.RWMutex
}

func (mngr *connManager) NewStreamConn(conn net.Conn, typ ConnType) *streamConn {
	//streamConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	stream := &streamConn{conn: conn, typ: typ}

	stream.writeChan = make(chan interface{})
	stream.writeDoneChan = make(chan int)
	stream.readChan = make(chan interface{})

	stream.onNewConnChan = make(chan struct{}, 1)
	stream.onConnCloseChan = make(chan struct{}, 1)

	stream.onDataFrameChan = make(chan []byte)
	stream.onDataFrameDoneChan = make(chan struct{})

	stream.onRawChan = make(chan []byte)
	stream.onRawDoneChan = make(chan struct{})

	stream.connManager = mngr
	stream.df = netdataframe.NewDataFrame()
	stream.readbuf = make([]byte, mngr.ReadbufLen)
	return stream
}

func (c *streamConn) String() string {
	return c.conn.RemoteAddr().String() + " -> " + c.conn.LocalAddr().String()
}

func (c *streamConn) Key() string {
	return c.String()
}