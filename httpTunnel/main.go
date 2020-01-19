package main

import "httpTunnel/proxyServer"

// 2019/10/03 22:56:23
func main() {

	srv := proxyServer.NewServer(9977)
	srv.Start()

}
