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
	BufferSize = 1024
	MaxEvents  = 32
	ServerAddr = "127.0.0.1"
	ServerPort = 5000
	MaxClient  = 10
)

func main() {
	var (
		serverFd 				int
		serverAddr				syscall.SockaddrInet4
		changes			 		[]syscall.Kevent_t
		buf						[BufferSize]byte
		err						error
		fdToClient				map[int]syscall.SockaddrInet4
	)
	//server initialization
	serverAddr.Port = ServerPort
	copy(serverAddr.Addr[:], net.ParseIP(ServerAddr).To4())
	serverFd, err = syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
	if err != nil {
		panic(err)
	}
	err = syscall.Bind(serverFd, &serverAddr)
	if err != nil {
		panic(err)
	}
	err = syscall.Listen(serverFd, MaxClient)
	if err != nil {
		panic(err)
	}

	//create kqueue
	kq, err := syscall.Kqueue()
	if err != nil {
		panic(err)
	}

	//add sockets to changes
	changes = append(changes, syscall.Kevent_t{Ident: uint64(serverFd), Filter: syscall.EVFILT_READ, Flags: syscall.EV_ADD})

	defer syscall.Close(serverFd)
	defer syscall.Close(kq)
	events := make([]syscall.Kevent_t, MaxEvents) 		//要创建一个有长度的切片，可以试试events = []syscall.Kevent_t,看看发生什么情况
	fdToClient = make(map[int]syscall.SockaddrInet4)	//创建fd与socket address的映射关系
	for {
		nev, err := syscall.Kevent(kq, changes, events, nil)
		if err != nil && err != syscall.EINTR {
			panic(err)
		}
		for i:=0; i<nev; i++ {
			if int(events[i].Ident) == serverFd {	//如果有客户端连接，将这个连接的文件描述符放入changes数组等待其活跃
				clientFd, sockAddr, err := syscall.Accept(serverFd)
				if err != nil {
					panic(err)
				}
				clientAddr, err := SockaddrToAddr(sockAddr)
				fdToClient[clientFd] = clientAddr
				if err != nil {
					panic(err)
				}
				log.Printf("Accept connection from %s:%d\n", strings.Replace(strings.Trim(fmt.Sprint(clientAddr.Addr), "[]"), " ", ".", -1), clientAddr.Port)
				changes = append(changes, syscall.Kevent_t{Ident: uint64(clientFd), Filter: syscall.EVFILT_READ, Flags: syscall.EV_ADD})
			} else {	//客户端连接的文件描述符可读，接收来自该客户端的数据并返回同样的数据
				eventFd := int(events[i].Ident)
				for {
					nread, err := syscall.Read(eventFd, buf[:])
					if nread > 0 {
						_, err := syscall.Write(eventFd, buf[:nread])
						if err != nil {
							panic(err)
						}
					} else {	//来自客户端的数据长度为0，说明客户端主动断开连接，因此退出这个读写循环
						break
					}
					if err != nil {
						panic(err)
					}
				}
				//客户端关闭连接的后续处理：关闭文件描述符并将该event从changes数组移除
				// note:kquue的官方手册里面event对应的文件描述符关闭，这个event就会被自动删除，
				// 但是实际关闭fd之后该event仍然存在，这里采用手动删除changes数组中该fd对应的event
				syscall.Close(eventFd)
				delEvent(eventFd, &changes)
				log.Printf("client %s:%d quit!\n", strings.Replace(strings.Trim(fmt.Sprint(fdToClient[eventFd].Addr), "[]"), " ", ".", -1), fdToClient[eventFd].Port)
			}
		}
	}
}

func delEvent(fd int, changes *[]syscall.Kevent_t) {
	length := len(*changes)
	if length == 1 && int((*changes)[0].Ident) == fd {
		*changes = (*changes)[:]
		return
	}
	for i:=0; i<length; i++ {
		if int((*changes)[i].Ident) == fd {
			if i == length-1 {
				*changes = (*changes)[0:i]
			} else if i == 0 {
				*changes = (*changes)[i+1:]
			} else {
				*changes = append((*changes)[0:i], (*changes)[i+1:]...)
			}
			return
		}
	}
}

// SockaddrToAddr simple switched to SockaddrInet4
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
