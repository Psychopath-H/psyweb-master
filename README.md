# PsyWeb
PsyCache是仿照极客兔兔的geeweb，geeorm，geerpc实现的微服务框架，框架中提供了web，orm，rpc三个大类的支持，内置丰富的工具类库，是一个微服务的解决方案。

主要工作：
### 1.web框架部分
- 封装Go语言内置的HTTP网络编程基础接口，使用前缀树算法存储路由节点，对路由访问进行分组控制；
-  支持自定义中间件设计，实现静态资源服务，支持不同格式数据(HTML,JSON,XML)，HTML模板的返回；
- 提供参数处理机制，使用反射和验证器验证POST方法参数传递是否合法；
- 提供日志记录工具，并能分级输出；
- 提供错误记录工具，对外开放错误处理接口；参考ants，提供了高性能协程池管理工具；
- 提供https的支持，提供了基于basic和JWT的Token认证工具；

### 1.orm框架部分
- 封装Go语言database/sql标准库，支持MySQL数据库的连接，使用接口隔离不同数据库差异，后续可进行不同实现；
-  使用反射获取任意struct对象名称和字段映射为数据库中的表，提供了数据库的创建，删除，链式操作的CRUD接口；
-  使用反射获取绑定结构体的钩子，支持CRUD前后调用钩子；
- 封装事务，用户自定义回调函数实现原子操作；结构体变更时，支持数据库表字段的自动迁移；

### 1.rpc框架部分
- 使用encoding/gob实现消息的编解码(序列化与反序列化)；
- 实现了一个支持异步和并发的高性能客户端和服务端，并提供了连接超时和服务端处理超时的处理机制；
- 通过随机选择和 Round Robin 轮询调度算法实现服务端负载均衡；
- 实现了简易注册中心，支持服务注册和接受心跳功能；客户端实现基于注册中心的服务发现机制；
- 在服务端提供服务容错功能(限流和熔断)，并使用jaeger进行单节点处的链路追踪；

#  Prerequisites
- **Golang** v1.21 or later
- **Jaeger** v2.30.0 or later

# Installation

借助于 [Go module] 的支持(Go 1.11+), 只需要添加如下引入

import "github.com/Psychopath-H/psyweb-master"

接着只需 `go [build|run|test]` 将会自动导入依赖.

或者，你也可以直接安装 `psyweb-master` 包, 运行一下命令

$ go get -u github.com/Psychopath-H/psyweb-master

# Usage

