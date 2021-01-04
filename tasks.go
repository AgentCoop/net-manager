package netmanager

import (
	job "github.com/AgentCoop/go-work"
	"net"
	"sync/atomic"
)

func (mngr *connManager) ConnectTask(j job.JobInterface) (job.Init, job.Run, job.Cancel) {
	run := func(task *job.TaskInfo) {
		conn, err := net.Dial(mngr.network, mngr.addr)
		task.Assert(err)

		stream := mngr.NewStreamConn(conn, Outbound)
		j.SetValue(stream)
		mngr.addConn(stream)

		stream.onNewConnChan <- job.DoneSig
		task.Done()
	}
	return nil, run, nil
}

func (mngr *connManager) AcceptTask(j job.JobInterface) (job.Init, job.Run, job.Cancel) {
	run := func(task *job.TaskInfo) {
		var lis net.Listener
		key := mngr.network + mngr.addr
		if _, ok := mngr.lisMap[key]; ! ok {
			l, err := net.Listen(mngr.network, mngr.addr)
			mngr.lisMap[key] = l
			task.Assert(err)
			lis = l
		}
		lis = mngr.lisMap[key]

		conn, acceptErr := lis.Accept()
		task.Assert(acceptErr)

		ac := mngr.NewStreamConn(conn, Inbound)
		j.SetValue(ac)
		mngr.addConn(ac)

		ac.onNewConnChan <- job.DoneSig
		task.Done()
	}
	return nil, run, nil
}

func (mngr *connManager) ReadTask(j job.JobInterface) (job.Init, job.Run, job.Cancel) {
	run := func(task *job.TaskInfo) {
		stream := j.GetValue().(*streamConn)
		n, err := stream.conn.Read(stream.readbuf)
		task.Assert(err)

		atomic.AddUint64(&stream.connManager.perfmetrics.bytesReceived, uint64(n))
		data := stream.readbuf[0:n]

		switch stream.DataKind {
		case DataFrameKind:
			stream.df.Capture(data)
			if stream.df.IsFullFrame() {
				stream.recvDataFrameChan <- stream.df
				<-stream.recvDataFrameSyncChan
			}
		case DataRawKind:
			stream.onRawChan <- data
			<-stream.onRawDoneChan
		}

		task.Tick()
	}
	cancel := func() {
		stream := j.GetValue().(*streamConn)
		mngr := stream.connManager
		close(stream.readChan)
		close(stream.recvDataFrameChan)
		close(stream.recvDataFrameSyncChan)
		mngr.delConn(stream)
		stream.conn.Close()
	}
	return nil, run, cancel
}

func (mngr *connManager) WriteTask(j job.JobInterface) (job.Init, job.Run, job.Cancel) {
	run := func(task *job.TaskInfo) {
		stream := j.GetValue().(*streamConn)
		var n int
		var err error
		select {
		case data := <- stream.writeChan:
			switch data.(type) {
			case []byte: // raw data
				n, err = stream.conn.Write(data.([]byte))
				task.Assert(err)
			case nil:
				// Handle error
			default:
				enc, err := stream.df.ToFrame(data)
				task.Assert(err)
				n, err = stream.conn.Write(enc)
				task.Assert(err)
			}
			// Sync with the writer
			stream.writeSyncChan <- n
			atomic.AddUint64(&stream.connManager.perfmetrics.bytesSent, uint64(n))
		default:
		}
		task.Tick()
	}
	cancel := func()  {
		stream := j.GetValue().(*streamConn)
		close(stream.writeChan)
		close(stream.writeSyncChan)
		stream.conn.Close()
	}
	return nil, run, cancel
}
