package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"syscall"
)

const (
	MaxClient = 10
	ServerAddr = "127.0.0.1"
	ServerPort = 5000
)

type Poll struct {
	fd 		int
	changes []syscall.Kevent_t
}

type conn struct {
	fd 			int
	listenAddr	syscall.SockaddrInet4
	clientAddr	syscall.SockaddrInet4
	out 		[]byte
}

type server struct {
	p 		*Poll
	cm  	map[int]*conn
	packet 	[]byte
}

func main() {
	//initialization server address
	var serverAddr syscall.SockaddrInet4
	serverAddr.Port = ServerPort
	copy(serverAddr.Addr[:], net.ParseIP(ServerAddr).To4())
	srv := &server{
		p:      newPoll(),
		cm:     make(map[int]*conn),
		packet: make([]byte, 1024),
	}
	serverFd, err := createListener(&serverAddr)
	if err != nil {
		panic(err)
	}
	srv.p.addRead(serverFd)		//监听端口放入changes等待
	//核心，reactor模式进行事件分发
	srv.p.wait(func(fd int) error {
		c := srv.cm[fd]
		switch {
		case c == nil:
			return srv.newConnection(serverAddr, serverFd)
		case len(c.out) > 0:
			return loopWrite(srv, c)
		default:
			return loopRead(srv, c)
		}
	})
}

func loopRead(s *server, c *conn) error {
	nread, err := syscall.Read(c.fd, s.packet)
	if err != nil {
		return err
	}
	if nread == 0 {
		log.Printf("close connectrion from: %s:%d\n", strings.Replace(strings.Trim(fmt.Sprint(c.clientAddr.Addr), "[]"), " ", ".", -1), c.clientAddr.Port)
		syscall.Close(c.fd)
	}
	c.out = s.packet[:nread]
	if len(c.out) != 0 {
		s.p.addWrite(c.fd)
	}
	return nil
}

func loopWrite(s *server, c *conn) error {
	nwrite, err := syscall.Write(c.fd, c.out)
	if err != nil {
		return err
	}
	if nwrite == len(c.out) {
		c.out = c.out[:0]
	}
	if len(c.out) == 0 {
		s.p.delWrite(c.fd)
	}
	return nil
}

func (s *server)newConnection(serverAddr syscall.SockaddrInet4, fd int) error {
	clientFd, sockAddr, err := syscall.Accept(fd)
	if err != nil {
		return err
	}
	if err = syscall.SetNonblock(clientFd, true); err != nil {
		return err
	}

	clientAddr, err := SockaddrToAddr(sockAddr)
	if err != nil {
		return err
	}
	c := &conn{
		fd:         clientFd,
		listenAddr: serverAddr,
		clientAddr: clientAddr,
		out:        nil,
	}
	s.cm[clientFd] = c
	log.Printf("accept connection from %s:%d\n", strings.Replace(strings.Trim(fmt.Sprint(c.clientAddr.Addr), "[]"), " ", ".", -1), clientAddr.Port)
	s.p.addRead(clientFd)
	return nil
}

func createListener(serverAddr *syscall.SockaddrInet4) (fd int, err error) {
	serverFd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
	if err != nil {
		return serverFd, err
	}
	err = syscall.Bind(serverFd, serverAddr)
	if err != nil {
		return serverFd, err
	}
	err = syscall.Listen(serverFd, MaxClient)
	if err != nil {
		return serverFd, err
	}
	log.Printf("initially create socket fd listening: %s:%d \n", strings.Replace(strings.Trim(fmt.Sprint(serverAddr.Addr), "[]"), " ", ".", -1), serverAddr.Port)
	return serverFd, nil
}


func SockaddrToAddr(sa syscall.Sockaddr) (syscall.SockaddrInet4, error) {
	var addr syscall.SockaddrInet4
	switch sa := sa.(type) {
	case *syscall.SockaddrInet4:
		addr.Addr = sa.Addr
		addr.Port = sa.Port
		return addr, nil
	default:
		return addr, errors.New("currently unsupported socket address")
	}
}

func newPoll() *Poll {
	p := new(Poll)
	kq, err := syscall.Kqueue()
	if err != nil {
		panic(err)
	}
	p.fd = kq
	return p
}

func (p *Poll) addRead(fd int) {
	p.changes = append(p.changes,
		syscall.Kevent_t{Ident:uint64(fd), Filter:syscall.EVFILT_READ, Flags:syscall.EV_ADD})
}

func (p *Poll) addWrite(fd int) {
	p.changes = append(p.changes,
		syscall.Kevent_t{Ident:uint64(fd), Filter:syscall.EVFILT_WRITE, Flags:syscall.EV_ADD})
}

func (p *Poll) delWrite(fd int) {
	p.changes = append(p.changes,
		syscall.Kevent_t{Ident:uint64(fd), Filter:syscall.EVFILT_WRITE, Flags:syscall.EV_DELETE})
}

func (p *Poll) wait(iter func(fd int) error) error{
	events := make([]syscall.Kevent_t, 128)
	for{
		nev, err := syscall.Kevent(p.fd, p.changes, events, nil)
		if err != nil {
			return err
		}
		p.changes = p.changes[:0]
		for i:=0; i<nev; i++ {
			if err := iter(int(events[i].Ident)); err != nil {
				return err
			}
		}
	}
}