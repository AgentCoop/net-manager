package netmanager

import (
	"net"
)

type Server struct {
	Host string
	Weight uint8
	MaxConns uint16
}

type ServerNet struct {
	Server *Server
	TcpAddr *net.TCPAddr
}

