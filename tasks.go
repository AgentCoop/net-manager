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

func readCancel(stream *stream, task *job.TaskInfo) {
	fmt.Printf("close read\n")
	mngr := stream.connManager
	close(stream.readChan)
	close(stream.recvDataFrameChan)
	close(stream.recvDataFrameSyncChan)
	mngr.delConn(stream)
}

func read(stream *stream, task *job.TaskInfo) {
	n, err := stream.conn.Read(stream.readbuf)
	task.Assert(err)

	atomic.AddUint64(&stream.connManager.perfmetrics.bytesReceived, uint64(n))
	data := stream.readbuf[0:n]

	switch stream.dataKind {
	case DataFrameKind:
		stream.frame.Capture(data)
		if stream.frame.IsFullFrame() {
			stream.recvDataFrameChan <- stream.frame
			<-stream.recvDataFrameSyncChan
		}
	case DataRawKind:
		stream.recvRawChan <- data
		<-stream.recvRawSyncChan
	}
}

func (stream *stream) ReadOnStreamTask(j job.JobInterface) (job.Init, job.Run, job.Finalize) {
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
		s := j.GetValue().(*stream)
		read(s, task)
		task.Tick()
	}
	fin := func(task *job.TaskInfo) {
		s := j.GetValue()
		if s == nil { return }
		readCancel(s.(*stream), task)
	}
	return nil, run, fin
}

func write(s *stream, task *job.TaskInfo) {
	var n int
	var err error

	select {
	case data := <- s.writeChan:
		switch data.(type) {
		case []byte: // raw data
			n, err = s.conn.Write(data.([]byte))
			task.Assert(err)
		case nil:
			// Handle error
		default:
			enc, err := s.frame.ToFrame(data)
			task.Assert(err)
			n, err = s.conn.Write(enc)
			task.Assert(err)
		}
		// Sync with the writer
		s.writeSyncChan <- n
		atomic.AddUint64(&s.connManager.perfmetrics.bytesSent, uint64(n))
	}
}

func writeCancel(stream *stream, task *job.TaskInfo) {
	fmt.Printf("close write\n")
	close(stream.writeChan)
	close(stream.writeSyncChan)
}

func (s *stream) WriteOnStreamTask(j job.JobInterface) (job.Init, job.Run, job.Finalize) {
	run := func(task *job.TaskInfo) {
		write(s, task)
		task.Tick()
	}
	fin := func(task *job.TaskInfo) {
		writeCancel(s, task)
	}
	return nil, run, fin
}

func WriteTask(j job.JobInterface) (job.Init, job.Run, job.Finalize) {
	run := func(task *job.TaskInfo) {
		s := j.GetValue().(*stream)
		write(s, task)
		task.Tick()
	}
	fin := func(task *job.TaskInfo)  {
		s := j.GetValue()
		if s == nil { return }
		writeCancel(s.(*stream), task)
	}
	return nil, run, fin
}
