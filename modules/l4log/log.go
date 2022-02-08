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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/certmagic"
	"github.com/mholt/caddy-l4/layer4"
	"go.uber.org/zap"
)

func init() {
	caddy.RegisterModule(Handler{})
}

// Handler is a simple handler that writes what it reads.
type Handler struct {
	StorageRaw json.RawMessage `json:"storage,omitempty" caddy:"namespace=caddy.storage inline_key=module"`
	storage    certmagic.Storage
	log        *zap.Logger
}

// CaddyModule returns the Caddy module information.
func (Handler) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "layer4.handlers.log",
		New: func() caddy.Module { return new(Handler) },
	}
}

func (h *Handler) Provision(ctx caddy.Context) error {
	h.log = ctx.Logger(h)
	if h.StorageRaw != nil {
		val, err := ctx.LoadModule(h, "StorageRaw")
		if err != nil {
			return fmt.Errorf("loading storage module: %v", err)
		}
		cmStorage, err := val.(caddy.StorageConverter).CertMagicStorage()
		if err != nil {
			return fmt.Errorf("creating storage configuration: %v", err)
		}
		h.storage = cmStorage
	}
	if h.storage == nil {
		return fmt.Errorf("l4log.storage is required")
	}
	return nil
}

// Handle handles the connection.
func (h Handler) Handle(cx *layer4.Connection, next layer4.Handler) error {
	req, ok := cx.GetVar("http_request").(*http.Request)
	if !ok {
		h.log.Warn("l4log does not handle non-http traffic")
		return next.Handle(cx)
	}
	targetUri := req.URL
	targetUri.Scheme = "http"
	targetUri.Host = req.Host

	reqR, reqW := io.Pipe()
	resR, resW := io.Pipe()

	tr := io.TeeReader(cx, reqW)
	mr := io.MultiWriter(cx, resW)

	reqTime := time.Now().Format(time.RFC3339)
	var resTime string

	go func() {
		// need to start the request and response PipeReaders ASAP
		// so their corresponding PipeWriters can be written to without
		// blocking
		var reqContent []byte
		go func() {
			c, err := io.ReadAll(reqR)
			if err != nil {
				h.log.Warn("Failed to log request traffic", zap.Error(err))
			}
			reqContent = c
		}()

		resContent, resErr := io.ReadAll(resR)
		if resErr != nil {
			h.log.Error("Failed to log response traffic", zap.Error(resErr))
			return
		}

		warcR, warcW := io.Pipe()
		defer warcW.Close()
		req := Message{reqTime, reqContent}
		res := Message{resTime, resContent}
		warc := CreateWarc(req, res, targetUri.String(), "t.0.d.0")

		go func() {
			warcContent, err := io.ReadAll(warcR)
			if err != nil {
				h.log.Error("Failed to render WARC", zap.Error(err))
				return
			}
			fmt.Println("Storing WARC content")
			err = h.storage.Store(warc.Info.Uuid.String()+".warc", warcContent)
			if err != nil {
				h.log.Error("Failed to store WARC", zap.Error(err))
			}
		}()

		warc.Render(warcW)
	}()

	nextcx := nextConn{cx, tr, mr, reqW}
	err := next.Handle(cx.Wrap(&nextcx))
	resTime = time.Now().Format(time.RFC3339)
	resW.Close()
	return err
}

type nextConn struct {
	*layer4.Connection
	io.Reader
	io.Writer
	reqW *io.PipeWriter
}

func (nc nextConn) Read(p []byte) (n int, err error) {
	n, err = nc.Reader.Read(p)
	if err == io.EOF {
		nc.reqW.Close()
	}
	return
}

func (nc nextConn) Write(p []byte) (n int, err error) {
	return nc.Writer.Write(p)
}

// Interface guard
var _ layer4.NextHandler = (*Handler)(nil)
