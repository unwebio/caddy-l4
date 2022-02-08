package l4log

import (
	_ "embed"
	"io"
	"log"
	"text/template"

	"github.com/google/uuid"
)

type warcInfoRecord struct {
	Uuid      uuid.UUID
	Timestamp string
}

type warcMessageRecord struct {
	Uuid      uuid.UUID
	Timestamp string
	Content   string
}

func (rec warcMessageRecord) ContentLength() int {
	return len(rec.Content)
}

type Message struct {
	Timestamp string
	Content   []byte
}

type Warc struct {
	Info      warcInfoRecord
	Request   warcMessageRecord
	Response  warcMessageRecord
	TargetUri string
	PublicIp  string
}

func CreateWarc(req Message, res Message) Warc {
	info := warcInfoRecord{uuid.New(), req.Timestamp}
	request := warcMessageRecord{uuid.New(), req.Timestamp, string(req.Content)}
	response := warcMessageRecord{uuid.New(), res.Timestamp, string(res.Content)}
	return Warc{info, request, response, "a", "b"}
}

//go:embed warc.tmpl
var tmpl string

func (warc Warc) Render(pw *io.PipeWriter) {
	var t = template.Must(template.New("WARC").Parse(tmpl))
	err := t.Execute(pw, warc)
	if err != nil {
		log.Println("executing template:", err)
	}
}
