package main

import (
	"fmt"
	"io"
	"net"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
)

// 2019/10/04 15:29:52
func main() {
	port := pflag.IntP("port", "p", 2999, "listen port")
	pflag.Parse()

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("Listen error: %v", err)
	}
	log.Infof("Listen on: %d", *port)
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Errorf("Accept error: %v", err)
			continue
		}
		go handleConn(conn)
	}
}

// handleConn TODO
// 2019/10/04 15:32:26
func handleConn(conn net.Conn) {
	log.Infof("Client %v comming", conn.RemoteAddr())

	err := readClientAuth2(conn)
	if err != nil {
		log.Errorf("%v", err)
		return
	}
	log.Infof("Read client auth OK")
	err = replyAuth(conn)
	if err != nil {
		log.Errorf("Reply auth error: %v", err)
		return
	}

	cmd, dest, port, err := readClientRequest(conn)
	if err != nil {
		log.Errorf("read client request error: %v", err)
		return
	}
	log.Infof("Read client request ok")

	switch cmd {
	case cmdConnect:
		remoteConn, err := net.Dial("tcp", fmt.Sprintf("%v:%v", dest, port))
		if err != nil {
			log.Errorf("dial to remote: %v on port: %v error: %v", dest, port, err)
			return
		}

		err = replyClientRequest(conn)
		if err != nil {
			log.Errorf("reply client request error: %v", err)
			return
		}
		log.Infof("reply client request OK")

		log.Infof("tunnel working...")
		go io.Copy(remoteConn, conn)
		io.Copy(conn, remoteConn)
	case cmdBind:
		log.Errorf("Not support bind cmd")
		//TODO
	case cmdUDP:
		log.Errorf("Not support udp cmd")
		//TODO
	}
}

//client auth
//  |Version | NMethods | Methods |
//  |   1    |      1   |     n   |

//server reply
//  | Version | Method |
//  |     1   |     1  |

//client request
//  | Version | CMD   | RSV   | ATYPE  | DST.ADDR  | DST.PORT |
//  |    1    |   1   |     1 |   1    |  variable |    2     |
//server reply
//  | Version | REP | RSV | ATYPE | BND.ADDR  |  BND.PORT|
//  |    1    |   1 |  1  |   1   |  variable |   2      |

const (
	layoutVersion  = 0
	layoutNMethods = 1
	layoutMethods  = 2
)
const (
	version5 = 5
	noAuth   = 0
	success  = 0
	rsv      = 0
)

// readClientAuth TODO
// 2019/10/04 16:56:32
func readClientAuth(r io.Reader) error {
	buf := make([]byte, 2)
	n, err := io.ReadFull(r, buf)
	if err != nil {
		return err
	}
	if n < len(buf) {
		return fmt.Errorf("Read less then %d", len(buf))
	}

	version := buf[layoutVersion]
	nmethods := buf[layoutNMethods]
	if version != version5 {
		return fmt.Errorf("version not 5")
	}
	if nmethods <= 0 {
		return fmt.Errorf("Nmethods must greater than 0")
	}
	buf2 := make([]byte, nmethods)
	n, err = io.ReadFull(r, buf2)
	if err != nil {
		return err
	}
	if n < int(nmethods) {
		return fmt.Errorf("Read less then %d", nmethods)
	}

	NoAuth := false
	for _, b := range buf2 {
		if b == noAuth {
			NoAuth = true
			break
		}
	}
	if !NoAuth {
		return fmt.Errorf("Only support method: 0 (no auth)")
	}
	return nil
}

type authState byte

const (
	stateVersion authState = iota
	stateNMethods
	stateMethods
	stateEnd
)
const (
	lenVersion  = 1
	lenNMethods = 1
)

