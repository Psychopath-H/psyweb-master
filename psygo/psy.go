package psygo

import (
	"errors"
	"github.com/Psychopath-H/psyweb-master/psygo/config"
	psyLog "github.com/Psychopath-H/psyweb-master/psygo/logger"
	"html/template"
	"log"
	"net/http"
	"path"
	"strings"
	"sync"
)

// HandlerFunc 定义了访问路由(网络请求)时的处理函数
type HandlerFunc func(c *Context)

// Engine 实现了ServeHTTP方法，所有打到特定端口的请求都会被路由到这里
type Engine struct {
	*RouterGroup
	router        *router
	groups        []*RouterGroup     //存储所有的路由分组
	htmlTemplates *template.Template //用于提前将html模板加载进内存
	funcMap       template.FuncMap   //自定义模板渲染函数。
	pool          sync.Pool          //用于存储Context上下文的池子,避免频繁创建context影响效率
	Logger        *psyLog.Logger
}

// RouterGroup 某个具体路由分组
type RouterGroup struct {
	prefix      string        // 该路由分组的前缀
	middlewares []HandlerFunc //中间件是应用在分组上的，还需要存储应用在分组上的中间件
	parent      *RouterGroup  //需要知道当前分组的父亲是谁
	engine      *Engine       //所有的分组共享一个engine实例
}

// New 新建一个引擎
func New() *Engine {
	engine := &Engine{router: newRouter()}
	engine.RouterGroup = &RouterGroup{engine: engine}
	engine.groups = []*RouterGroup{engine.RouterGroup}
	engine.pool.New = func() any {
		return engine.allocateContext()
	}
	return engine
}

// allocateContext 分配一个内存
func (engine *Engine) allocateContext() any {
	return &Context{Engine: engine}
}

func Default() *Engine {
	engine := New()
	engine.Use(Logger(), Recovery())
	engine.Logger = psyLog.Default()
	return engine
}

// SetLogPathWithConf 通过配置设置日志存储位置
func (engine *Engine) SetLogPathWithConf() error {
	logPath, ok := config.Conf.Log["path"]
	if ok {
		engine.Logger.SetLogPath(logPath.(string))
		return nil
	}
	return errors.New("config log.path not exist")
}

// SetFuncMap 设置html模板渲染函数
func (engine *Engine) SetFuncMap(funcMap template.FuncMap) {
	engine.funcMap = funcMap
}

// LoadHTMLGlob 将html模板提前加载进内存
func (engine *Engine) LoadHTMLGlob(pattern string) {
	engine.htmlTemplates = template.Must(template.New("").Funcs(engine.funcMap).ParseGlob(pattern))
}

func (engine *Engine) LoadHTMLGlobByConf() {
	pattern, ok := config.Conf.Template["pattern"]
	if !ok {
		panic("config template.pattern not exist")
	}
	engine.htmlTemplates = template.Must(template.New("").Funcs(engine.funcMap).ParseGlob(pattern.(string)))
}

// Group 在该组基础上定义一个新的 RouterGroup(实现了分组嵌套),所有的 groups 共享同一个 engine 实例
func (group *RouterGroup) Group(prefix string) *RouterGroup {
	engine := group.engine
	newGroup := &RouterGroup{
		prefix: group.prefix + prefix,
		parent: group,
		engine: engine,
	}
	engine.groups = append(engine.groups, newGroup)
	return newGroup
}

// addRoute 添加路由和方法
func (group *RouterGroup) addRoute(method string, comp string, handler HandlerFunc) {
	pattern := group.prefix + comp
	log.Printf("Route %4s - %s", method, pattern)
	group.engine.router.addRoute(method, pattern, handler)
}

// GET defines the method to add GET request
func (group *RouterGroup) GET(pattern string, handler HandlerFunc) {
	group.addRoute("GET", pattern, handler)
}

// POST defines the method to add POST request
func (group *RouterGroup) POST(pattern string, handler HandlerFunc) {
	group.addRoute("POST", pattern, handler)
}

// DELETE defines the method to add DELETE request
func (group *RouterGroup) DELETE(pattern string, handler HandlerFunc) {
	group.addRoute("DELETE", pattern, handler)
}

