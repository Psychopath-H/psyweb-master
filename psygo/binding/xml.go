package binding

import (
	"encoding/xml"
	"errors"
	"io"
	"net/http"
)

type xmlBinding struct {
}

func (x xmlBinding) Bind(r *http.Request, obj any) error {
	if r == nil || r.Body == nil {
		return errors.New("invalid request")
	}
	return decodeXML(r.Body, obj)
}

func (x xmlBinding) Name() string {
	return "xml"
}

func decodeXML(body io.ReadCloser, obj any) error {
	decoder := xml.NewDecoder(body) //创建一个新的 XML 解码器,解码器将从该输入流中读取数据并解码为 Go 数据结构。
	if err := decoder.Decode(obj); err != nil {
		return err
	}
	return threePartValidate(obj) //使用第三方验证器
}
