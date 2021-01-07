package netmanager

import (
	job "github.com/AgentCoop/go-work"
	netdataframe "github.com/AgentCoop/net-dataframe"
)

type ConnManager interface {
	ConnectTask(job.JobInterface) (job.Init, job.Run, job.Finalize)
	AcceptTask(job.JobInterface) (job.Init, job.Run, job.Finalize)
	GetNetworkManager() NetManager
}

type Stream interface {
	Read() <-chan interface{}
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
	ProxyReadUpstreamTask(j job.JobInterface) (job.Init, job.Run, job.Finalize)
	ProxyReadDownstreamTask(j job.JobInterface) (job.Init, job.Run, job.Finalize)
	ProxyWriteUpstreamTask(j job.JobInterface) (job.Init, job.Run, job.Finalize)
	ProxyWriteDownstreamTask(j job.JobInterface) (job.Init, job.Run, job.Finalize)
}

func NewProxyConn(upstreamServer *ServerNet, downstream Stream) *proxyConn {
	p := &proxyConn{upstreamServer: upstreamServer, downstream: downstream.(*stream)}
	return p
}

type StreamOptions struct {
	DataKind DataKind
}

func DefaultStreamOptions() *StreamOptions {
	return &StreamOptions{
		DataKind: DataFrameKind,
	}
}
