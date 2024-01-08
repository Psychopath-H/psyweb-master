package render

import (
	"github.com/Psychopath-H/psyweb-master/psygo/internal/bytesconv"
	"html/template"
	"net/http"
)

type HTML struct {
	Data       any
	Name       string
	Template   *template.Template
	IsTemplate bool
}

func (h *HTML) RenderData(w http.ResponseWriter, statusCode int) error {
	h.WriteContentType(w)
	w.WriteHeader(statusCode)
	if h.IsTemplate {
		err := h.Template.ExecuteTemplate(w, h.Name, h.Data)
		return err
	}
	_, err := w.Write(bytesconv.StringToBytes(h.Data.(string)))
	return err
}

func (h *HTML) WriteContentType(w http.ResponseWriter) {
	writeContentType(w, "text/html; charset=utf-8")
}
