package main

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/evanphx/wildcat"
)

type httpCodec struct {
	parser        *wildcat.HTTPParser
	contentLength int
	buf           []byte
}

func (hc *httpCodec) parse(data []byte) (int, []byte, error) {
	bodyOffset, err := hc.parser.Parse(data)
	if err != nil {
		return 0, nil, err
	}

	contentLength := hc.getContentLength()
	if contentLength > -1 {
		bodyEnd := bodyOffset + contentLength
		var body []byte
		if len(data) >= bodyEnd {
			body = data[bodyOffset:bodyEnd]
		}
		return bodyEnd, body, nil
	}

	// Transfer-Encoding: chunked
	lastChunk := []byte("0\r\n\r\n")
	if idx := bytes.Index(data[bodyOffset:], lastChunk); idx != -1 {
		bodyEnd := idx + 5
		var body []byte
		if len(data) >= bodyEnd {
			req, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(data[:bodyEnd])))
			if err != nil {
				return bodyEnd, nil, err
			}
			body, _ = io.ReadAll(req.Body)
		}
		return bodyEnd, body, nil
	}

	// Requests without a body.
	CRLF := []byte("\r\n\r\n")
	if idx := bytes.Index(data, CRLF); idx != -1 {
		return idx + 4, nil, nil
	}

	return 0, nil, errors.New("invalid http request")
}

func (hc *httpCodec) getContentLength() int {
	var contentLengthKey = []byte("Content-Length")

	if hc.contentLength != -1 {
		return hc.contentLength
	}

	val := hc.parser.FindHeader(contentLengthKey)
	if val != nil {
		i, err := strconv.ParseInt(string(val), 10, 0)
		if err == nil {
			hc.contentLength = int(i)
		}
	}

	return hc.contentLength
}

func (hc *httpCodec) resetParser() {
	hc.contentLength = -1
}

func (hc *httpCodec) reset() {
	hc.resetParser()
	hc.buf = hc.buf[:0]
}