// readClientAuth2 TODO
// 2019/10/04 20:56:54
func readClientAuth2(r io.Reader) error {
	state := stateVersion
	var (
		buf      []byte
		nmethods int
	)

OUTTER:
	for {
		switch state {
		case stateVersion:
			buf = make([]byte, lenVersion)
			n, err := io.ReadFull(r, buf)
			if err != nil {
				return err
			}
			if n != lenVersion {
				return fmt.Errorf("Insufficient data when read version")
			}
			if buf[0] != version5 {
				return fmt.Errorf("Version not match")
			}
			state = stateNMethods
		case stateNMethods:
			buf = make([]byte, lenNMethods)
			n, err := io.ReadFull(r, buf)
			if err != nil {
				return err
			}
			if n != lenVersion {
				return fmt.Errorf("Insufficient data when read nmethods")
			}
			nmethods = int(buf[0])
			log.Infof("nmethods: %v", nmethods)
			state = stateMethods
		case stateMethods:
			buf = make([]byte, nmethods)
			n, err := io.ReadFull(r, buf)
			if err != nil {
				return err
			}
			if n != len(buf) {
				return fmt.Errorf("Insufficient data when read methods")
			}
			noAuthen := false
			for _, v := range buf {
				if v == noAuth {
					noAuthen = true
				}
			}
			if !noAuthen {
				return fmt.Errorf("Only support no auth")
			}
			state = stateEnd
		case stateEnd:
			break OUTTER
		}
	}
	return nil
}

// replyAuth TODO
// 2019/10/04 17:06:09
func replyAuth(conn net.Conn) error {
	_, err := conn.Write([]byte{version5, noAuth})
	return err
}

const (
	layoutCMD   = 1
	layoutRSV   = 2
	layoutAType = 3
	layoutADDR  = 4
)

const (
	cmdConnect = 1
	cmdBind    = 2
	cmdUDP     = 3
)

const (
	addressIPV4   = 1
	addressDomain = 3
	addressIPV6   = 4
)

// readClientRequest TODO
// 2019/10/04 17:08:54
func readClientRequest(r io.Reader) (int, string, int, error) {
	// read ver cmd rsv atype
	buf := make([]byte, 4)
	n, err := io.ReadFull(r, buf)
	if err != nil {
		return 0, "", 0, err
	}
	if n < 4 {
		return 0, "", 0, fmt.Errorf("read less then 4")
	}

	version := buf[layoutVersion]
	cmd := buf[layoutCMD]
	rsv := buf[layoutRSV]
	atype := buf[layoutAType]

	if version != version5 || rsv != 0 {
		return 0, "", 0, fmt.Errorf("version not valid or rsv not valid")
	}
	var dest string

	switch atype {
	case addressIPV4:
		buf = make([]byte, 4)
		n, err = io.ReadFull(r, buf)
		if err != nil {
			return 0, "", 0, err
		}
		if n != len(buf) {
			return 0, "", 0, fmt.Errorf("read less then %d", len(buf))
		}
		dest = net.IPv4(buf[0], buf[1], buf[2], buf[3]).String()

	case addressIPV6:
		buf = make([]byte, 16)
		n, err = io.ReadFull(r, buf)
		if err != nil {
			return 0, "", 0, err
		}
		if n != len(buf) {
			return 0, "", 0, fmt.Errorf("read less then %d", len(buf))
		}
		var ipv6 net.IP
		copy(ipv6, buf)
		dest = ipv6.String()

	case addressDomain:
		buf = make([]byte, 1)
		n, err = io.ReadFull(r, buf)
		if err != nil {
			return 0, "", 0, err
		}
		if n != len(buf) {
			return 0, "", 0, fmt.Errorf("read less then %d", len(buf))
		}
		domainNameLen := buf[0]
		buf = make([]byte, domainNameLen)
		n, err = io.ReadFull(r, buf)
		if err != nil {
			return 0, "", 0, err
		}
		if n != len(buf) {
			return 0, "", 0, fmt.Errorf("read less then %d", len(buf))
		}
		dest = string(buf)
	}
	log.Infof("dest: %v", dest)

	buf = make([]byte, 2)
	n, err = io.ReadFull(r, buf)
	if err != nil {
		return 0, "", 0, err
	}
	if n < len(buf) {
		return 0, "", 0, fmt.Errorf("read less then %v", len(buf))
	}
	port := int(buf[0])<<8 | int(buf[1])
	log.Infof("dest port: %v", port)
	return int(cmd), dest, port, nil
}

// replyClientRequest TODO
// 2019/10/04 17:27:48
func replyClientRequest(conn net.Conn) error {
	_, err := conn.Write([]byte{version5, success, rsv, addressIPV4, 0, 0, 0, 0, 1, 2})
	return err
}