### web框架部分使用示例
```
package main  
  
import (  
   "fmt"  
   "github.com/Psychopath-H/psyweb-master/psygo"
    psyLog "github.com/Psychopath-H/psyweb-master/psygo/logger"  
   "github.com/Psychopath-H/psyweb-master/psygo/pool"
   "github.com/Psychopath-H/psyweb-master/psygo/psyerror"
   "github.com/Psychopath-H/psyweb-master/psygo/token"
   "html/template"
   "log"
   "net/http"
   "sync"
   "time"
)  
  
type student struct {  
   Name string  
   Age  int8  
}  
  
type User struct {  
   Name      string   `xml:"name" json:"name" binding:"required" validate:"required"`  
   Age       int      `xml:"age" json:"age" validate:"required,max=50,min=18"`  
   Addresses []string `json:"addresses"`  
   Email     string   `json:"email"`  
}  
  
// FormatAsDate 是在html模板里使用自己定义的函数对数据格式进行渲染  
func FormatAsDate(t time.Time) string {  
   year, month, day := t.Date()  
   return fmt.Sprintf("%d-%02d-%02d", year, month, day)  
}  
  
// onlyForV2是v2组别的中间件  
func onlyForV2() psygo.HandlerFunc {  
   return func(c *psygo.Context) {  
      // Start timer  
      t := time.Now()  
      // Process request  
      c.Next()  
      // Calculate resolution time  
      log.Printf("[%d] %s in %v", c.StatusCode, c.Req.RequestURI, time.Since(t))  
   }  
}  
  
func main() {  
   r := psygo.Default()  
   //在模板中使用自己定义的函数进行渲染  
   r.SetFuncMap(template.FuncMap{  
      "FormatAsDate": FormatAsDate,  
   })  
   //用户访问localhost:9999/assets/css/psy.css，最终返回/blog/static/css/psy.css。  
   //实现了静态服务器  
   //expect localhost:9999/assets/css/psy.css  
   r.Static("/assets", "./static")  
  
   //不使用分组，纯html文件，不加载任何模板  
   r.GET("/html", func(c *psygo.Context) {  
      //expect localhost:9999/html  
      c.HTML(http.StatusOK, "<h1>你好 huqinxin</h1>")  
   })  
  
   //根据指定文件加载模板  
   r.GET("/htmlTemplate", func(c *psygo.Context) {  
      //expect localhost:9999/htmlTemplate  
      _ = c.HTMLTemplate(http.StatusOK, "login.html", "huqinxin", "template/login.html", "template/header.html")  
   })  
  
   //加载html模板  
   r.GET("/htmlTemplateGlob", func(c *psygo.Context) {  
      //expect localhost:9999/htmlTemplateGlob  
      _ = c.HTMLTemplateGlob(http.StatusOK, "login.html", "huqinxin", "template/*.html")  
   })  
  
   //先将模板加载进内存，然后再渲染  
   //r.LoadHTMLGlob("template/*")  
   r.LoadHTMLGlobByConf()  
   r.GET("/login", func(c *psygo.Context) {  
      //expect localhost:9999/login  
      c.TemplateLoaded(http.StatusOK, "login.html", psygo.H{  
         "Name": "huqinxin",  
      })  
   })  
  
   stu1 := &student{Name: "huqinxin", Age: 24}  
   stu2 := &student{Name: "Jack", Age: 22}  
   //模板渲染例子1  
   r.GET("/students", func(c *psygo.Context) {  
      // expect http://localhost:9999/students  
      c.TemplateLoaded(http.StatusOK, "arr.tmpl", psygo.H{  
         "title":  "gee",  
         "stuArr": [2]*student{stu1, stu2},  
      })  
   })  
   //模板渲染例子2  
   r.GET("/date", func(c *psygo.Context) {  
      // expect http://localhost:9999/date  
      c.TemplateLoaded(http.StatusOK, "custom_func.tmpl", psygo.H{  
         "title": "gee",  
         "now":   time.Date(2019, 8, 17, 0, 0, 0, 0, time.UTC),  
      })  
   })  
  
   //v1分组，GET请求，各种格式数据返回形式的具体使用  
   v1 := r.Group("/v1")  
   {  
      v1.GET("/html", func(c *psygo.Context) {  
         //expect localhost:9999/v1/html  
         c.HTML(http.StatusOK, "<h1>Hello huqinxin</h1>")  
      })  
  
      v1.GET("/string", func(c *psygo.Context) {  
         // expect localhost:9999/v1/string?name=huqinxin  
         c.String(http.StatusOK, "hello %s, you're at %s\n", c.GetQuery("name"), c.Path)  
      })  
  
      v1.GET("/json", func(c *psygo.Context) {  
         // expect localhost:9999/v1/json  
         c.JSON(http.StatusOK, psygo.H{  
            "Name":   "huqinxin",  
            "base":   "HangZhou",  
            "school": "TJU",  
         })  
      })  
  
      v1.GET("/xml", func(c *psygo.Context) {  
         //expect localhost:9999/v1/xml  
         user := &User{  
            "huqinxin",  
            23,  
            []string{"天津", "杭州"},  
            "17364525694@163.com",  
         }  
         c.XML(http.StatusOK, user)  
      })  
  
      v1.GET("/redirect", func(ctx *psygo.Context) {  
         // expect localhost:9999/v1/redirect  
         ctx.Redirect(http.StatusFound, "/html")  
      })  
   }  
  
   //v2分组，文件的上传与下载  
   v2 := r.Group("/v2")  
   v2.Use(onlyForV2()) // v2 group middleware  
   {  
      v2.GET("/DownloadFiles", func(c *psygo.Context) {  
         // expect http://localhost:9999/v2/DownloadFiles 从服务器上下载文件，文件实现保存在了服务器上  
         //ctx.File("template/MyExcelTest.xlsx")  
         //ctx.FileAttachment("template/MyExcelTest.xlsx", "h.xlsx")         c.FileFromFS("MyExcelTest.xlsx", http.Dir("template"))  
      })  
      v2.POST("/PostFiles", func(c *psygo.Context) {  
         // expect http://localhost:9999/v2/PostFiles POST的Body里有文件数据  
         //file := c.FormFile("file") 单个  
         files := c.FormFiles("file") //多个  
         for _, file := range files {  
            err := c.SaveUploadedFile(file, "./upload/"+file.Filename)  
            if err != nil {  
               log.Println(err)  
               return  
            }  
         }  
         c.JSON(http.StatusOK, psygo.H{  
            "status": "ok",  
            "files":  files,  
         })  
      })  
   }  
  
   //v3分组，参数处理  
   v3 := r.Group("/v3")  
   {  
      v3.GET("/queryArray", func(c *psygo.Context) {  
         // expect http://localhost:9999/v3/queryArray??id=1&age=24&name=huqinxin  
         name, ok := c.GetQueryArray("name")  
         if ok {  
            c.JSON(http.StatusOK, fmt.Sprintf("Query success, name is %s", name))  
         }  
      })  
  
      v3.GET("/queryMap", func(c *psygo.Context) {  
         // expect http://localhost:9999/v3/queryMap?user[id]=1&user[name]=huqinxin  
         value, ok := c.GetQueryMap("user")  
         if ok {  
            c.JSON(http.StatusOK, value)  
         }  
      })  
  
      v3.POST("/postFormArray", func(c *psygo.Context) {  
         // expect http://localhost:9999/v3/postFormArray post方法的Body里写有name=huqinxin  
         name, ok := c.GetPostFormArray("name")  
         if ok {  
            c.JSON(http.StatusOK, fmt.Sprintf("Post success, name is %s", name))  
         }  
      })  
  
      v3.POST("/postFormMap", func(c *psygo.Context) {  
         // expect http://localhost:9999/v3/postFormArray post方法的Body写有addressMap[home]=hangzhou  addressMap[school]=tianjin  
         value, ok := c.GetPostFormMap("addressMap")  
         if ok {  
            c.JSON(http.StatusOK, value)  
         }  
      })  
  
      v3.POST("/postJsonParams", func(c *psygo.Context) {  
         // expect http://localhost:9999/v3/postJsonParams post方法的Body写有JSON的数据  
         user := &User{}  
         //psygo.DisableLocalBindValidation()  
         psygo.EnableJsonDecoderDisallowUnknownFields()  
         err := c.BindJson(user)  
         //err := c.Bind(user)  
         if err == nil {  
            c.JSON(http.StatusOK, user)  
         } else {  
            log.Println(err)  
         }  
      })  
  
      v3.POST("/postXMLParams", func(c *psygo.Context) {  
         // expect http://localhost:9999/v3/postXMLParams post方法的Body写有XML的数据  
         user := &User{}  
         err := c.BindXML(user)  
         if err == nil {  
            c.JSON(http.StatusOK, user)  
         } else {  
            log.Println(err)  
         }  
      })  
   }  
  
   //v4分组，日志记录工具  
   _ = r.SetLogPathWithConf()  
   v4 := r.Group("/v4")  
   {  
      v4.GET("/logDebugLevel", func(c *psygo.Context) {  
         // expect http://localhost:9999/v4/logDebugLevel  
         c.Engine.Logger.SetLogWriterOnFile("./log", "debug.log", psyLog.LevelDebug)  
         _ = c.Engine.Logger.Debug("debug as followed: ...")  
      })  
  
      v4.GET("/logInfoLevel", func(c *psygo.Context) {  
         // expect http://localhost:9999/v4/logInfoLevel  
         c.Engine.Logger.SetLogWriterOnFile("./log", "info.log", psyLog.LevelInfo)  
         _ = c.Engine.Logger.Info("info as followed: ...")  
      })  
  
      v4.GET("/logErrorLevel", func(c *psygo.Context) {  
         // expect http://localhost:9999/v4/logErrorLevel  
         c.Engine.Logger.SetLogWriterOnFile("./log", "error.log", psyLog.LevelError)  
         _ = c.Engine.Logger.Error("error as followed: ...")  
      })  
  
      v4.GET("/logWithConf", func(c *psygo.Context) {  
         // expect http://localhost:9999/v4/logWithConf  
         _ = c.Engine.Logger.SetLogWriter(psyLog.LevelDebug)  
         _ = c.Engine.Logger.Debug("debug as followed: ...")  
      })  
  
      v4.GET("/logTextFormatter", func(c *psygo.Context) {  
         // expect http://localhost:9999/v4/logTextFormatter  
         c.Engine.Logger.SetLogWriterOnFile("./log", "debug.log", psyLog.LevelDebug)  
         c.Engine.Logger.Formatter = &psyLog.TextFormatter{}  
         _ = c.Engine.Logger.WithFields(psyLog.Fields{  
            "name": "huqinxin",  
            "id":   23,  
         }).Debug("debug as followed: ...")  
      })  
  
      v4.GET("/logLevelTest", func(c *psygo.Context) {  
         // expect http://localhost:9999/v4/logLevelTest  
         c.Engine.Logger.Level = psyLog.LevelInfo  
         c.Engine.Logger.SetLogWriterOnFile("./log", "debug.log", psyLog.LevelDebug)  
         c.Engine.Logger.SetLogWriterOnFile("./log", "info.log", psyLog.LevelInfo)  
         c.Engine.Logger.SetLogWriterOnFile("./log", "error.log", psyLog.LevelError)  
         _ = c.Engine.Logger.Debug("debug as followed: ...")  
         _ = c.Engine.Logger.Info("info as followed: ...")  
         _ = c.Engine.Logger.Error("error as followed: ...")  
      })  
   }  
  
   v5 := r.Group("/v5")  
   {  
      v5.POST("/errorTool", func(c *psygo.Context) {  
         // expect http://localhost:9999/v5/errorTool post方法的Body写有JSON的数据  
         c.Engine.Logger.Level = psyLog.LevelDebug  
         errDealer := psyerror.Default()  
         errDealer.Result(func(psyError *psyerror.PsyError) {  
            _ = c.Engine.Logger.Debug(psyError.Error())  
            c.JSON(http.StatusBadRequest, psyError.Error())  
         })  
         user := &User{}  
         psygo.EnableJsonDecoderDisallowUnknownFields()  
         err := c.BindJson(user)  
         errDealer.Put(err)  
  
      })  
   }  
  
   v6 := r.Group("/v6")  
   {  
      v6.GET("/psyPool", func(c *psygo.Context) {  
         // expect http://localhost:9999/v6/psyPool  
         p, _ := pool.NewPool(50000)  
         defer p.Release()  
         runSamples := 5  
         var wg sync.WaitGroup  
  
         syncCalculateSum := func() {  
            demoFunc()  
            wg.Done()  
         }  
         for i := 0; i < runSamples; i++ {  
            wg.Add(1)  
            _ = p.Submit(syncCalculateSum)  
         }  
         wg.Wait()  
         fmt.Printf("running goroutines: %d\n", p.Running())  
         fmt.Printf("finish all tasks.\n")  
      })  
  
      v6.GET("/psyPoolLimitedCap", func(c *psygo.Context) {  
         // expect http://localhost:9999/v6/psyPoolLimitedCap  
         //p, _ := psypool.NewPool(2)         p, _ := pool.NewPoolWithConf()  
         currentTime := time.Now().UnixMilli()  
         var wg sync.WaitGroup  
         wg.Add(5)  
         _ = p.Submit(func() {  
            time.Sleep(1 * time.Second)  
            fmt.Println(1)  
            wg.Done()  
         })  
         _ = p.Submit(func() {  
            time.Sleep(2 * time.Second)  
            fmt.Println(2)  
            wg.Done()  
         })  
         _ = p.Submit(func() {  
            time.Sleep(3 * time.Second)  
            fmt.Println(3)  
            wg.Done()  
         })  
         _ = p.Submit(func() {  
            time.Sleep(4 * time.Second)  
            fmt.Println(4)  
            wg.Done()  
         })  
         _ = p.Submit(func() {  
            time.Sleep(5 * time.Second)  
            fmt.Println(5)  
            wg.Done()  
         })  
         wg.Wait()  
         fmt.Printf("time:%#vs \n", float32(time.Now().UnixMilli()-currentTime)/1000)  
         c.JSON(http.StatusOK, "success")  
      })  
  
   }  
  
   v7 := r.Group("/v7")  
   {  
      var secrets = psygo.H{  
         "foo":      psygo.H{"email": "foo@bar.com", "phone": "123433"},  
         "austin":   psygo.H{"email": "austin@example.com", "phone": "666"},  
         "lena":     psygo.H{"email": "lena@guapa.com", "phone": "523443"},  
         "huqinxin": psygo.H{"email": "17364525694@163.com", "phone": "17364525694"},  
      }  
      accounts := psygo.Accounts{  
         "foo":      "bar",  
         "austin":   "1234",  
         "lena":     "hello2",  
         "manu":     "4321",  
         "huqinxin": "666",  
      }  
      v7.Use(psygo.BasicAuth(accounts))  
      v7.GET("/basicAuth", func(c *psygo.Context) {  
         // expect http://localhost:9999/v7/basicAuth Header的方法体中应该要有psygo.Accounts的basic64编码数据  
         user, _ := c.Get(psygo.AuthUserKey)  
         if secret, exist := secrets[user.(string)]; exist {  
            c.JSON(http.StatusOK, psygo.H{"user": user, "secret": secret})  
         } else {  
            c.JSON(http.StatusOK, psygo.H{"user": user, "secret": "NO SECRET :("})  
         }  
      })  
   }  
   v8 := r.Group("/v8")  
   {  
      jwt := &token.JWTAuth{}  
      jwt.SetAuthFailHandler(func(c *psygo.Context, err error) {  
         c.JSON(http.StatusUnauthorized, err.Error())  
      })  
      jwt.SetRefreshTime(time.Second * 15)  
      v8.Use(jwt.AuthInterceptor())  
      v8.GET("/login", func(c *psygo.Context) {  
         // expect http://localhost:9999/v8/login 获得JWT的编码数据  
         if err := jwt.CreateTokenWithDuration(c, "huqinxin", 23, time.Second*30); err != nil {  
            jwt.AuthFailHandler(c, err)  
         }  
         c.JSON(http.StatusOK, "login succeed and token is send")  
      })  
  
      v8.GET("/profile", func(c *psygo.Context) {  
         // expect http://localhost:9999/v8/profile 通过获得的JWT数据去访问这个路由  
         c.JSON(http.StatusOK, "Access Succeed")  
      })  
   }  
  
   //检验错误恢复中间件  
   r.GET("/panic", func(c *psygo.Context) {  
      names := []string{"huqinxin"}  
      c.String(http.StatusOK, names[100])  
   })  
  
   r.Run(":9999")  
   //r.RunTLS(":9999", "key/server.pem", "key/server.key")  
}  
  
func demoFunc() {  
   time.Sleep(10 * time.Millisecond)  
   fmt.Println("Hello World!")  
}
```

