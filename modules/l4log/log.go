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
	"fmt"
	"io"
	"net"

	// "net"

	"github.com/caddyserver/caddy/v2"
	"github.com/unwebio/caddy-l4/layer4"
)

func init() {
	caddy.RegisterModule(Handler{})
}

// Handler is a simple handler that writes what it reads.
type Handler struct{}

// CaddyModule returns the Caddy module information.
func (Handler) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "layer4.handlers.log",
		New: func() caddy.Module { return new(Handler) },
	}
}

// Handle handles the connection.
func (h *Handler) Handle(cx *layer4.Connection, next layer4.Handler) (err error) {
	return next.Handle(cx)
}

type nextConn struct {
  net.Conn
	io.Reader
}

func (nc nextConn) Read(p []byte) (n int, err error) {
	n, err = nc.Reader.Read(p)
	fmt.Printf("Read %d bytes\n", n)
	if n > 0 {
		fmt.Printf("Bytes read: %s", p[:n])
	}
	return
}

// Interface guard
var _ layer4.NextHandler = (*Handler)(nil)
