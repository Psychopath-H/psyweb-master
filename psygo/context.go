package psygo

import (
	"errors"
	"github.com/Psychopath-H/psyweb-master/psygo/binding"
	"github.com/Psychopath-H/psyweb-master/psygo/render"
	"html/template"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
)

const MaxMultipartMemory = 32 << 20 //32M

type H map[string]any

type Context struct {
	//origin objects
	Writer     http.ResponseWriter
	Req        *http.Request
	Path       string            //请求的路径
	Method     string            //请求的方法
	Params     map[string]string //本次路由得到的模糊参数
	StatusCode int               //请求状态码
	queryCache url.Values        //query参数缓存
	formCache  url.Values        //form(表单)参数缓存
	handlers   []HandlerFunc     //本次请求上下文中所用到的中间件
	index      int               //中间件的下标索引
	Errors     errorMsgs         //存储了本context中框架内部产生的错误
	mu         sync.RWMutex
	Keys       map[string]any //存储了每个请求的key/value对
	sameSite   http.SameSite
	Engine     *Engine
}

func (c *Context) reset(w http.ResponseWriter, req *http.Request) {
	c.Writer = w
	c.Req = req
	c.Path = req.URL.Path
	c.Method = req.Method
	c.Params = nil
	c.StatusCode = 0
	c.queryCache = url.Values{}
	c.formCache = url.Values{}
	c.handlers = nil
	c.index = -1
}

//func newContext(w http.ResponseWriter, req *http.Request) *Context {
//	return &Context{
//		Writer: w,
//		Req:    req,
//		Path:   req.URL.Path,
//		Method: req.Method,
//		//day5
//		index: -1,
//	}
//}

func (c *Context) Status(code int) {
	c.StatusCode = code
	c.Writer.WriteHeader(code)
}

// requestHeader 获得request Header中key的值
func (c *Context) requestHeader(key string) string {
	return c.Req.Header.Get(key)
}

// Header is an intelligent shortcut for c.Writer.Header().Set(key, value).
// It writes a header in the response.
// If value == "", this method removes the header `c.Writer.Header().Del(key)`
func (c *Context) Header(key, value string) {
	if value == "" {
		c.Writer.Header().Del(key)
		return
	}
	c.Writer.Header().Set(key, value)
}

func (c *Context) Error(err error) *Error {
	if err == nil {
		panic("err is nil")
	}

	var parsedError *Error
	ok := errors.As(err, &parsedError)
	if !ok {
		parsedError = &Error{
			Err:  err,
			Type: ErrorTypePrivate,
		}
	}

	c.Errors = append(c.Errors, parsedError)
	return parsedError
}

// Set 设置用户名和密码
func (c *Context) Set(key string, value any) {
	c.mu.Lock()
	if c.Keys == nil {
		c.Keys = make(map[string]any)
	}
	c.Keys[key] = value
	c.mu.Unlock()
	return
}

// Get 获得用户名和密码
func (c *Context) Get(key string) (value any, ok bool) {
	c.mu.RLock()
	value, ok = c.Keys[key]
	c.mu.RUnlock()
	return
}

// HTML 使用纯html文本(不使用任何模板)
func (c *Context) HTML(statusCode int, html string) {
	c.Render(statusCode, &render.HTML{
		Data:       html,
		IsTemplate: false,
	})
}

// TemplateLoaded 使用已加载入内存的模板
func (c *Context) TemplateLoaded(statusCode int, name string, obj any) {
	c.Render(statusCode, &render.HTML{
		Data:       obj,
		IsTemplate: true,
		Template:   c.Engine.htmlTemplates,
		Name:       name,
	})
}

// HTMLTemplate 加载指定模板
func (c *Context) HTMLTemplate(statusCode int, name string, obj any, filename ...string) error {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(statusCode)
	t, err := template.New(name).ParseFiles(filename...)
	if err != nil {
		log.Println(err)
		return err
	}
	if err = t.Execute(c.Writer, obj); err != nil {
		return c.Error(err)
	}
	return nil
}

// HTMLTemplateGlob 批量加载模板
func (c *Context) HTMLTemplateGlob(statusCode int, name string, obj any, pattern string) error {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(statusCode)
	t, err := template.New(name).ParseGlob(pattern)
	if err != nil {
		log.Println(err)
		return err
	}
	if err = t.Execute(c.Writer, obj); err != nil {
		return c.Error(err)
	}
	return nil
}

// JSON 返回JSON格式的数据
func (c *Context) JSON(statusCode int, obj any) {
	c.Render(statusCode, &render.JSON{
		Data: obj,
	})
}

// XML 返回xml格式的数据
func (c *Context) XML(statusCode int, data any) {
	c.Render(statusCode, &render.XML{
		Data: data,
	})
}

// String 返回string（字符串）格式的数据
func (c *Context) String(statusCode int, format string, values ...any) {
	c.Render(statusCode, &render.String{
		Format: format,
		Data:   values,
	})
}

