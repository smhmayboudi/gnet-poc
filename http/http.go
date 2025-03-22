package main

import (
	"flag"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/evanphx/wildcat"
	"github.com/panjf2000/gnet/v2"
)

type httpServer struct {
	gnet.BuiltinEventEngine

	addr      string
	multicore bool
	eng       gnet.Engine
}

func writeResponse(hc *httpCodec, body []byte) {
	// You may want to determine the URL path and write the corresponding response.
	// ...

	hc.buf = append(hc.buf, "HTTP/1.1 200 OK\r\nServer: gnet\r\nContent-Type: text/plain\r\nDate: "...)
	hc.buf = time.Now().AppendFormat(hc.buf, "Mon, 02 Jan 2006 15:04:05 GMT")
	hc.buf = append(hc.buf, "\r\nContent-Length: "...)
	hc.buf = append(hc.buf, strconv.Itoa(len(body))...)
	hc.buf = append(hc.buf, "\r\n\r\n"...)
	hc.buf = append(hc.buf, body...)
}

func (hs *httpServer) OnBoot(eng gnet.Engine) gnet.Action {
	hs.eng = eng
	log.Printf("echo server with multi-core=%t is listening on %s\n", hs.multicore, hs.addr)
	return gnet.None
}

func (hs *httpServer) OnOpen(c gnet.Conn) ([]byte, gnet.Action) {
	c.SetContext(&httpCodec{parser: wildcat.NewHTTPParser()})
	return nil, gnet.None
}

func (hs *httpServer) OnTraffic(c gnet.Conn) gnet.Action {
	hc := c.Context().(*httpCodec)
	buf, _ := c.Peek(-1)
	n := len(buf)

pipeline:
	nextOffset, body, err := hc.parse(buf)
	hc.resetParser()
	if err != nil {
		goto response
	}
	if len(buf) < nextOffset {
		goto response
	}
	writeResponse(hc, body)
	buf = buf[nextOffset:]
	if len(buf) > 0 {
		goto pipeline
	}
response:
	if len(hc.buf) > 0 {
		c.Write(hc.buf)
	}
	hc.reset()
	c.Discard(n - len(buf))
	return gnet.None
}

func main() {
	var port int
	var multicore, reuseport bool

	// Example command: go run echo.go --port 8080 --multicore=true --reuseport=true
	flag.IntVar(&port, "port", 8080, "--port 8080")
	flag.BoolVar(&multicore, "multicore", true, "--multicore true")
	flag.BoolVar(&reuseport, "reuseport", false, "--reuseport true")
	flag.Parse()
	echo := new(httpServer)
	log.Println("server exits:", gnet.Run(echo, fmt.Sprintf("tcp://:%d", port), gnet.WithMulticore(multicore), gnet.WithReusePort(reuseport)))
}
