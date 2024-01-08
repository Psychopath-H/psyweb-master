package binding

//const defaultMemory = 32 << 20
//
//type formBinding struct{}
//
//func (formBinding) Name() string {
//	return "form"
//}
//
//func (formBinding) Bind(req *http.Request, obj any) error {
//	if err := req.ParseForm(); err != nil {
//		return err
//	}
//	if err := req.ParseMultipartForm(defaultMemory); err != nil && !errors.Is(err, http.ErrNotMultipart) {
//		return err
//	}
//	if err := mapForm(obj, req.Form); err != nil {
//		return err
//	}
//	return threePartValidate(obj)
//}
