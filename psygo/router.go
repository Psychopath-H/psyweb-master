package psygo

import (
	"net/http"
	"strings"
)

// router 路由器，存储了前缀树根节点和处理方法
type router struct {
	// roots 存储了每个请求方法(GET,POST)的前缀树根节点
	roots map[string]*node
	//handlers 存储了每个特定方法特定路由请求的处理函数 例如:handlers['GET-/p/:lang/doc'], handlers['POST-/p/book']
	handlers map[string]HandlerFunc
}

// newRouter 新建一个路由器
func newRouter() *router {
	return &router{
		roots:    make(map[string]*node),
		handlers: make(map[string]HandlerFunc),
	}
}

// parsePattern 解析一下路由地址，只允许有一个*存在 例子: "/p/:lang/doc" -> [p :lang doc]
func parsePattern(pattern string) []string {
	vs := strings.Split(pattern, "/")

	parts := make([]string, 0)
	for _, item := range vs {
		if item != "" {
			parts = append(parts, item)
			if item[0] == '*' {
				break
			}
		}
	}
	return parts
}

// addRoute 通过前缀树添加路由 method:GET pattern:/p/:lang/doc
func (r *router) addRoute(method string, pattern string, handler HandlerFunc) {
	parts := parsePattern(pattern)
	key := method + "-" + pattern // 比如 GET-/p/:lang/doc
	_, ok := r.roots[method]
	if !ok { //如果该方法没有前缀树(往往在刚开始会这样)，就新建一个
		r.roots[method] = &node{}
	}
	r.roots[method].insert(pattern, parts, 0) //构建前缀树路由
	r.handlers[key] = handler                 //为路由添加方法
}

// getRoute 通过前缀树获得路由，并得到模糊参数
func (r *router) getRoute(method string, path string) (*node, map[string]string) {
	searchParts := parsePattern(path) //"/p/:lang/doc" -> [p :lang doc]
	params := make(map[string]string)
	root, ok := r.roots[method]
	if !ok {
		return nil, nil
	}
	dstNode := root.search(searchParts, 0)
	if dstNode != nil { //能够找到匹配的路由
		parts := parsePattern(dstNode.pattern)
		for index, part := range parts {
			//把实际参数取出来，建立一个映射
			if part[0] == ':' {
				params[part[1:]] = searchParts[index]
			}
			if part[0] == '*' && len(part) > 1 {
				params[part[1:]] = strings.Join(searchParts[index:], "/")
				break
			}
		}
		return dstNode, params
	}
	return nil, nil
}

func (r *router) handle(c *Context) {
	n, params := r.getRoute(c.Method, c.Path)

	if n != nil {
		c.Params = params
		key := c.Method + "-" + n.pattern
		//handle 函数中，将从路由匹配得到的 Handler 添加到 c.handlers列表中，执行c.Next()。
		c.handlers = append(c.handlers, r.handlers[key])
	} else {
		c.handlers = append(c.handlers, func(c *Context) {
			c.String(http.StatusNotFound, "404 NOT FOUND: %s\n", c.Path)
		})
	}
	c.Next()
}
