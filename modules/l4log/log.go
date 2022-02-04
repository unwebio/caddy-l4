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

	go func(pr *io.PipeReader) {
		if _, err := io.Copy(os.Stdout, pr); err != nil {
			h.logger.Error("upstream connection", zap.Error(err))
		}
		return
	}(pr)

	nextc := *cx
	nextc.Conn = nextConn{
		Conn:   cx,
		Reader: io.TeeReader(cx, pw),
		pipe:   pw,
	}

	return next.Handle(&nextc)
}

type nextConn struct {
	net.Conn
	io.Reader
	pipe *io.PipeWriter
}

func (nc nextConn) Read(p []byte) (n int, err error) {
	n, err = nc.Reader.Read(p)
	if err == io.EOF {
		nc.pipe.Close()
	}
	return
}

// Interface guard
var _ layer4.NextHandler = (*Handler)(nil)
