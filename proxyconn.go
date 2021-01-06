package netmanager

import (
	job "github.com/AgentCoop/go-work"
)

type ProxyConn interface {
	Upstream() *StreamConn
	Downstream() *StreamConn
	ProxyReadUpstreamTask(j job.JobInterface) (job.Init, job.Run, job.Finalize)
	ProxyReadDownstreamTask(j job.JobInterface) (job.Init, job.Run, job.Finalize)
	ProxyWriteUpstreamTask(j job.JobInterface) (job.Init, job.Run, job.Finalize)
	ProxyWriteDownstreamTask(j job.JobInterface) (job.Init, job.Run, job.Finalize)
}

type proxyConn struct {
	upstreamServer *ServerNet
	upstreamConn *StreamConn
	downstreamConn *StreamConn
}

func (n *netManager) NewProxyConn(upstreamServer *ServerNet, downstream *StreamConn) *proxyConn {
	p := &proxyConn{upstreamServer: upstreamServer, downstreamConn: downstream}
	return p
}

func (p *proxyConn) Upstream() *StreamConn {
	return p.upstreamConn
}

func (p *proxyConn) Downstream() *StreamConn {
	return p.downstreamConn
}

func (p *proxyConn) ProxyConnectTask(j job.JobInterface) (job.Init, job.Run, job.Finalize) {
	run := func(task *job.TaskInfo) {
		netMngr := p.downstreamConn.connManager.netManager
		conn, err := netMngr.reuseOrNewConn(p.upstreamServer.TcpAddr)
		task.Assert(err)
		p.upstreamConn = conn
		task.Done()
	}
	return nil, run, nil
}

func (p *proxyConn) ProxyReadUpstreamTask(j job.JobInterface) (job.Init, job.Run, job.Finalize) {
	run := func(task *job.TaskInfo) {
		read(p.upstreamConn, task)
		task.Tick()
	}
	return nil, run, nil
}

func (p *proxyConn) ProxyWriteUpstreamTask(j job.JobInterface) (job.Init, job.Run, job.Finalize) {
	run := func(task *job.TaskInfo) {
		write(p.upstreamConn, task)
		task.Tick()
	}
	return nil, run, nil
}

func (p *proxyConn) ProxyReadDownstreamTask(j job.JobInterface) (job.Init, job.Run, job.Finalize) {
	run := func(task *job.TaskInfo) {
		read(p.downstreamConn, task)
		task.Tick()
	}
	return nil, run, nil
}

func (p *proxyConn) ProxyWriteDownstreamTask(j job.JobInterface) (job.Init, job.Run, job.Finalize) {
	run := func(task *job.TaskInfo) {
		write(p.downstreamConn, task)
		task.Tick()
	}
	return nil, run, nil
}