// Data 返回[]byte格式的数据
func (c *Context) Data(statusCode int, ContentType string, data []byte) {
	c.Render(statusCode, &render.Data{
		Data:        data,
		ContentType: ContentType,
	})
}

// Redirect 支持重定向操作
func (c *Context) Redirect(statusCode int, url string) {
	c.Render(statusCode, &render.Redirect{
		Code:     statusCode,
		Request:  c.Req,
		Location: url,
	})
}

// Render 根据参数传递的特定格式进行渲染
func (c *Context) Render(statusCode int, r render.Render) {
	c.StatusCode = statusCode
	if err := r.RenderData(c.Writer, statusCode); err != nil {
		_ = c.Error(err)
		c.Abort()
	}
}

// Param 获得模糊参数
func (c *Context) Param(key string) string {
	value, _ := c.Params[key]
	return value
}

// Fail 以JSON形式返回请求错误的信息
func (c *Context) Fail(statusCode int, err string) {
	c.index = len(c.handlers)
	c.JSON(statusCode, err)
}

// FormFile 获得根据传入的name得到的文件
func (c *Context) FormFile(name string) (*multipart.FileHeader, error) {
	//在调用 c.Request.FormFile() 方法之前，首先判断 c.Request.MultipartForm 是否为空是为了确认请求是否为多部分表单类型。
	//如果为空，意味着请求并非文件上传类型的请求，而是其他类型的表单提交，此时调用 c.Request.FormFile() 是没有意义的，因为该请求中并没有上传文件。
	//因此，这个检查是为了避免在非文件上传类型的请求中不必要地调用文件上传相关的方法，从而防止出现不必要的错误或异常情况。
	//if c.Req.MultipartForm == nil {
	//	if err := c.Req.ParseMultipartForm(MaxMultipartMemory); err != nil {
	//		return nil, err
	//	}
	//}
	f, fh, err := c.Req.FormFile(name) //把前面代码注释掉的原因是，FormFile方法里已经判断过了是否为多部分表单的请求，因此可以不再提前做判断
	if err != nil {
		return nil, err
	}
	_ = f.Close()
	return fh, err
}

// FormFiles 获得多部分表单中同名(name)的多个文件
func (c *Context) FormFiles(name string) []*multipart.FileHeader {
	multipartForm, err := c.MultipartForm()
	if err != nil {
		return make([]*multipart.FileHeader, 0)
	}
	return multipartForm.File[name]
}

// SaveUploadedFile 将上传上来的文件存储到指定位置
func (c *Context) SaveUploadedFile(file *multipart.FileHeader, dst string) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, src)
	return err
}

// MultipartForm 返回多部分表单
func (c *Context) MultipartForm() (*multipart.Form, error) {
	err := c.Req.ParseMultipartForm(MaxMultipartMemory)
	return c.Req.MultipartForm, err
}

// File 支持从web网站下载对应文件
func (c *Context) File(filePath string) {
	http.ServeFile(c.Writer, c.Req, filePath)
}

// FileAttachment 支持从web网站下载对应文件并进行改名
func (c *Context) FileAttachment(filepath, filename string) {
	if isASCII(filename) {
		c.Writer.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	} else {
		c.Writer.Header().Set("Content-Disposition", `attachment; filename*=UTF-8''`+url.QueryEscape(filename))
	}
	http.ServeFile(c.Writer, c.Req, filepath)
}

// FileFromFS 支持从文件系统下载对应文件 filepath是相对文件系统的路径
func (c *Context) FileFromFS(filepath string, fs http.FileSystem) {
	//把原来请求的路径恢复一下
	defer func(old string) {
		c.Req.URL.Path = old
	}(c.Req.URL.Path)

	c.Req.URL.Path = filepath

	http.FileServer(fs).ServeHTTP(c.Writer, c.Req)
}

// initQueryCache 初始化查询列表
func (c *Context) initQueryCache() {
	if c.Req != nil {
		c.queryCache = c.Req.URL.Query()
	} else {
		c.queryCache = url.Values{}
	}
}

// DefaultQuery 获得默认的查询
func (c *Context) DefaultQuery(key, defaultValue string) string {
	array, ok := c.GetQueryArray(key)
	if !ok {
		return defaultValue
	}
	return array[0]
}

// GetQuery 获得查询的参数
func (c *Context) GetQuery(key string) string {
	c.initQueryCache()
	return c.queryCache.Get(key)
}

// GetQueryArray 得到查询参数的列表
func (c *Context) GetQueryArray(key string) (values []string, ok bool) {
	c.initQueryCache()
	values, ok = c.queryCache[key]
	return
}

// GetQueryMap 获得以Map类型传递的参数
func (c *Context) GetQueryMap(key string) (map[string]string, bool) {
	c.initQueryCache()
	return c.getMap(c.queryCache, key)
}

