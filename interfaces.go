package netmanager

import (
	job "github.com/AgentCoop/go-work"
	netdataframe "github.com/AgentCoop/net-dataframe"
)

type ConnManager interface {
	ConnectTask(job.Job) (job.Init, job.Run, job.Finalize)
	AcceptTask(job.Job) (job.Init, job.Run, job.Finalize)
	GetNetworkManager() NetManager
	GetConnTotalCount(state StreamState, typ ConnType) int
	GetInboundLimit() int
}

type Stream interface {
	//Read() <-chan interface{}
	Write() chan<- interface{}
	WriteSync() int
	RecvDataFrame() <-chan netdataframe.DataFrame
	RecvDataFrameSync()
	RecvRaw() <-chan []byte
	RecvRawSync()
	NewConn() <-chan struct{}

	GetConnManager() ConnManager
	Key() string
	String() string
	GetState() StreamState
	CanBeReused() bool
	IsConnected() bool
	Refresh()
	CloseWithReuse()
	Close()
}

type ProxyConn interface {
	Upstream() *stream
	Downstream() *stream
	ProxyReadUpstreamTask(j job.Job) (job.Init, job.Run, job.Finalize)
	ProxyReadDownstreamTask(j job.Job) (job.Init, job.Run, job.Finalize)
	ProxyWriteUpstreamTask(j job.Job) (job.Init, job.Run, job.Finalize)
	ProxyWriteDownstreamTask(j job.Job) (job.Init, job.Run, job.Finalize)
}

func NewProxyConn(upstreamServer *ServerNet, downstream Stream) *proxyConn {
	p := &proxyConn{upstreamServer: upstreamServer, downstream: downstream.(*stream)}
	return p
}

type ConnManagerOptions struct {
	InboundLimit	int
	ReadbufLen		int
}

type StreamOptions struct {
	DataKind DataKind
}

func DefaultStreamOptions() *StreamOptions {
	return &StreamOptions{
		DataKind: DataFrameKind,
	}
}