// PUT defines the method to add PUT request
func (group *RouterGroup) PUT(pattern string, handler HandlerFunc) {
	group.addRoute("PUT", pattern, handler)
}

// PATCH defines the method to add PATCH request
func (group *RouterGroup) PATCH(pattern string, handler HandlerFunc) {
	group.addRoute("PATCH", pattern, handler)
}

// OPTIONS defines the method to add OPTIONS request
func (group *RouterGroup) OPTIONS(pattern string, handler HandlerFunc) {
	group.addRoute("OPTIONS", pattern, handler)
}

// HEAD defines the method to add HEAD request
func (group *RouterGroup) HEAD(pattern string, handler HandlerFunc) {
	group.addRoute("HEAD", pattern, handler)
}

// Run defines the method to start a http server
func (engine *Engine) Run(addr string) (err error) {
	return http.ListenAndServe(addr, engine)
}

// RunTLS 开启https的支持
func (engine *Engine) RunTLS(addr, certFile, keyFile string) {
	err := http.ListenAndServeTLS(addr, certFile, keyFile, engine.Handler())
	if err != nil {
		log.Fatal(err)
	}
}

func (engine *Engine) Handler() http.Handler {
	return engine
}

// ServeHTTP engine实现了ServeHTTP方法后，打到特定端口的请求都会被路由到这里去处理
func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	c := engine.pool.Get().(*Context)
	//当我们接收到一个具体请求时，要判断该请求适用于哪些中间件，
	//在这里我们简单通过 URL 的前缀来判断。得到中间件列表后，赋值给 c.handlers。
	var middlewares []HandlerFunc
	for _, group := range engine.groups {
		if strings.HasPrefix(req.URL.Path, group.prefix) {
			middlewares = append(middlewares, group.middlewares...)
		}
	}
	c.reset(w, req)
	c.handlers = middlewares
	c.Engine = engine
	engine.router.handle(c) //注意这里的调用顺序，先将中间件的handler放到context中，再把本请求路由匹配的handler放进去。
	engine.pool.Put(c)
}

// Use 函数，将中间件应用到某个 Group。
func (group *RouterGroup) Use(middlewares ...HandlerFunc) {
	group.middlewares = append(group.middlewares, middlewares...)
}

//建立静态handler

// 还记得我们之前设计动态路由的时候，支持通配符*匹配多级子路径。比如路由规则/assets/*filepath，
// 可以匹配/assets/开头的所有的地址。例如/assets/js/geektutu.js，匹配后，参数filepath就赋值为js/geektutu.js。
// 那如果我么将所有的静态文件放在/usr/web目录下，那么filepath的值即是该目录下文件的相对地址。映射到真实的文件后，将文件返回，静态服务器就实现了。
// 找到文件后，如何返回这一步，net/http库已经实现了。因此，gee 框架要做的，仅仅是解析请求的地址，映射到服务器上文件的真实地址，
// 交给http.FileServer处理就好了
func (group *RouterGroup) createStaticHandler(relativePath string, fs http.FileSystem) HandlerFunc {
	absolutePath := path.Join(group.prefix, relativePath) //拼接路径
	//http.StripPrefix内部是一个能将*http.Request传递过来的路径剥去传入的参数absolutePath
	//然后剩下的相对路径是相对于fs http.FileSystem这个文件系统下的，因此可以通过这个函数隐藏框架内部的文件名。
	fileServer := http.StripPrefix(absolutePath, http.FileServer(fs)) //这个函数的作用是从请求的URL中去除指定的前缀absolutePath，然后将请求传递给后面的handler处理。
	return func(c *Context) {
		file := c.Param("filepath")
		//检查文件是否存在或者我们是否有权限去访问文件
		if _, err := fs.Open(file); err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		fileServer.ServeHTTP(c.Writer, c.Req)
	}
}

// Static 设置静态路由 relativePath: /assets  root:./static
func (group *RouterGroup) Static(relativePath string, root string) {
	handler := group.createStaticHandler(relativePath, http.Dir(root))
	urlPattern := path.Join(relativePath, "/*filepath")
	//注册 GET 的handlers
	group.GET(urlPattern, handler)
}