### orm框架部分使用示例
使用orm框架前，你需要在本地开启mysql服务
在 psymaster-web/blog/service/user_test.go 文件下给出了orm框架的使用测试示例

### 微服务整体使用示例
使用rpc框架对单节点进行链路追踪时需要在本地192.168.100.100先开启jaeger服务
服务端：
```
package main  
  
import (  
   "errors"  
   "github.com/Psychopath-H/psyweb-master/goodsCenter/service"   "github.com/Psychopath-H/psyweb-master/psygo"   "github.com/Psychopath-H/psyweb-master/psygo/tracer"   "github.com/opentracing/opentracing-go"   "github.com/uber/jaeger-client-go"   "github.com/uber/jaeger-client-go/config"   "log"   "net"   "net/http"   "rpc"   "rpc/breaker"   "rpc/registry"   "sync"   "time")  
  
func startRegistry(wg *sync.WaitGroup) {  
   l, _ := net.Listen("tcp", ":9999")  
   registry.HandleHTTP()  
   wg.Done()  
   _ = http.Serve(l, nil)  
}  
  
func startServer(registryAddr string, wg *sync.WaitGroup) { // registryAddr -> "http://localhost:9999/_rpc_/registry"  
   var gs service.GoodsService  
   l, _ := net.Listen("tcp", ":0")  
   server := rpc.NewServer()  
   _ = server.Register(&gs)  
  
   server.SetLimiter(20, 100, time.Millisecond*time.Duration(200))  
  
   settings := breaker.Settings{}  
   //降级实现  
   settings.Fallback = func(err error) (any, error) {  
      return "降级处理", errors.New("downgrading solution invoked")  
   }  
   server.CircuitBreaker = breaker.NewCircuitBreaker(settings)  
  
   registry.Heartbeat(registryAddr, "tcp@"+l.Addr().String(), 0)  
   wg.Done()  
   server.Accept(l)  
}  
  
func main() {  
   r := psygo.Default()  
   registryAddr := "http://localhost:9999/_rpc_/registry"  
   var wg sync.WaitGroup  
   wg.Add(1)  
   go startRegistry(&wg)  
   wg.Wait()  
  
   time.Sleep(time.Second)  
   wg.Add(1)  
   go startServer(registryAddr, &wg)  
   wg.Wait()  
  
   //使用链路追踪  
   createTracer, closer, err := tracer.CreateTracer("goodsCenter", &config.SamplerConfig{  
      Type:  jaeger.SamplerTypeConst,  
      Param: 1,  
   }, &config.ReporterConfig{  
      LogSpans:          true,  
      CollectorEndpoint: "http://192.168.100.100:14268/api/traces",  
   }, config.Logger(jaeger.StdLogger))  
   if err != nil {  
      panic(err)  
   }  
   defer closer.Close()  
  
   v1 := r.Group("/goodsCenter")  
   {  
      v1.GET("/jaeger", func(c *psygo.Context) {  
         span := createTracer.StartSpan("Upstream Service")  
         defer span.Finish()  
         DownstreamService(createTracer, span)  
         c.JSON(http.StatusOK, "jaeger invoked")  
      })  
   }  
   _ = r.Run(":9001")  
  
}  
  
func DownstreamService(createTracer opentracing.Tracer, span opentracing.Span) {  
   log.Println("DownstreamService is invoked")  
   startSpan := createTracer.StartSpan("Downstream Service", opentracing.ChildOf(span.Context()))  
   defer startSpan.Finish()  
}
```
客户端：
```
package main  
  
import (
   "context"  
   "github.com/Psychopath-H/psyweb-master/orderCenter/service"
   "github.com/Psychopath-H/psyweb-master/psygo"
   "log"
   "net/http"
   "rpc/xclient"
   "sync"
   "time"
)  
  
func main() {  
   registryAddr := "http://localhost:9999/_rpc_/registry"  
  
   r := psygo.Default()  
  
   v1 := r.Group("/orderCenter")  
   {  
      v1.GET("/sum", func(c *psygo.Context) {  
  
         d := xclient.NewRegistryDiscovery(registryAddr, 0)  
         xc := xclient.NewXClient(d, xclient.RandomSelect, nil)  
         var reply int  
         var err error  
         defer func() { _ = xc.Close() }()  
         // send request & receive response  
         var wg sync.WaitGroup  
         for i := 0; i < 5; i++ {  
            wg.Add(1)  
            go func(i int) {  
               defer wg.Done()  
               err = xc.Call(context.Background(), "GoodsService.Sum", &service.Args{Price1: i, Price2: i * i}, &reply)  
               logSumPrint(err, reply)  
            }(i)  
         }  
         wg.Wait()  
         if err != nil {  
            c.JSON(http.StatusInternalServerError, "rpc failed")  
         } else {  
            c.JSON(http.StatusOK, "rpc succeed")  
         }  
      })  
   }  
  
   {  
      v1.GET("/sum_limiting", func(c *psygo.Context) {  
         d := xclient.NewRegistryDiscovery(registryAddr, 0)  
         xc := xclient.NewXClient(d, xclient.RandomSelect, nil)  
         var reply int  
         var err error  
         defer func() { _ = xc.Close() }()  
         // send request & receive response  
         var wg sync.WaitGroup  
         for i := 0; i < 100; i++ {  
            wg.Add(1)  
            go func(i int) {  
               defer wg.Done()  
               err = xc.Call(context.Background(), "GoodsService.Sum", &service.Args{Price1: i, Price2: i * i}, &reply)  
               logSumPrint(err, reply)  
            }(i)  
         }  
         wg.Wait()  
  
         if err != nil {  
            c.JSON(http.StatusInternalServerError, "rpc failed")  
         } else {  
            c.JSON(http.StatusOK, "rpc succeed")  
         }  
      })  
   }  
  
   {  
      v1.GET("/sum_melted", func(c *psygo.Context) {  
         d := xclient.NewRegistryDiscovery(registryAddr, 0)  
         xc := xclient.NewXClient(d, xclient.RandomSelect, nil)  
         var reply int  
         var err error  
         defer func() { _ = xc.Close() }()  
         // send request & receive response  
         var wg sync.WaitGroup  
         for i := 0; i < 20; i++ {  
            wg.Add(1)  
            go func(i int) {  
               defer wg.Done()  
               if i < 10 {  
                  err = xc.Call(context.Background(), "GoodsService.Error", &service.Args{Price1: i, Price2: i * i}, &reply)  
                  logErrorPrint(err, reply)  
               } else {  
                  err = xc.Call(context.Background(), "GoodsService.Sum", &service.Args{Price1: i, Price2: i * i}, &reply)  
                  logSumPrint(err, reply)  
               }  
  
            }(i)  
         }  
  
         time.Sleep(time.Second * 12)  
  
         for i := 20; i < 40; i++ {  
            wg.Add(1)  
            go func(i int) {  
               defer wg.Done()  
               err = xc.Call(context.Background(), "GoodsService.Sum", &service.Args{Price1: i, Price2: i * i}, &reply)  
               logSumPrint(err, reply)  
            }(i)  
         }  
         wg.Wait()  
  
         if err != nil {  
            c.JSON(http.StatusInternalServerError, "rpc failed")  
         } else {  
            c.JSON(http.StatusOK, "rpc succeed")  
         }  
      })  
   }  
  
   _ = r.Run(":9002")  
  
}  
  
func logErrorPrint(err error, reply int) {  
   if err != nil {  
      log.Printf("%s error: %v", "GoodsService.Error", err)  
   } else {  
      log.Printf("%s success: reply = %d", "GoodsService.Error", reply)  
   }  
}  
  
func logSumPrint(err error, reply int) {  
   if err != nil {  
      log.Printf("%s error: %v", "GoodsService.Sum", err)  
   } else {  
      log.Printf("%s success: reply = %d", "GoodsService.Sum", reply)  
   }  
}
```

