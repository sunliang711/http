package proxyServer

import (
	"bufio"
	"bytes"
	"fmt"
	"httpTunnel/httpUtil"
	"io"
	"net"
	"net/http"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
)

// Server TODO
// 2019/10/03 23:16:33
type Server struct {
	IP   string
	Port int
}

// NewServer TODO
// 2019/10/03 23:19:27
func NewServer(port int) *Server {
	srv := &Server{
		Port: port,
	}

	return srv
}

// Start TODO
// 2019/10/03 23:20:19
func (srv *Server) Start() error {
	address := fmt.Sprintf("%v:%v", srv.IP, srv.Port)
	ln, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	log.Infof("Listen on %v", address)
	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		go handleConn(conn)
	}
}

// handleConn TODO
// 2019/10/03 23:25:31
func handleConn(conn net.Conn) {
	log.Infof("New client: %v comming", conn.RemoteAddr())
	r := bufio.NewReader(conn)
	// request line
	requestLine, err := httpUtil.ParseRequestLine(r)
	if err != nil {
		log.Errorf("ParseRequestLine error: %v", err)
		return
	}
	log.Infof("Client request line info:")
	log.Infof("method: %v", requestLine.Method)
	log.Infof("path: %v", requestLine.Path)
	log.Infof("major: %v", requestLine.Major)
	log.Infof("minor: %v", requestLine.Minor)

	// header
	headers, err := httpUtil.ParseRequestHeaders(r)
	if err != nil {
		log.Errorf("ParseRequestHeaders error: %v", err)
		return
	}
	log.Infof("headers: %v", headers)

	// read body
	var (
		body []byte
	)
	contentLength, exist := headers["Content-Length"]
	if exist {
		bufLen, err := strconv.Atoi(contentLength)
		if err != nil {
			log.Errorf("Content-Length invalid")
			return
		}
		buf := make([]byte, bufLen)
		n, err := io.ReadFull(r, buf)
		if err != nil && err != io.EOF {
			log.Errorf("Read body error: %v", err)
			return
		}
		if n < bufLen {
			log.Errorf("Read body size less than Content-Length")
			return
		}
		body = buf
	}
	log.Infof("body: %v", body)

	serveProxy(conn, requestLine, headers, body)
}

const (
	methodConnect = "CONNECT"
	methodGet     = "GET"
	methodPost    = "POST"
)

// serveProxy TODO
// 2019/10/04 11:48:32
func serveProxy(conn net.Conn, requestLine *httpUtil.RequestLine, headers map[string]string, body []byte) (err error) {
	switch requestLine.Method {
	case methodConnect:
		remoteHost := requestLine.Path
		log.Infof("Dial to remote host: %v", remoteHost)
		remoteConn, err := net.DialTimeout("tcp", remoteHost, time.Second*2)
		if err != nil {
			log.Errorf("Dial remote host: %v error: %v", remoteHost, err)
			return err
		}
		log.Infof("Dial to remote host: %v OK", remoteHost)

		respToClient := fmt.Sprintf("HTTP/%v.%v 200 Connection established\r\n\r\n", requestLine.Major, requestLine.Minor)
		log.Infof("Resp to client: %v", respToClient)
		conn.Write([]byte(respToClient))

		go io.Copy(remoteConn, conn)
		io.Copy(conn, remoteConn)
	case methodGet:
		bodyAsReader := bytes.NewReader(body)
		log.Infof("Prepare request to remote host: %v", requestLine.Path)
		req, err := http.NewRequest(methodGet, requestLine.Path, bodyAsReader)
		if err != nil {
			log.Errorf("Make new http request error: %v", err)
			return err
		}
		for k, v := range headers {
			log.Infof("Add header: %v->%v", k, v)
			req.Header.Add(k, v)
		}

		httpClient := http.Client{}
		log.Infof("Send request to remote host: %v", requestLine.Path)
		resp, err := httpClient.Do(req)
		if err != nil {
			log.Errorf("Request to remote host error: %v", err)
			return err
		}
		log.Info("Get response of remote host: %v", requestLine.Path)
		defer resp.Body.Close()
		defer conn.Close()
		log.Infof("Send response to client")
		remoteResponseToClient(resp, conn)
	case methodPost:

	}
	return
}

// remoteResponseToClient TODO
// 2019/10/04 12:18:46
func remoteResponseToClient(resp *http.Response, clientConn net.Conn) {
	statusLine := fmt.Sprintf("%v %v\r\n", resp.Proto, resp.Status)
	headers := ""
	for k, v := range resp.Header {
		headers += fmt.Sprintf("%v: %v\r\n", k, v)
	}
	headers += "\r\n"

	clientConn.Write([]byte(statusLine))
	clientConn.Write([]byte(headers))
	io.Copy(clientConn, resp.Body)
}
