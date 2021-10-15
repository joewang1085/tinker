# package http
## http.ListenAndServe(addr string, handler Handler)
    - addr: listen 地址
    - handler: 指定handler, 实现 ServeHTTP 即可

## Handler 接口
    ``` 
    type Handler interface {
	    ServeHTTP(ResponseWriter, *Request)
    }
    ```
##  默认的 DefaultServeMux
handler==nil时, 则使用默认的DefaultServeMux, 一般都是默认的DefaultServeMux, 它是一个ServeMux变量.  
ServeMux是HTTP请求的多路转接器.  
   - 注册HTTP handler:ServeMux内部的一个哈希表中添加一条记录  
   - 多路转接: 对每一个接收请求的URL匹配调用注册的handler
ServeMux 方法:  
![avatar](servemux.jpg)  
     
```
// DefaultServeMux is the default ServeMux used by Serve.
var DefaultServeMux = &defaultServeMux

var defaultServeMux ServeMux

type ServeMux struct {
    mu    sync.RWMutex // 读写锁
    m     map[string]muxEntry // 管理所有注册路由哈希表
    es    []muxEntry // 按pattern长度降序排列的匹配列表, 记录值均以/结尾
    hosts bool       // 是否存在hosts, 即不以'/'开头的pattern
}
注：url 分为“/”结尾和非“/”结尾两类， 以“/”结尾则表示“/*”都可以匹配，是范围路由，而非以“/”结尾则表示确定路由。因此es内保存的是以“/”结尾的范围路由，用来实现范围路由匹配的，按照url长度降序，则会优先匹配到长的url, 更精确。 但是 es是一个切片，但是查询效率一般，O(n). 其他的路由查找策略，gin使用字典树O(logn)，另外使用反射类的查询效率更低。

type muxEntry struct {
    h       Handler
    pattern string
}

// ServeHTTP dispatches the request to the handler whose
// pattern most closely matches the request URL.
func (mux *ServeMux) ServeHTTP(w ResponseWriter, r *Request) {
    if r.RequestURI == "*" {
	    if r.ProtoAtLeast(1, 1) {
		    w.Header().Set("Connection", "close")
	    }
	    w.WriteHeader(StatusBadRequest)
	    return
    }
    h, _ := mux.Handler(r)
    h.ServeHTTP(w, r)
}
```

ServeMux 的底层match逻辑:  
```
func (mux *ServeMux) match(path string) (h Handler, pattern string) {
	// 直接匹配成功的情况
	v, ok := mux.m[path]
	if ok {
		return v.h, v.pattern
	}
	// 寻找最接近的最长匹配，mux.es切片中包含了所有子树，并降序排列，因此遍历一次即可找出最接近的模式
	for _, e := range mux.es {
		if strings.HasPrefix(path, e.pattern) {
			return e.h, e.pattern
		}
	}
	return nil, ""
}
```
假设此时DefaultServeMux注册了两个模式: /a/, /a/b/，此时DefaultServeMux的结构为  
```
{
    m: {
        "/a/": { h: HandlerA, pattern: "/a/" },
        "/a/b/": { h: HandlerB, pattern: "/a/b" },
    },
    es: [{ h: HandlerB, pattern: "/a/b" }, { h: HandlerA, pattern: "/a/" }]
}
```
当请求路径为/a/b/c，将进入第二个if语句，在match方法中进行匹配：最终路径/a/b/c将返回handlerB  

   
## 注册 http handler

1. http.HandleFunc(pattern string, handleFunc func(ResponseWriter, *Request), 可以将一个 func(ResponseWriter, *Request) 函数注册成http handler
```
func HandleFunc(pattern string, handler func(ResponseWriter, *Request)) {
    DefaultServeMux.HandleFunc(pattern, handler)
}

type HandlerFunc func(ResponseWriter, *Request)

// ServeHTTP calls f(w, r).
func (f HandlerFunc) ServeHTTP(w ResponseWriter, r *Request) {
    f(w, r)
}
```  
原理，是通过强制类型转换将一个函数转化成了HandlerFunc类型，该类型实现了ServeHTTP, 从而实现了Handler接口。  
简单的handler 可以直接使用 HandlerFunc  

2. http.Handle(pattern string, handler Handler), 注册一个自定义的Handler, 用户需要自己实现Handler接口。  
自定义的Handler 可以实现更复杂的业务，可以在接收者保存业务数据，在 ServHTTP 方法自定义自己的业务逻辑。  
```
func Handle(pattern string, handler Handler) { 
    DefaultServeMux.Handle(pattern, handler) 
}
```

3. DefaultServeMux.Handle 注册逻辑:
```
func (mux *ServeMux) Handle(pattern string, handler Handler) {
mux.mu.Lock()
defer mux.mu.Unlock()

if pattern == "" {
	panic("http: invalid pattern")
}
if handler == nil {
	panic("http: nil handler")
}
if _, exist := mux.m[pattern]; exist { // 如果注册一个已注册的处理器，将panic
	panic("http: multiple registrations for " + pattern)
}

if mux.m == nil {
	mux.m = make(map[string]muxEntry)
}
e := muxEntry{h: handler, pattern: pattern} // 注册
mux.m[pattern] = e
if pattern[len(pattern)-1] == '/' {
	mux.es = appendSorted(mux.es, e) // 以斜杠结尾的pattern将存入es切片并按pattern长度降序排列
}

if pattern[0] != '/' {
	mux.hosts = true // 不以"/"开头的模式将视作存在hosts
}
}
```
