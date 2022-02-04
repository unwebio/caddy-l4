// Copyright 2020 Matthew Holt
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package l4log

import (
	"io"
	"os"
	"net"
	"fmt"
	"log"

	"github.com/caddyserver/caddy/v2"
	"github.com/unwebio/caddy-l4/layer4"
	"go.uber.org/zap"
)

func init() {
	caddy.RegisterModule(Handler{})
}

// Handler is a simple handler that writes what it reads.
type Handler struct{
	logger *zap.Logger
}

// CaddyModule returns the Caddy module information.
func (Handler) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "layer4.handlers.log",
		New: func() caddy.Module { return new(Handler) },
	}
}

func (h *Handler) Provision(ctx caddy.Context) {
	h.logger = ctx.Logger(h)
}

// Handle handles the connection.
func (h *Handler) Handle(cx *layer4.Connection, next layer4.Handler) error {
  pr, pw := io.Pipe()
	nextc := *cx
	nextc.Conn = nextConn{
		Conn:   cx,
		logger: pw,
		logged: pr,
	}
	return next.Handle(&nextc)
}

type nextConn struct {
	net.Conn
	io.Reader
	logger *io.PipeWriter
	logged *io.PipeReader
}

func (nc nextConn) Read(p []byte) (n int, err error) {
	fmt.Println("Reading from nextConn")
	n, err = nc.Conn.Read(p)
	if n > 0 {
		fmt.Printf("Writing to logger: %s\n", p[:n])
		if n, err := nc.logger.Write(p[:n]); err != nil {
			return n, err
		}
	}
	if err == io.EOF {
		fmt.Println("Reading from nextConn :: EOF")
		if _, err := io.Copy(os.Stdout, nc.logged); err != nil {
			log.Fatal(err)
		}
		nc.logger.Close()
	}
	return
}

// Interface guard
var _ layer4.NextHandler = (*Handler)(nil)