func (c *Context) getMap(cache map[string][]string, key string) (map[string]string, bool) {
	dicts := make(map[string]string)
	exist := false
	for k, value := range cache {
		if i := strings.IndexByte(k, '['); i >= 1 && k[0:i] == key {
			if j := strings.IndexByte(k[i+1:], ']'); j >= 1 {
				exist = true
				dicts[k[i+1:][:j]] = value[0]
			}
		}
	}
	return dicts, exist
}

// initPostFromCache 初始化表单查询列表
func (c *Context) initPostFormCache() {
	if c.Req != nil {
		if err := c.Req.ParseMultipartForm(MaxMultipartMemory); err != nil {
			if !errors.Is(err, http.ErrNotMultipart) {
				log.Println(err)
			}
		}
		c.formCache = c.Req.PostForm
	} else {
		c.formCache = url.Values{}
	}
}

// GetPostForm 得到表单的第一项
func (c *Context) GetPostForm(key string) (string, bool) {
	if values, ok := c.GetPostFormArray(key); ok {
		return values[0], ok
	}
	return "", false
}

// PostFormArray 得到表单查询参数的列表
func (c *Context) PostFormArray(key string) (values []string) {
	values, _ = c.GetPostFormArray(key)
	return
}

// GetPostFormArray 得到表单查询参数的列表
func (c *Context) GetPostFormArray(key string) (values []string, ok bool) {
	c.initPostFormCache()
	values, ok = c.formCache[key]
	return
}

// GetPostFormMap 获得以Map类型传递的表单参数
func (c *Context) GetPostFormMap(key string) (map[string]string, bool) {
	c.initPostFormCache()
	return c.getMap(c.formCache, key)
}

func (c *Context) PostFormMap(key string) (dicts map[string]string) {
	dicts, _ = c.GetPostFormMap(key)
	return
}

// Bind 检查 Content-Type 去自动选择一个绑定格式
func (c *Context) Bind(obj any) error {
	b := binding.Default(filterFlags(c.Req.Header.Get("Content-Type")))
	if b == nil {
		return errors.New("can't match binding")
	}
	return c.MustBindWith(obj, b)
}

// BindJson 处理从post请求的body部分传递过来的data，将Json格式数据转换为Go里的数据结构
func (c *Context) BindJson(obj any) error {
	return c.MustBindWith(obj, binding.JSON)
}

// BindXML 处理从post请求的body部分传递过来的data，将XML格式数据转换为Go里的数据结构
func (c *Context) BindXML(obj any) error {
	return c.MustBindWith(obj, binding.XML)
}

// MustBindWith 在绑定发生错误时，会向response status code设置为400
func (c *Context) MustBindWith(obj any, bind binding.Binding) error {
	if err := c.ShouldBindWith(obj, bind); err != nil {
		return err
	}
	return nil
}

// ShouldBind 和 c.Bind()大致相同，但是对于当绑定发生错误时，并不把response status code设置为400
func (c *Context) ShouldBind(obj any) error {
	b := binding.Default(filterFlags(c.Req.Header.Get("Content-Type")))
	if b != nil {
		return errors.New("can't match binding")
	}
	return c.ShouldBindWith(obj, b)
}

func (c *Context) ShouldBindJSON(obj any) error {
	return c.ShouldBindWith(obj, binding.JSON)
}

func (c *Context) ShouldBindXML(obj any) error {
	return c.ShouldBindWith(obj, binding.XML)
}

func (c *Context) ShouldBindWith(obj any, bind binding.Binding) error {
	return bind.Bind(c.Req, obj)
}

// SetCookie 添加一个Set-Cookie头部放到ResponseWriter的头部中
func (c *Context) SetCookie(name, value string, maxAge int, path, domain string, secure, httpOnly bool) {
	if path == "" {
		path = "/"
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    url.QueryEscape(value),
		MaxAge:   maxAge,
		Path:     path,
		Domain:   domain,
		SameSite: c.sameSite,
		Secure:   secure,
		HttpOnly: httpOnly,
	})
}

//day5 当接收到请求后，匹配路由，该请求的所有信息都保存在Context中。
//中间件也不例外，接收到请求后，应查找所有应作用于该路由的中间件，
//保存在Context中，依次进行调用。为什么依次调用后，还需要在Context中保存呢？
//因为在设计中，中间件不仅作用在处理流程前，也可以作用在处理流程后，即在用户定义的 Handler 处理完毕后，还可以执行剩下的操作。

func (c *Context) Next() {
	//index作为context中的变量，这里即可以视为一个全局变量，任何中间件的c.Next()操作都会使得调用的中间件往后移动
	c.index++
	s := len(c.handlers)
	for ; c.index < s; c.index++ {
		c.handlers[c.index](c)
	}
}

// Abort prevents pending handlers from being called. Note that this will not stop the current handler.
// Let's say you have an authorization middleware that validates that the current request is authorized.
// If the authorization fails (ex: the password does not match), call Abort to ensure the remaining handlers
// for this request are not called.
func (c *Context) Abort() {
	c.index = len(c.handlers)
}
