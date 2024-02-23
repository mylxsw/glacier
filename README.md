# Glaicer Framework

Glacier 是一款支持依赖注入的模块化的应用开发框架，它以 [go-ioc](https://github.com/mylxsw/go-ioc) 依赖注入容器核心，为 Go 应用开发解决了依赖传递和模块化的问题。

- [特性](#特性)
- [使用](#使用)
- [执行流程](#执行流程)
- [核心概念](#核心概念)
	- [依赖注入](#依赖注入)
		- [Binder](#binder)
		- [Resolver](#resolver)
	- [Provider](#provider)
		- [ProviderBoot](#providerboot)
		- [DaemonProvider](#daemonprovider)
		- [ProviderAggregate](#provideraggregate)
		- [Service](#service)
		- [ModuleLoadPolicy](#moduleloadpolicy)
		- [Priority](#priority)
- [Web 框架](#web-框架)
	- [Usage](#usage)
	- [控制器](#控制器)
- [事件管理](#事件管理)
	- [本地内存作为事件存储后端](#本地内存作为事件存储后端)
	- [Redis 作为事件存储后端](#redis-作为事件存储后端)
- [定时任务](#定时任务)
- [日志](#日志)
- [Eloquent ORM](#eloquent-orm)
- [平滑退出](#平滑退出)
- [第三方框架集成](#第三方框架集成)
- [示例项目](#示例项目)

## 特性

- **依赖注入**：通过依赖注入的方式来管理对象的依赖，支持单例、原型对象创建
- **模块化**：通过 Provider 特性，轻松实现应用的模块化
- **内置 Web 开发支持**：Glacier 内置了对 Web 应用开发的支持，提供了功能丰富的 API 简化 web 开发

API 文档看这里：https://pkg.go.dev/github.com/mylxsw/glacier 。

## 使用

创建一个新的项目，使用下面的命令安装 Glacier 开发框架

```bash
go get github.com/mylxsw/glacier
```

为了简化应用的创建过程，我们一般可以通过 starter 模板来创建应用

```go
import "github.com/mylxsw/glacier/starter/app"
...

// 方法一：快捷启动应用
app.MustStart("1.0", 3, func(app *app.App) error {
	// 这里完成应用的初始化
	// ...
	return nil
})

// 方法二： 分步骤启动应用
ins := app.Create("1.0", 3)
// 应用初始化
// ...
app.MustRun(ins)
```

示例:

```go
app.MustStart("1.0", 3, func(ins *app.App) error {
	ins.AddStringFlag("listen", ":8080", "http listen address")
	
	ins.Provider(web.Provider(
		listener.FlagContext("listen"),
		web.SetRouteHandlerOption(func(cc infra.Resolver, router web.Router, mw web.RequestMiddleware) {
			router.Get("/", func(ctx web.Context) web.Response {
				return ctx.JSON(web.M{})
			})
		}),
	))
	return nil
})
```

> 代码示例可以参考当前项目的 [example](https://github.com/mylxsw/glacier/tree/main/example) 目录。

## 执行流程

![执行流程](./arch.svg)

## 核心概念

### 依赖注入

Glacier 框架充分利用了 [go-ioc](https://github.com/mylxsw/go-ioc) 提供的依赖注入能力，为应用提供了功能强大的依赖注入特性。

在使用依赖注入特性时，首先要理解以下两个接口的作用

- `infra.Binder` 该接口用于对象创建实例方法的绑定，简单说就是向 `go-ioc` 容器注册对象的创建方法
- `infra.Resolver` 该接口用于对象的实例化，获取对象实例

无论是 `Binder` 还是 `Resolver`，都会有一个 `interface{}` 类型的参数，它的类型为符合一定规则的函数，后面在 `Binder` 和 `Resolver` 部分将会详细说明。

#### Binder

`infra.Binder` 是一个对象定义接口，用于将实例的创建方法绑定到依赖注入容器，提供了以下常用方法

- `Prototype(initialize interface{}) error` 原型绑定，每次访问绑定的实例都会基于 `initialize` 函数重新创建新的实例
- `Singleton(initialize interface{}) error` 单例绑定，每次访问绑定的实例都是同一个，只会在第一次访问的时候创建初始实例
- `BindValue(key string, value interface{}) error` 将一个具体的值绑定到 `key`

`Prototype` 和 `Singleton` 方法参数 `initialize interface{}` 支持以下两种形式

- 形式1：`func(依赖参数列表...) (绑定类型定义, error)`

  ```go
  // 这里使用单例方法定义了数据库连接对象的创建方法
  binder.Singleton(func(conf *Config) (*sql.DB, error) {
  	return sql.Open("mysql", conf.MySQLURI)
  })

  binder.Singleton(func(c infra.FlagContext) *Config {
		...
		return &Config{
			Listen:   c.String("listen"),
			MySQLURI: c.String("mysql_uri"),
			APIToken: c.String("api_token"),
			...		
		}
	})
  ```

- 形式2：`func(注入参数列表...) 绑定类型定义`

	```go
	binder.Singleton(func() UserRepo { return &userRepoImpl{} })
	binder.Singleton(func(db *sql.DB) UserRepo { 
		// 这里我们创建的 userRepoImpl 对象，依赖 sql.DB 对象，只需要在函数
		// 参数中，将依赖列举出来，容器会自动完成这些对象的创建
		return &userRepoImpl{db: db} 
	})
	```

#### Resolver

`infra.Resolver` 是对象实例化接口，通过依赖注入的方式获取实例，提供了以下常用方法

- `Resolve(callback interface{}) error` 执行 callback 函数，自动为 callback 函数提供所需参数
- `Call(callback interface{}) ([]interface{}, error)` 执行 callback 函数，自动为 callback 函数提供所需参数，支持返回值，返回参数为 `Call` 的第一个数组参数
- `AutoWire(object interface{}) error` 自动对结构体对象进行依赖注入，object 必须是结构体对象的指针。自动注入字段（公开和私有均支持）需要添加 `autowire` tag，支持以下两种
	- autowire:"@" 根据字段的类型来注入
	- autowire:"自定义key" 根据自定义的key来注入（查找名为 key 的绑定）
- `Get(key interface{}) (interface{}, error)` 直接通过 key 来查找对应的对象实例

```go
// Resolve
resolver.Resolve(func(db *sql.DB) {...})
err := resolver.Resolve(func(db *sql.DB) error {...})

// Call
resolver.Call(func(userRepo UserRepo) {...})
// Call 带有返回值
// 这里的 err 是依赖注入过程中的错误，比如依赖对象创建失败
// results 是一个类型为 []interface{} 的数组，数组中按次序包含了 callback 函数的返回值，以下面的代码为例，其中
// results[0] - string
// results[1] - error
results, err := resolver.Call(func(userRepo UserRepo) (string, error) {...})
// 由于每个返回值都是 interface{} 类型，因此在使用时需要执行类型断言，将其转换为具体的类型再使用
returnValue := results[0].(string)
returnErr := results[1].(error)

// AutoWire
// 假设我们有一个 UserRepo，创建该结构体时需要数据库的连接实例
type UserRepo struct {
  db *sql.DB `autowire:"@"`
}

userRepo := UserRepo{}
resolver.AutoWire(&userRepo)

// 现在 userRepo 中的 db 参数已经自动被设置为了数据库连接对象，可以继续执行后续的操作了
```

### Provider

在 Glacier 应用开发框架中，Provider 是应用模块化的核心，每个独立的功能模块通过 Provider 完成实例初始化，每个 Provider 都需要实现 `infra.Provider` 接口。 在每个功能模块中，我们通常会创建一个名为 provider.go 的文件，在该文件中创建一个 provider 实现

```go
type Provider struct{}

func (Provider) Register(binder infra.Binder) {
	... // 这里可以使用 binder 向 IOC 容器注册当前模块中的实例创建方法
}
```

**Provider** 接口只有一个必须实现的方法 `Register(binder infra.Binder)`，该方法用于注册当前模块的对象到 IOC 容器中，实现依赖注入的支持。

例如，我们实现一个基于数据库的用户管理模块 `repo`，该模块包含两个方法

```go
package repo

type UserRepo struct {
  db *sql.DB
}

func (repo *UserRepo) Login(username, password string) (*User, error) {...}
func (repo *UserRepo) GetUser(username string) (*User, error) {...}
```

为了使该模块能够正常工作，我们需要在创建 `UserRepo` 时，提供 `db` 参数，在 Glacier 中，我们可以这样实现

```go
package repo

type Provider struct {}

func (Provider) Register(binder infra.Binder) {
  binder.Singleton(func(db *sql.DB) *UserRepo { return &UserRepo {db: db} })
}
```

在我们的应用创建时，使用 `ins.Provider` 方法注册该模块

```go
ins := app.Default("1.0")
...
ins.MustSingleton(func() (*sql.DB, error) {
	return sql.Open("mysql", "user:pwd@tcp(ip:3306)/dbname")
})
// 在这里加载模块的 Provider
ins.Provider(repo.Provider{})
...
app.MustRun(ins)
```

#### ProviderBoot

在我们使用 Provider 时，默认只需要实现一个接口方法 `Register(binder infra.Binder)` 即可，该方法用于将模块的实例创建方法注册到 Glacier 框架的 IOC 容器中。

在 Glaicer 中，还提供了一个 `ProviderBoot` 接口，该接口包含一个 `Boot(resolver Resolver)` 方法，实现该方法的模块，可以在 Glacier 框架启动过程中执行一些模块自有的业务逻辑，该方法在所有的模块全部加载完毕后执行（所有的模块的 `Register` 方法都已经执行完毕），因此，系统中所有的对象都是可用的。

`Boot(resolver Resolver)` 方法中适合执行一些在应用启动过程中所必须完成的一次性任务，任务应该尽快完成，以避免影响应用的启动。

```go
type Provider struct{}

func (Provider) Register(binder infra.Binder) {
	binder.MustSingleton(func(conf *configs.Config) *grpc.Server { return ... })
}

func (Provider) Boot(resolver infra.Resolver) {
	resolver.MustResolve(func(serv *grpc.Server) {
		protocol.RegisterMessageServer(serv, NewEventService())
		protocol.RegisterHeartbeatServer(serv, NewHeartbeatService())
	})
}
```

#### DaemonProvider

模块 Provider 的 `Boot` 方法是阻塞执行的，通常用于执行一些在应用启动时需要执行的一些初始化任务，在一个应用中，所有的 Provider 的 `Boot` 方法是串行执行的。

而 `DaemonProvider` 接口则为模块提供了异步执行的能力，模块的 `Daemon(ctx context.Context, resolver infra.Resolver)` 方法是异步执行的，我们可以在这里执行创建 web 服务器等操作。

```go
func (Provider) Daemon(_ context.Context, app infra.Resolver) {
	app.MustResolve(func(
		serv *grpc.Server, conf *configs.Config, gf graceful.Graceful,
	) {
		listener, err := net.Listen("tcp", conf.GRPCListen)
		...
		gf.AddShutdownHandler(serv.GracefulStop)
		...
		if err := serv.Serve(listener); err != nil {
			log.Errorf("GRPC Server has been stopped: %v", err)
		}
	})
}
```

#### ProviderAggregate

ProviderAggregate 接口为应用提供了一种能够聚合其它模块 Provider 的能力，在 `Aggregate() []Provider`方法中，我们可以定义多个我们当前模块所依赖的其它模块，在 Glacier 框架启动过程中，会优先加载这里定义的依赖模块，然后再加载我们的当前模块。

我们可以通过 `ProviderAggregate` 来创建我们自己的模块， `Aggregates() []infra.Provider` 方法中返回依赖的子模块，框架会先初始化子模块，然后再初始化当前模块。

```go
// 创建自定义模块，初始化了 Glacier 框架内置的 Web 框架
type Provider struct{}

func (Provider) Aggregates() []infra.Provider {
	return []infra.Provider{
		// 加载了 web 模块，为应用提供 web 开发支持
		web.Provider(
			listener.FlagContext("listen"), // 从命令行参数 listen 获取监听端口
			web.SetRouteHandlerOption(s.routes), // 设置路由规则
			web.SetExceptionHandlerOption(func(ctx web.Context, err interface{}) web.Response {
				log.Errorf("error: %v, call stack: %s", err, debug.Stack())
				return nil
			}), // Web 异常处理
		),
	}
}

func (Provider) routes(cc infra.Resolver, router web.Router, mw web.RequestMiddleware) {
	router.Controllers(
		"/api",
		// 这里添加控制器
		controller.NewWelcomeController(cc),
		controller.NewUserController(cc),
	)
}

func (Provider) Register(app infra.Binder) {}
```

#### Service

在 Glacier 框架中，Service 代表了一个后台模块，Service 会在框架生命周期中持续运行。要实现一个 Service，需要实现 `infra.Service` 接口，该接口只包含一个方法

- `Start() error` 用于启动 Service

除了 `Start` 方法之外，还支持以下控制方法，不过它们都是可选的

- `Init(resolver Resolver) error` 用于 Service 的初始化，注入依赖等
- `Stop()` 触发 Service 的停止运行
- `Reload()` 触发 Service 的重新加载

以下为一个示例

```go
type DemoService struct {
	resolver infra.Resolver
	stopped chan interface{}
}

// Init 可选方法，用于在 Service 启动之前初始化一些参数
func (s *DemoService) Init(resolver infra.Resolver) error {
	s.resolver = resolver
	s.stopped = make(chan interface{})
	return nil
}

// Start 用于 Service 的启动
func (s *DemoService) Start() error {
	for {
		select {
		case <-s.stopped:
			return nil
		default:
			... // 业务代码
		}
	}
}

// Stop 和 Reload 都是可选方法
func (s *DemoService) Stop() { s.stopped <- struct{}{} }
func (s *DemoService) Reload() { ... }

```

在我们的应用创建时，使用 `app.Service` 方法注册 Service

```go
ins := app.Create("1.0")
...
ins.Service(&service.DemoService{})
...
app.MustRun(ins)
```

#### ModuleLoadPolicy

**Provider** 和 **Service** 支持按需加载，要使用此功能，只需要让 **Provider** 和 **Service** 实现 **ShouldLoad(...) bool** 方法。`ShouldLoad` 方法用于控制 **Provider** 和 **Service** 是否加载，支持以下几种形式

- `func (Provider) ShouldLoad(...依赖) bool`
- `func (Provider) ShouldLoad(...依赖) (bool, error)`

示例

```go
type Provider struct{}
func (Provider) Register(binder infra.Binder) {...}

// 只有当 config.AuthType == ldap 的时候才会加载当前 Provider
func (Provider) ShouldLoad(config *config.Config) bool {
	return str.InIgnoreCase(config.AuthType, []string{"ldap"})
}
```

> 注意：`ShouldLoad` 方法在执行时，`Provider` 并没有完成 `Register` 方法的执行，因此，在 `ShouldLoad` 方法的参数列表中，只能使用在应用创建时全局注入的对象实例。
> 
> ```go
> ins := app.Create("1.0")
> ...
> ins.Singleton(func(c infra.FlagContext) *config.Config { return ... })
> ...
> app.MustRun(ins)
> ```

#### Priority

实现 `infra.Priority` 接口的 **Provider **、 **Service **，会按照 `Priority()` 方法的返回值大小依次加载，值越大，加载顺序越靠后，默认的优先级为 `1000`。

```go
type Provider struct {}
func (Provider) Register(binder infra.Binder) {...}

func (Provider) Priority() int {
	return 10
}
```

## Web 框架

Glacier 是一个应用框架，为了方便 Web 开发，也内置了一个灵活的 Web 应用开发框架。

### Usage

Glaicer Web 在 Glacier 框架中是一个内置的 **DaemonProvider**，与其它的模块并无不同。我们通过 `web.Provider(builder infra.ListenerBuilder, options ...Option) infra.DaemonProvider` 方法创建 Web 模块。

参数 `builder` 用于创建 Web 服务的 listener（用于告知 Web 框架如何监听端口），在 Glaicer 中，有以下几种方式来创建 listener：

- `listener.Default(listenAddr string) infra.ListenerBuilder` 该构建器使用固定的 listenAddr 来创建 listener
- `listener.FlagContext(flagName string) infra.ListenerBuilder` 该构建器根据命令行选项 flagName 来获取要监听的地址，以此来创建 listener 
- `listener.Exist(listener net.Listener) infra.ListenerBuilder` 该构建器使用应存在的 listener 来创建

参数 `options` 用于配置 web 服务的行为，包含以下几种常用的配置

- `web.SetRouteHandlerOption(h RouteHandler) Option` 设置路由注册函数，在该函数中注册 API 路由规则
- `web.SetExceptionHandlerOption(h ExceptionHandler) Option` 设置请求异常处理器
- `web.SetIgnoreLastSlashOption(ignore bool) Option` 设置路由规则忽略最后的 `/`，默认是不忽略的
- `web.SetMuxRouteHandlerOption(h MuxRouteHandler) Option` 设置底层的 gorilla Mux 对象，用于对底层的 Gorilla 框架进行直接控制
- `web.SetHttpWriteTimeoutOption(t time.Duration) Option` 设置 HTTP 写超时时间
- `web.SetHttpReadTimeoutOption(t time.Duration) Option` 设置 HTTP 读超时时间
- `web.SetHttpIdleTimeoutOption(t time.Duration) Option` 设置 HTTP 空闲超时时间
- `web.SetMultipartFormMaxMemoryOption(max int64)` 设置表单解析能够使用的最大内存
- `web.SetTempFileOption(tempDir, tempFilePattern string) Option` 设置临时文件存储规则
- `web.SetInitHandlerOption(h InitHandler) Option` 初始化阶段，web 应用对象还没有创建，在这里可以更新 web 配置
- `web.SetListenerHandlerOption(h ListenerHandler) Option` 服务初始化阶段，web 服务对象已经创建，此时不能再更新 web 配置了

最简单的使用 Web 模块的方式是直接创建 Provider，

```go
// Password 该结构体时 /complex 接口的返回值定义
type Password struct {
	Password string `json:"password"`
}

// Glacier 框架初始化
ins := app.Default("1.0")
...
// 添加命令行参数 listen，指定默认监听端口 :8080
ins.AddStringFlag("listen", ":8080", "http listen address")
...
ins.Provider(web.Provider(
	// 使用命令行 flag 的 listener builder
	listener.FlagContext("listen"), 
	// 设置路由规则
	web.SetRouteHandlerOption(func(resolver infra.Resolver, r web.Router, mw web.RequestMiddleware) {
		...
		r.Get("/simple", func(ctx web.Context, gen *password.Generator) web.Response {
			...
			return ctx.JSON(web.M{"password": pass})
		})
		
		r.Get("/complex", func(ctx web.Context, gen *password.Generator) Password {...})
	}),
))

app.MustRun(ins)
```

更好的方式是使用模块化，编写一个独立的 Provider 

```go
type Provider struct{}

// Aggregates 实现 infra.ProviderAggregate 接口
func (Provider) Aggregates() []infra.Provider {
	return []infra.Provider{
		web.Provider(
			confListenerBuilder{},
			web.SetRouteHandlerOption(routes),
			web.SetMuxRouteHandlerOption(muxRoutes),
			web.SetExceptionHandlerOption(exceptionHandler),
		),
	}
}

// Register 实现 infra.Provider 接口
func (Provider) Register(binder infra.Binder) {}

// exceptionHandler 异常处理器
func exceptionHandler(ctx web.Context, err interface{}) web.Response {
	return ctx.JSONWithCode(web.M{"error": fmt.Sprintf("%v", err)}, http.StatusInternalServerError)
}

// routes 注册路由规则
func routes(resolver infra.Resolver, router web.Router, mw web.RequestMiddleware) {
	mws := make([]web.HandlerDecorator, 0)
	// 添加 web 中间件
	mws = append(mws,
		mw.AccessLog(log.Module("api")),
		mw.CORS("*"),
	)

	// 注册控制器，所有的控制器 API 都以 `/api` 作为接口前缀
	router.WithMiddleware(mws...).Controllers(
		"/api",
		controller.NewServerController(resolver),
		controller.NewClientController(resolver),
	)
}

func muxRoutes(resolver infra.Resolver, router *mux.Router) {
	resolver.MustResolve(func() {
		// 添加 prometheus metrics 支持
		router.PathPrefix("/metrics").Handler(promhttp.Handler())
		// 添加健康检查接口支持
		router.PathPrefix("/health").Handler(HealthCheck{})
	})
}

// 创建自定义的 listener 构建器，从配置对象中读取 listen 地址
type confListenerBuilder struct{}

func (l confListenerBuilder) Build(resolver infra.Resolver) (net.Listener, error) {
	return listener.Default(resolver.MustGet((*config.Server)(nil)).(*config.Server).HTTPListen).Build(resolver)
}
```

### 控制器

控制器必须实现 `web.Controller` 接口，该接口只有一个方法

- `Register(router Router)` 用于注册当前控制器的路由规则

```go
type UserController struct {...}

// NewUserController 控制器创建方法，返回 web.Controller 接口
func NewUserController() web.Controller { return &UserController{...} }

// Register 注册当前控制器关联的路由规则
func (ctl UserController) Register(router web.Router) {
	router.Group("/users/", func(router web.Router) {
		router.Get("/", u.Users).Name("users:all")
		router.Post("/", u.Add)
		router.Post("/{id}/", u.Update)
		router.Get("/{id}/", u.User).Name("users:one")
		router.Delete("/{id}/", u.Delete).Name("users:delete")
	})

	router.Group("/users-helper/", func(router web.Router) {
		router.Get("/names/", u.UserNames)
	})
}

// 读取 JSON 请求参数，直接返回实例，会以 json 的形式返回给客户端
func (ctl UserController) Add(ctx web.Context, userRepo repository.UserRepo) (*repository.User, error) {
	var userForm *UserForm
	if err := ctx.Unmarshal(&userForm); err != nil {
		return nil, web.WrapJSONError(fmt.Errorf("invalid request: %v", err), http.StatusUnprocessableEntity)
	}
	ctx.Validate(userForm, true)
	...
	return ...
}

// 直接返回错误，如果 error 不为空，则返回错误给客户端
func (ctl UserController) Delete(ctx web.Context, userRepo repository.UserRepo) error {
	userID := ctx.PathVar("id")
	...
	return userRepo.DeleteID(userID)
}

// 返回 web.Response，可以使用多种格式返回，如 ctx.Nil, ctx.API, ctx.JSON, ctx.JSONWithCode, ctx.JSONError, ctx.YAML, ctx.Raw, ctx.HTML, ctx.HTMLWithCode, ctx.Error 等
func (u UserController) Users(ctx web.Context, userRepo repository.UserRepo, roleRepo repository.RoleRepo) web.Response {
	page := ctx.IntInput("page", 1)
	perPage := ctx.IntInput("per_page", 10)
	...
	return ctx.JSON(web.M{
		"users": users,
		"next":  next,
		"search": web.M{
			"name":  name,
			"phone": phone,
			"email": email,
		},
	})
}
```

使用 `web.Router` 实例的 `Controllers` 方法注册控制器。

```go
// routes 注册路由规则
func routes(resolver infra.Resolver, router web.Router, mw web.RequestMiddleware) {
	mws := make([]web.HandlerDecorator, 0)
	// 添加 web 中间件
	mws = append(mws,
		mw.AccessLog(log.Module("api")),
		mw.CORS("*"),
	)

	// 注册控制器，所有的控制器 API 都以 `/api` 作为接口前缀
	router.WithMiddleware(mws...).Controllers(
		"/api",
		controller.NewUserController(),
	)
}
```


## 事件管理

Glacier 框架提供了一个简单的事件管理模块，可以用于发布和监听应用运行中的事件，进行相应的业务处理。

通过 `event.Provider(handler func(resolver infra.Resolver, listener Listener), options ...Option) infra.Provider ` 来初始化事件管理器。

```go
ins.Provider(event.Provider(
  func(cc infra.Resolver, listener event.Listener) {
    listener.Listen(func(event CronEvent) {
      log.Debug("a new cron task executed")
      // 执行监听到定时任务执行事件后要触发的操作
    })
  },
  // 设置事件管理器选项
  event.SetStoreOption(func(cc infra.Resolver) event.Store {
    // 设置使用默认的内存事件存储
    return event.NewMemoryEventStore(true, 100)
  }),
))
```

发布事件时，使用 Glacier 框架的依赖注入能力，获取 `event.Publisher` 接口实现

```go
ins.Async(func(publisher event.Publisher) {
  for i := 0; i < 10; i++ {
    publisher.Publish(CronEvent{GoroutineID: uint64(i)})
  }
})
```

### 本地内存作为事件存储后端

Glacier 内置了基于内存的事件存储后端，说有事件的监听器都是同步执行的。

```go
// 设置事件管理器选项
event.SetStoreOption(func(cc infra.Resolver) event.Store {
	// 设置使用默认的内存事件存储
	return event.NewMemoryEventStore(true, 100)
})
```

### Redis 作为事件存储后端

使用内存作为事件存储后端时，当应用异常退出的时候，可能会存在事件的丢失，你可以使用这个基于 Redis 的事件存储后端 [redis-event-store](https://github.com/mylxsw/redis-event-store) 来获得事件的持久化支持。

## 定时任务

Glacier 提供了内置的定时任务支持，使用 `scheduler.Provider` 来实现。

```go
type Provider struct{}
func (Provider) Register(binder infra.Binder) {...}

func (Provider) Aggregates() []infra.Provider {
	return []infra.Provider{
		// 加载 scheduler 定时任务模块
		scheduler.Provider(
			func(resolver infra.Resolver, creator scheduler.JobCreator) {
				// 添加一个名为 test-job 的任务，每隔 10s 执行一次
				_ = cr.Add("test-job", "@every 10s", TestJob)
				// 添加一个名称为 test-timeout-job 的任务，每隔 5s 执行一次
				// 通过 AddAndRunOnServerReady 添加的任务会在服务启动时先执行一次
				_ = creator.AddAndRunOnServerReady(
					"test-timeout-job", 
					"@every 5s",
					// 使用 scheduler.WithoutOverlap 包装的函数，当前一次调度还没有执行完毕，本次调度的时间已到，本次调度将会被取消
					scheduler.WithoutOverlap(TestTimeoutJob).SkipCallback(func() { 
						... // 当前一个任务还没有执行完毕时，当前任务会被跳过，跳过时会触发该函数的执行
					}),
				)
			},
		),
	}
}
```

`scheduler.Provider` 支持分布式锁，通过 `SetLockManagerOption` 选项可以指定分布式锁的实现，以满足任务在一组服务器中只会被触发一次的逻辑。

```go
scheduler.Provider(
	func(resolver infra.Resolver, creator scheduler.JobCreator) {...},
	// 设置分布式锁
	scheduler.SetLockManagerOption(func(resolver infra.Resolver) scheduler.LockManagerBuilder {
		// get redis instance
		redisClient := resolver.MustGet(&redis.Client{}).(*redis.Client)
		return func(name string) scheduler.LockManager {
			// create redis lock
			return redisLock.New(redisClient, name, 10*time.Minute)
		}
	}),
)
```

> 注意：Glacier 框架没有内置分布式锁的实现，在 [mylxsw/distribute-locks](https://github.com/mylxsw/distribute-locks) 实现了一个简单的基于 Redis 的分布式锁实现，可以参考使用。

## 日志

在 Glacier 中，默认使用 [asteria](https://github.com/mylxsw/asteria) 作为日志框架，asteria 是一款功能强大、灵活的结构化日志框架，支持多种日志输出格式以及输出方式，支持为日志信息添加上下文信息。

最简单的方式是通过 `log.SetDefaultLogger(logger infra.Logger)` 方法为 Glacier 框架设置默认的日志处理器，

```go
// import "github.com/mylxsw/glacier/log"

// 默认设置，使用 asteria 日志框架
// import asteria "github.com/mylxsw/asteria/log"
log.SetDefaultLogger(asteria.Module("glacier"))
// 使用标准库中的日志包，Glacier 对标准库日志包进行了简单封装
log.SetDefaultLogger(log.StdLogger())
```

当然，如果使用了 starter 模板项目创建的应用，也可以使用 `WithLogger(logger infra.Logger)` 方法来设置日志处理器。

```go
ins := app.Default("1.0")
...
// 设置使用标准库日志包，不输出 DEBUG 日志
ins.WithLogger(log.StdLogger(log.DEBUG))
...
```

除了默认的 `asteria` 日志库以及 Glacier 自带的 `StdLogger` 之外，还可以使用其它第三方的日志包，只需要简单的封装，实现 `infra.Logger` 接口即可。

```go
type Logger interface {
	Debug(v ...interface{})
	Debugf(format string, v ...interface{})
	Info(v ...interface{})
	Infof(format string, v ...interface{})
	Error(v ...interface{})
	Errorf(format string, v ...interface{})
	Warning(v ...interface{})
	Warningf(format string, v ...interface{})
	// Critical 关键性错误，遇到该日志输出时，应用直接退出
	Critical(v ...interface{})
	// Criticalf 关键性错误，遇到该日志输出时，应用直接退出
	Criticalf(format string, v ...interface{})
}
```

## Eloquent ORM

Eloquent ORM 是为 Go 开发的一款数据库 ORM 框架，它的设计灵感来源于著名的 PHP 开发框架 Laravel，支持 MySQL 等数据库。

项目地址为 [mylxsw/eloquent](https://github.com/mylxsw/eloquent)，可以配合 Glacier 框架使用。

## 平滑退出

Glacier 支持平滑退出，当我们按下键盘的 `Ctrl+C` 时（接收到 SIGINT， SIGTERM, Interrupt 等信号）， Glacier 将会接收到关闭的信号，然后触发应用的关闭行为。默认情况下，我们的应用会立即退出，我们可以通过 starter 模板创建的应用上启用平滑支持选项 `WithShutdownTimeoutFlagSupport(timeout time.Duration)` 来设置默认的平滑退出时间

```go
ins := app.Create("1.0")
ins.WithShutdownTimeoutFlagSupport(5 * time.Second)
...

// Provider 中获取 `gf.Graceful` 实例，注册关闭时的处理函数
resolver.MustResolve(func(gf graceful.Graceful) {
	gf.AddShutdownHandler(func() {
		...
	})
})
```

## 第三方框架集成

- [giris](https://github.com/mylxsw/giris): [Iris Web Framework](https://www.iris-go.com/) 适配

## 示例项目

- [Example](https://github.com/mylxsw/glacier/tree/main/example) 使用示例
- [WebDAV Server](https://github.com/mylxsw/webdav-server) 一款支持 LDAP 作为用户数据库的 WebDAV 服务器
- [Adanos Alert](https://github.com/mylxsw/adanos-alert) 一个功能强大的开源告警平台，通过事件聚合机制，为监控系统提供钉钉、邮件、HTTP、JIRA、语音电话等告警方式的支持
- [Healthcheck](https://github.com/mylxsw/healthcheck) 为应用服务提供健康检查告警支持
- [Sync](https://github.com/mylxsw/sync) 跨服务器文件同步服务
- [Tech Share](https://github.com/mylxsw/tech-share) 一个用于中小型团队内部技术分享管理的 Web 应用
- [Universal Exporter](https://github.com/mylxsw/universal-exporter) 一个通用的 Prometheus 维度工具，目前支持从数据库中查询生成 Metric 数据
- [Graphviz Server](https://github.com/mylxsw/graphviz-server) 一个 Web 服务，封装了对 Graphviz 的接口调用，实现通过 Web API 的方式生成 Graphviz 图形
- [MySQL Guard](https://github.com/mylxsw/mysql-guard) 用于 MySQL 长事务检测杀死和死锁告警
- [Password Server](https://github.com/mylxsw/password-server) 一个生成随机密码的简单 web 服务器