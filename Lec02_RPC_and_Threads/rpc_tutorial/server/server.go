package main

import (
	"log"
	"net"
	"net/http"
	"net/rpc"

	"learn-rpc/data"
)

func main() {
	arith := new(data.Arith)
	rpc.Register(arith)
	rpc.HandleHTTP()
	l, e := net.Listen("tcp", ":1234")
	if e != nil {
		log.Fatal("listen error:", e)
	}
	http.Serve(l, nil)
}
