package render

import "net/http"

type Render interface {
	RenderData(w http.ResponseWriter, statusCode int) error
	WriteContentType(w http.ResponseWriter)
}

func writeContentType(w http.ResponseWriter, value string) {
	w.Header().Set("Content-Type", value)
}
