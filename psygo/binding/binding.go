package binding

import "net/http"

const (
	MIMEJSON = "application/json"
	MIMEXML  = "application/xml"
)

// Binding 实现了Binding接口的具体结构可以作为绑定器验证参数post传递过来的参数是否符合要求
type Binding interface {
	Name() string
	Bind(*http.Request, any) error
}

var (
	JSON = jsonBinding{}
	XML  = xmlBinding{}
	//Form = formBinding{}
)

func Default(contentType string) Binding {
	switch contentType {
	case MIMEJSON:
		return JSON
	case MIMEXML:
		return XML
	default:
		return nil
	}
}
