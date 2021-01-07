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

func (p *proxyConn) ProxyConnectTask(j job.JobInterface) (job.Init, job.Run, job.Finalize) {
	run := func(task *job.TaskInfo) {
		netMngr := p.downstream.connManager.netManager
		conn, err := netMngr.reuseOrNewConn(p.upstreamServer.TcpAddr)
		task.Assert(err)
		p.upstream = conn
		p.upstream.dataKind = DataRawKind
		task.Done()
	}
	return nil, run, nil
}

func (p *proxyConn) ProxyReadUpstreamTask(j job.JobInterface) (job.Init, job.Run, job.Finalize) {
	run := func(task *job.TaskInfo) {
		read(p.upstream, task)
		task.Tick()
	}
	return nil, run, nil
}

func (p *proxyConn) ProxyWriteUpstreamTask(j job.JobInterface) (job.Init, job.Run, job.Finalize) {
	run := func(task *job.TaskInfo) {
		write(p.upstream, task)
		task.Tick()
	}
	return nil, run, nil
}

func (p *proxyConn) ProxyReadDownstreamTask(j job.JobInterface) (job.Init, job.Run, job.Finalize) {
	init := func(task *job.TaskInfo) {
		p.downstream.dataKind = DataRawKind
	}
	run := func(task *job.TaskInfo) {
		read(p.downstream, task)
		task.Tick()
	}
	return init, run, nil
}

func (p *proxyConn) ProxyWriteDownstreamTask(j job.JobInterface) (job.Init, job.Run, job.Finalize) {
	run := func(task *job.TaskInfo) {
		write(p.downstream, task)
		task.Tick()
	}
	return nil, run, nil
}
