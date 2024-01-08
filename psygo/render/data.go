package render

import "net/http"

// Data contains ContentType and bytes data.
type Data struct {
	ContentType string
	Data        []byte
}

// Render (Data) writes data with custom ContentType.
func (r Data) RenderData(w http.ResponseWriter, code int) (err error) {
	r.WriteContentType(w)
	w.WriteHeader(code)
	_, err = w.Write(r.Data)
	return
}

// WriteContentType (Data) writes custom ContentType.
func (r Data) WriteContentType(w http.ResponseWriter) {
	writeContentType(w, r.ContentType)
}
