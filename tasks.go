package netmanager

import (
	"fmt"
	job "github.com/AgentCoop/go-work"
	"net"
	"sync/atomic"
)

func (mngr *connManager) ConnectTask(j job.JobInterface) (job.Init, job.Run, job.Finalize) {
	run := func(task *job.TaskInfo) {
		conn, err := net.Dial(mngr.network, mngr.addr)
		task.Assert(err)

		stream := mngr.NewStreamConn(conn, Outbound)
		j.SetValue(stream)
		mngr.addConn(stream)

		stream.newConnChan <- job.DoneSig
		task.Done()
	}
	return nil, run, nil
}

func (mngr *connManager) AcceptTask(j job.JobInterface) (job.Init, job.Run, job.Finalize) {
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

		ac.newConnChan <- job.DoneSig
		task.Done()
	}
	return nil, run, nil
}

func readCancel(stream *StreamConn, task *job.TaskInfo) {
	fmt.Printf("close read\n")
	mngr := stream.connManager
	close(stream.readChan)
	close(stream.recvDataFrameChan)
	close(stream.recvDataFrameSyncChan)
	mngr.delConn(stream)
}

func read(stream *StreamConn, task *job.TaskInfo) {
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
		stream.recvRawChan <- data
		<-stream.recvRawSyncChan
	}
}

func (stream *StreamConn) ReadOnStreamTask(j job.JobInterface) (job.Init, job.Run, job.Finalize) {
	init := func(task *job.TaskInfo){
	}
	run := func(task *job.TaskInfo) {
		read(stream, task)
		task.Tick()
	}
	fin := func(task *job.TaskInfo) {
		readCancel(stream, task)
	}
	return init, run, fin
}

func ReadTask(j job.JobInterface) (job.Init, job.Run, job.Finalize) {
	run := func(task *job.TaskInfo) {
		stream := j.GetValue().(*StreamConn)
		read(stream, task)
		task.Tick()
	}
	fin := func(task *job.TaskInfo) {
		stream := j.GetValue()
		if stream == nil {
			return
		}
		readCancel(stream.(*StreamConn), task)
	}
	return nil, run, fin
}

func write(stream *StreamConn, task *job.TaskInfo) {
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
	}
}

func writeCancel(stream *StreamConn, task *job.TaskInfo) {
	fmt.Printf("close write\n")
	close(stream.writeChan)
	close(stream.writeSyncChan)
}

func (stream *StreamConn) WriteOnStreamTask(j job.JobInterface) (job.Init, job.Run, job.Finalize) {
	run := func(task *job.TaskInfo) {
		write(stream, task)
		task.Tick()
	}
	fin := func(task *job.TaskInfo) {
		writeCancel(stream, task)
	}
	return nil, run, fin
}

func WriteTask(j job.JobInterface) (job.Init, job.Run, job.Finalize) {
	run := func(task *job.TaskInfo) {
		stream := j.GetValue().(*StreamConn)
		write(stream, task)
		task.Tick()
	}
	fin := func(task *job.TaskInfo)  {
		stream := j.GetValue().(*StreamConn)
		writeCancel(stream, task)
	}
	return nil, run, fin
}