```
访问http://localhost:9002/orderCenter/sum
客户端和服务端结果如下示例：

```
server：
2024/01/08 20:03:31 rpc registry path: /_rpc_/registry
2024/01/08 20:03:32 rpc server: register GoodsService.Error
2024/01/08 20:03:32 rpc server: register GoodsService.Sleep
2024/01/08 20:03:32 rpc server: register GoodsService.Sum
2024/01/08 20:03:32 tcp@[::]:56237 send heart beat to registry http://localhost:9999/_rpc_/registry
2024/01/08 20:03:32 debug logging disabled
2024/01/08 20:03:32 Initializing logging reporter
2024/01/08 20:03:32 debug logging disabled
2024/01/08 20:03:32 Route  GET - /goodsCenter/jaeger
2024/01/08 20:07:32 tcp@[::]:56237 send heart beat to registry http://localhost:9999/_rpc_/registry

client:
{"log_level":"INFO","log_time":"2024/01/08 - 20:03:56","msg":"conf/app.toml file not load，because not exist"}
2024/01/08 20:03:56 Route  GET - /orderCenter/sum
2024/01/08 20:03:56 Route  GET - /orderCenter/sum_limiting
2024/01/08 20:03:56 Route  GET - /orderCenter/sum_melted
2024/01/08 20:05:49 rpc registry: refresh servers from registry http://localhost:9999/_rpc_/registry
2024/01/08 20:05:49 GoodsService.Sum success: reply = 20
2024/01/08 20:05:49 GoodsService.Sum success: reply = 2
2024/01/08 20:05:49 GoodsService.Sum success: reply = 2
2024/01/08 20:05:49 GoodsService.Sum success: reply = 12
2024/01/08 20:05:49 GoodsService.Sum success: reply = 0
[PSYGO] | 2024/01/08 - 20:05:49 |  200  |     17.1473ms |             ::1  | GET      "/orderCenter/sum" 
```

熔断示例：
访问http://localhost:9002/orderCenter/sum_melted
服务端结果如下示例：

```
2024/01/08 20:11:15 rpc registry: refresh servers from registry http://localhost:9999/_rpc_/registry
2024/01/08 20:11:15 GoodsService.Error error: rpcServer internal error
2024/01/08 20:11:15 GoodsService.Error error: rpcServer internal error
2024/01/08 20:11:15 GoodsService.Sum error: downgrading solution invoked
2024/01/08 20:11:15 GoodsService.Error error: rpcServer internal error
2024/01/08 20:11:15 GoodsService.Error error: rpcServer internal error
2024/01/08 20:11:15 GoodsService.Sum error: downgrading solution invoked
2024/01/08 20:11:15 GoodsService.Error error: rpcServer internal error
2024/01/08 20:11:15 GoodsService.Error error: rpcServer internal error
2024/01/08 20:11:15 GoodsService.Error error: downgrading solution invoked
2024/01/08 20:11:15 GoodsService.Error error: downgrading solution invoked
2024/01/08 20:11:15 GoodsService.Sum error: downgrading solution invoked
2024/01/08 20:11:15 GoodsService.Error error: downgrading solution invoked
2024/01/08 20:11:15 GoodsService.Sum error: downgrading solution invoked
2024/01/08 20:11:15 GoodsService.Sum error: downgrading solution invoked
2024/01/08 20:11:15 GoodsService.Error error: downgrading solution invoked
2024/01/08 20:11:15 GoodsService.Sum error: downgrading solution invoked
2024/01/08 20:11:15 GoodsService.Sum error: downgrading solution invoked
2024/01/08 20:11:15 GoodsService.Sum error: downgrading solution invoked
2024/01/08 20:11:15 GoodsService.Sum error: downgrading solution invoked
2024/01/08 20:11:15 GoodsService.Sum error: downgrading solution invoked
2024/01/08 20:11:27 rpc registry: refresh servers from registry http://localhost:9999/_rpc_/registry
2024/01/08 20:11:27 GoodsService.Sum success: reply = 1560
2024/01/08 20:11:27 GoodsService.Sum success: reply = 870
2024/01/08 20:11:27 GoodsService.Sum success: reply = 600
2024/01/08 20:11:27 GoodsService.Sum success: reply = 420
2024/01/08 20:11:27 GoodsService.Sum success: reply = 1260
2024/01/08 20:11:27 GoodsService.Sum success: reply = 1122
2024/01/08 20:11:27 GoodsService.Sum success: reply = 462
2024/01/08 20:11:27 GoodsService.Sum success: reply = 506
2024/01/08 20:11:27 GoodsService.Sum success: reply = 600
2024/01/08 20:11:27 GoodsService.Sum success: reply = 650
2024/01/08 20:11:27 GoodsService.Sum success: reply = 1190
2024/01/08 20:11:27 GoodsService.Sum success: reply = 1190
2024/01/08 20:11:27 GoodsService.Sum success: reply = 756
2024/01/08 20:11:27 GoodsService.Sum success: reply = 930
2024/01/08 20:11:27 GoodsService.Sum success: reply = 992
2024/01/08 20:11:27 GoodsService.Sum success: reply = 812
2024/01/08 20:11:27 GoodsService.Sum success: reply = 1056
2024/01/08 20:11:27 GoodsService.Sum success: reply = 1332
2024/01/08 20:11:27 GoodsService.Sum success: reply = 1482
2024/01/08 20:11:27 GoodsService.Sum success: reply = 1406
[PSYGO] | 2024/01/08 - 20:11:27 |  200  |     12.01329s |             ::1  | GET      "/orderCenter/sum_melted"
```
