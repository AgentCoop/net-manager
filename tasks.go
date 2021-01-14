package netmanager

import (
	job "github.com/AgentCoop/go-work"
	netdataframe "github.com/AgentCoop/net-dataframe"
	"net"
	"sync/atomic"
)

func readFin(stream *stream, task job.Task) {
	mngr := stream.connManager
	close(stream.readChan)
	close(stream.recvDataFrameChan)
	close(stream.recvDataFrameSyncChan)
	close(stream.recvRawChan)
	close(stream.recvRawSyncChan)
	mngr.delConn(stream)
	stream.Close()
}

func read(stream *stream, task job.Task) {
	n, err := stream.conn.Read(stream.readbuf) // Read network data
	task.Assert(err) // Assert that there is no error. A failed assertion will stop job execution.

	atomic.AddUint64(&stream.connManager.perfmetrics.bytesReceived, uint64(n))
	data := stream.readbuf[0:n]
	// Send read data in raw stream or in data frames using special task channels
	switch stream.dataKind {
	case DataFrameKind:
		frames, err := stream.framerecv.Capture(data)
		task.Assert(err)
		for _, frame := range frames {
			// Ping/Pong synchronization. Send a frame to the channel and wait for a notification from another task
			// that it was processed.
			stream.recvDataFrameChan <- frame
			<-stream.recvDataFrameSyncChan
		}
	case DataRawKind:
		stream.recvRawChan <- data
		<-stream.recvRawSyncChan
	}
	// Tick and wait for new data
	task.Tick()
}

func writeFin(stream *stream, task job.Task) {
	close(stream.writeChan)
	close(stream.writeSyncChan)
	stream.Close()
}

func write(s *stream, task job.Task) {
	var n int
	var err error

	select {
	case data := <- s.writeChan: // Some task "asked" us to send network data
		task.AssertNotNil(data)
		switch data.(type) {
		case []byte: // raw data
			n, err = s.conn.Write(data.([]byte))
			task.Assert(err)
		default:
			enc, err := netdataframe.ToFrame(data)
			task.Assert(err)
			n, err = s.conn.Write(enc.GetBytes())
			task.Assert(err)
		}
		// Sync with the writer
		s.writeSyncChan <- n // Tell that task that we are done sending data
		atomic.AddUint64(&s.connManager.perfmetrics.bytesSent, uint64(n))
		task.Tick()
	}
}

// Serves as an oneshot task to establish a TCP connection to the target host
func (mngr *connManager) ConnectTask(j job.Job) (job.Init, job.Run, job.Finalize) {
	run := func(task job.Task) {
		conn, err := net.Dial(mngr.network, mngr.addr)
		task.Assert(err)

		stream := mngr.NewStreamConn(conn, Outbound)
		j.SetValue(stream)
		mngr.addConn(stream)

		// Notify other tasks that connection was successfully established
		stream.newConnChan <- job.NotifySig
		// Finish task execution and kick off execution of other tasks.
		task.Done()
	}
	return nil, run, nil
}

func (mngr *connManager) AcceptTask(j job.Job) (job.Init, job.Run, job.Finalize) {
	run := func(task job.Task) {
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

		ac.newConnChan <- job.NotifySig
		task.Done()
	}
	return nil, run, nil
}

func (stream *stream) ReadOnStreamTask(j job.Job) (job.Init, job.Run, job.Finalize) {
	init := func(task job.Task){
	}
	run := func(task job.Task) {
		read(stream, task)
	}
	fin := func(task job.Task) {
		readFin(stream, task)
	}
	return init, run, fin
}

func ReadTask(j job.Job) (job.Init, job.Run, job.Finalize) {
	run := func(task job.Task) {
		s := j.GetValue().(*stream)
		read(s, task)
	}
	fin := func(task job.Task) {
		s := j.GetValue()
		if s == nil { return }
		readFin(s.(*stream), task)
	}
	return nil, run, fin
}

func (s *stream) WriteOnStreamTask(j job.Job) (job.Init, job.Run, job.Finalize) {
	run := func(task job.Task) {
		write(s, task)
	}
	fin := func(task job.Task) {
		writeFin(s, task)
	}
	return nil, run, fin
}

func WriteTask(j job.Job) (job.Init, job.Run, job.Finalize) {
	run := func(task job.Task) {
		s := j.GetValue().(*stream)
		write(s, task)
	}
	fin := func(task job.Task)  {
		s := j.GetValue()
		if s == nil { return }
		writeFin(s.(*stream), task)
	}
	return nil, run, fin
}
