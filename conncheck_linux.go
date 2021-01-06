
// +build linux darwin dragonfly freebsd netbsd openbsd solaris illumos
// +build go1.14

package netmanager

import (
	"errors"
	"net"
	"reflect"
	"syscall"
)

const (
	FieldFd = "fd"
	FieldPfd = "pfd"
	FieldSysfd = "Sysfd"
)

var (
	errNetFdNotAvailable = errors.New("POSIX netFd struct is not available")
	errPollFdNotAvailable = errors.New("POSIX poll.FD struct is not available")
)

// A dirty way to fall through the abstraction layer to get the precious socket file descriptor
func getSocketFd(l net.Conn) int {
	v := reflect.ValueOf(l)
	netFD := reflect.Indirect(reflect.Indirect(v).FieldByName(FieldFd))
	if ! netFD.CanAddr() { panic(errNetFdNotAvailable) }

	pfd := reflect.Indirect(netFD.FieldByName(FieldPfd))
	if ! pfd.CanAddr() { panic(errPollFdNotAvailable) }

	SysFd := reflect.Indirect(pfd.FieldByName(FieldSysfd))
	if ! SysFd.CanAddr() { return -1 }

	fd := int(SysFd.Int())
	return fd
}

func connCheck(conn net.Conn) bool {
	if conn == nil { return false }
	var err error
	var n int
	var buf [1]byte

	socketFd := getSocketFd(conn)
	if socketFd == - 1 { return false }
	n, _, err = syscall.Recvfrom(socketFd, buf[:], syscall.MSG_PEEK)

	switch {
	case n >= 0 && err == nil:
		return true
	// https://man7.org/linux/man-pages/man2/recv.2.html
	// EAGAIN or EWOULDBLOCK
	// The socket is marked nonblocking and the receive operation
	// would block, or a receive timeout had been set and the
	// timeout expired before data was received.  POSIX.1 allows
	// either error to be returned for this case, and does not
	// require these constants to have the same value, so a
	// portable application should check for both possibilities.
	case err == syscall.EAGAIN || err == syscall.EWOULDBLOCK:
		return true
	default:
		return false
	}
}
