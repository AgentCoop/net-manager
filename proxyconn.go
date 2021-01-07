package netmanager

import (
	job "github.com/AgentCoop/go-work"
)

type proxyConn struct {
	upstreamServer *ServerNet
	upstream       *stream
	downstream     *stream
}

func (p *proxyConn) Upstream() *stream {
	return p.upstream
}

func (p *proxyConn) Downstream() *stream {
	return p.downstream
}

func (p *proxyConn) ProxyConnectTask(j job.Job) (job.Init, job.Run, job.Finalize) {
	run := func(task job.Task) {
		netMngr := p.downstream.connManager.netManager
		conn, err := netMngr.reuseOrNewConn(p.upstreamServer.TcpAddr)
		task.Assert(err)
		p.upstream = conn
		p.upstream.dataKind = DataRawKind
		task.Done()
	}
	return nil, run, nil
}

func (p *proxyConn) ProxyReadUpstreamTask(j job.Job) (job.Init, job.Run, job.Finalize) {
	run := func(task job.Task) {
		read(p.upstream, task)
		task.Tick()
	}
	return nil, run, nil
}

func (p *proxyConn) ProxyWriteUpstreamTask(j job.Job) (job.Init, job.Run, job.Finalize) {
	run := func(task job.Task) {
		write(p.upstream, task)
		task.Tick()
	}
	return nil, run, nil
}

func (p *proxyConn) ProxyReadDownstreamTask(j job.Job) (job.Init, job.Run, job.Finalize) {
	init := func(task job.Task) {
		p.downstream.dataKind = DataRawKind
	}
	run := func(task job.Task) {
		read(p.downstream, task)
		task.Tick()
	}
	return init, run, nil
}

func (p *proxyConn) ProxyWriteDownstreamTask(j job.Job) (job.Init, job.Run, job.Finalize) {
	run := func(task job.Task) {
		write(p.downstream, task)
		task.Tick()
	}
	return nil, run, nil
}
