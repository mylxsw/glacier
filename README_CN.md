# Glacier Framework

Glacier 是一个基于依赖注入的模块化 Go 应用开发框架，以 [go-ioc](https://github.com/mylxsw/go-ioc) 容器为核心，帮助你构建结构清晰、易于维护的 Go 应用。

```
go get github.com/mylxsw/glacier
```

API 文档：https://pkg.go.dev/github.com/mylxsw/glacier

## 目录

- [快速开始](#快速开始)
- [核心概念](#核心概念)
  - [依赖注入](#依赖注入)
  - [Provider（模块）](#provider模块)
  - [Service（后台服务）](#service后台服务)
- [Web 开发](#web-开发)
  - [基本用法](#基本用法)
  - [控制器](#控制器)
  - [中间件](#中间件)
  - [Handler 返回值](#handler-返回值)
  - [Web 配置项](#web-配置项)
- [事件系统](#事件系统)
- [定时任务](#定时任务)
- [日志](#日志)
- [平滑退出](#平滑退出)
- [完整示例](#完整示例)
- [相关项目](#相关项目)

## 快速开始

下面是一个最小可运行的 Glacier 应用——一个返回 JSON 的 HTTP 服务器：

```go
package main

import (
    "github.com/mylxsw/glacier/listener"
    "github.com/mylxsw/glacier/starter/app"
    "github.com/mylxsw/glacier/infra"
    "github.com/mylxsw/glacier/web"
)

func main() {
    app.MustStart("1.0", 3, func(ins *app.App) error {
        // 添加命令行参数 --listen，默认 :8080
        ins.AddStringFlag("listen", ":8080", "HTTP 监听地址")

        // 注册 Web 模块
        ins.Provider(web.Provider(
            listener.FlagContext("listen"),
            web.SetRouteHandlerOption(func(cc infra.Resolver, router web.Router, mw web.RequestMiddleware) {
                router.Get("/hello", func(ctx web.Context) web.Response {
                    name := ctx.InputWithDefault("name", "World")
                    return ctx.JSON(web.M{"message": "Hello, " + name})
                })
            }),
        ))

        return nil
    })
}
```

运行后访问 `http://localhost:8080/hello?name=Glacier`，即可看到 JSON 响应。

> `app.MustStart` 的三个参数分别是：版本号、异步任务 Runner 数量、初始化函数。

### 两种启动方式

```go
// 方式一：一步到位
app.MustStart("1.0", 3, func(ins *app.App) error {
    // 初始化配置...
    return nil
})

// 方式二：分步创建，适合需要更精细控制的场景
ins := app.Create("1.0", 3)
// 配置 ins...
app.MustRun(ins)
```

## 核心概念

Glacier 围绕三个核心概念构建：**依赖注入**、**Provider** 和 **Service**。理解了它们，就掌握了整个框架。

### 依赖注入

依赖注入是 Glacier 的基础。框架通过 IoC（控制反转）容器自动管理对象的创建和依赖关系，你只需要告诉容器"如何创建对象"，使用时容器会自动把依赖组装好。

核心的两个接口：

| 接口 | 作用 | 类比 |
|------|------|------|
| `infra.Binder` | 注册对象的创建方法 | "告诉工厂怎么造东西" |
| `infra.Resolver` | 从容器中获取对象实例 | "从工厂拿东西" |

#### Binder：注册对象

```go
// Singleton：单例，整个应用生命周期中只创建一次
binder.Singleton(func() *Database {
    return &Database{DSN: "localhost:3306"}
})

// 支持自动注入依赖：参数由容器自动提供
binder.Singleton(func(conf *Config) (*sql.DB, error) {
    return sql.Open("mysql", conf.MySQLURI)
})

// Prototype：原型，每次获取都创建新实例
binder.Prototype(func() *RequestLogger {
    return &RequestLogger{CreatedAt: time.Now()}
})

// BindValue：绑定一个具体的值到指定 key
binder.BindValue("app_name", "MyApp")
```

#### Resolver：获取对象

```go
// Resolve：执行函数，参数由容器自动注入
resolver.Resolve(func(db *sql.DB) {
    db.Query("SELECT 1")
})

// Call：与 Resolve 类似，但支持获取返回值
results, err := resolver.Call(func(db *sql.DB) (string, error) {
    return "ok", nil
})

// AutoWire：自动注入结构体字段（需添加 autowire tag）
type UserService struct {
    DB     *sql.DB    `autowire:"@"`    // 按类型注入
    Config *Config    `autowire:"@"`    // 按类型注入
}
svc := &UserService{}
resolver.AutoWire(svc)
// 现在 svc.DB 和 svc.Config 已被自动赋值
```

#### 一个完整的依赖注入示例

```go
app.MustStart("1.0", 3, func(ins *app.App) error {
    // 第一步：注册配置对象的创建方法
    ins.MustSingleton(func(c infra.FlagContext) *Config {
        return &Config{
            DBAddr: c.String("db-addr"),
        }
    })

    // 第二步：注册数据库连接（自动依赖上面的 Config）
    ins.MustSingleton(func(conf *Config) (*sql.DB, error) {
        return sql.Open("mysql", conf.DBAddr)
    })

    // 第三步：注册 UserRepo（自动依赖上面的 sql.DB）
    ins.MustSingleton(func(db *sql.DB) *UserRepo {
        return &UserRepo{db: db}
    })

    // 使用时，容器会自动解析整条依赖链：
    // Config -> sql.DB -> UserRepo
    return nil
})
```

### Provider（模块）

Provider 是 Glacier 中实现模块化的核心机制。每个独立的功能模块封装为一个 Provider，通过实现 `infra.Provider` 接口完成注册。

#### 基本 Provider

最简单的 Provider 只需实现 `Register` 方法：

```go
package user

type Provider struct{}

func (Provider) Register(binder infra.Binder) {
    binder.Singleton(func(db *sql.DB) *UserRepo {
        return &UserRepo{db: db}
    })
}
```

注册到应用中：

```go
ins.Provider(user.Provider{})
```

#### Provider 的扩展接口

Provider 可以通过实现额外的接口来获得更多能力：

| 接口 | 方法 | 用途 |
|------|------|------|
| `ProviderBoot` | `Boot(resolver Resolver)` | 所有模块注册完毕后执行初始化逻辑 |
| `DaemonProvider` | `Daemon(ctx, resolver)` | 异步运行的后台任务（如 HTTP 服务器） |
| `ProviderAggregate` | `Aggregates() []Provider` | 声明依赖的子模块，框架会先加载它们 |

**ProviderBoot** — 启动时执行初始化：

```go
func (Provider) Boot(resolver infra.Resolver) {
    // 此时所有模块已注册完毕，可以安全地使用任何依赖
    resolver.MustResolve(func(db *sql.DB) {
        // 执行数据库迁移等一次性任务
    })
}
```

**DaemonProvider** — 运行后台任务：

```go
func (Provider) Daemon(ctx context.Context, resolver infra.Resolver) {
    resolver.MustResolve(func(server *grpc.Server) {
        // 启动 gRPC 服务器（异步运行，不阻塞其他模块）
        server.Serve(listener)
    })
}
```

**ProviderAggregate** — 聚合子模块：

```go
func (Provider) Aggregates() []infra.Provider {
    return []infra.Provider{
        web.Provider(
            listener.FlagContext("listen"),
            web.SetRouteHandlerOption(routes),
        ),
    }
}
```

#### 条件加载（ShouldLoad）

Provider 支持按条件决定是否加载：

```go
func (Provider) ShouldLoad(conf *Config) bool {
    return conf.EnableFeatureX  // 返回 false 则不加载该模块
}
```

> 注意：`ShouldLoad` 执行时当前模块的 `Register` 尚未执行，因此只能依赖全局已注册的对象（如配置对象）。

#### 加载优先级（Priority）

控制模块的加载顺序，数字越小越先加载（默认为 1000）：

```go
func (Provider) Priority() int {
    return 10  // 比默认的 1000 更早加载
}
```

### Service（后台服务）

Service 代表一个持续运行的后台任务。只需实现 `Start() error` 方法：

```go
type HealthChecker struct {
    stopped chan struct{}
}

func (s *HealthChecker) Start() error {
    for {
        select {
        case <-s.stopped:
            return nil
        default:
            // 执行健康检查...
            time.Sleep(30 * time.Second)
        }
    }
}

// 以下方法均为可选

func (s *HealthChecker) Init(resolver infra.Resolver) error {
    s.stopped = make(chan struct{})
    return nil  // 在 Start 之前执行，用于初始化
}

func (s *HealthChecker) Stop() {
    close(s.stopped)  // 收到退出信号时调用
}

func (s *HealthChecker) Reload() {
    // 收到 reload 信号时调用（可选）
}

func (s *HealthChecker) Name() string {
    return "health-checker"  // 用于日志标识（可选）
}
```

注册 Service：

```go
ins.Service(&HealthChecker{})
```

Service 同样支持 `ShouldLoad` 和 `Priority` 接口。

## Web 开发

Glacier 内置了基于 [Gorilla Mux](https://github.com/gorilla/mux) 的 Web 开发框架，以 `DaemonProvider` 的形式集成，与其他模块统一管理。

### 基本用法

```go
ins.Provider(web.Provider(
    listener.FlagContext("listen"),  // 监听地址从命令行参数获取
    web.SetRouteHandlerOption(func(cc infra.Resolver, router web.Router, mw web.RequestMiddleware) {
        router.Get("/users", listUsers)
        router.Post("/users", createUser)
        router.Get("/users/{id}", getUser)
        router.Put("/users/{id}", updateUser)
        router.Delete("/users/{id}", deleteUser)
    }),
))
```

**Listener 构建器**有三种方式：

```go
listener.FlagContext("listen")      // 从命令行参数读取地址
listener.Default(":8080")           // 固定地址
listener.Exist(existingListener)    // 使用已有的 net.Listener
```

### 控制器

对于较复杂的应用，推荐使用控制器来组织路由。控制器需实现 `web.Controller` 接口：

```go
type UserController struct {
    resolver infra.Resolver
}

func NewUserController(cc infra.Resolver) web.Controller {
    return &UserController{resolver: cc}
}

// Register 注册该控制器下的所有路由
func (c *UserController) Register(router web.Router) {
    router.Group("/users", func(router web.Router) {
        router.Get("/", c.List)
        router.Post("/", c.Create)
        router.Get("/{id}", c.Get)
        router.Delete("/{id}", c.Delete)
    })
}

func (c *UserController) List(ctx web.Context) web.Response {
    page := ctx.IntInput("page", 1)
    return ctx.JSON(web.M{"page": page, "users": []string{}})
}

func (c *UserController) Create(ctx web.Context, userRepo *UserRepo) (*User, error) {
    var form UserForm
    if err := ctx.Unmarshal(&form); err != nil {
        return nil, web.WrapJSONError(err, http.StatusBadRequest)
    }
    return userRepo.Create(form)
}

func (c *UserController) Get(ctx web.Context) web.M {
    return web.M{"id": ctx.PathVar("id")}
}

func (c *UserController) Delete(ctx web.Context, userRepo *UserRepo) error {
    return userRepo.DeleteByID(ctx.PathVar("id"))
}
```

在路由注册函数中挂载控制器：

```go
web.SetRouteHandlerOption(func(cc infra.Resolver, router web.Router, mw web.RequestMiddleware) {
    router.WithMiddleware(mw.AccessLog(log.Default())).
        Controllers("/api",
            NewUserController(cc),
            NewOrderController(cc),
        )
})
```

### 中间件

中间件通过 `web.RequestMiddleware` 参数获取，支持链式调用：

```go
func routes(cc infra.Resolver, router web.Router, mw web.RequestMiddleware) {
    // 组合多个中间件
    router.WithMiddleware(
        mw.AccessLog(log.Default()),  // 访问日志
        mw.CORS("*"),                 // 跨域支持
        mw.AuthHandler(func(ctx web.Context, typ, credential string) error {
            // 认证逻辑
            return nil
        }),
    ).Controllers("/api", controllers...)
}
```

### Handler 返回值

Glacier 的 Handler 非常灵活，支持多种返回值模式，框架会自动处理序列化：

```go
// 返回 web.Response —— 完全控制响应格式
func(ctx web.Context) web.Response {
    return ctx.JSON(web.M{"key": "value"})       // JSON 响应
    return ctx.JSONWithCode(data, 201)            // JSON + 自定义状态码
    return ctx.YAML(data)                         // YAML 响应
    return ctx.Raw(func(w http.ResponseWriter) {  // 原始 HTTP 响应
        w.Write([]byte("raw"))
    })
    return ctx.HTML("template", data)             // HTML 模板渲染
    return ctx.Redirect("/other", 302)            // 重定向
    return ctx.JSONError("not found", 404)        // 错误响应
}

// 返回任意结构体/map —— 自动序列化为 JSON
func(ctx web.Context) web.M {
    return web.M{"hello": "world"}
}
func(ctx web.Context) *User {
    return &User{Name: "test"}
}

// 返回 error —— 非 nil 时自动转为错误响应
func(ctx web.Context) error {
    return errors.New("something went wrong")
}

// 返回 (结构体, error) —— 同时支持正常和错误情况
func(ctx web.Context, repo *UserRepo) (*User, error) {
    return repo.FindByID(ctx.PathVar("id"))
}

// 返回 string —— 直接作为响应体
func(ctx web.Context) string {
    return "Hello, World"
}

// Handler 参数支持自动依赖注入
func(ctx web.Context, db *sql.DB, repo *UserRepo) web.Response {
    // db 和 repo 由容器自动注入
    ...
}
```

### Web 配置项

```go
web.Provider(
    listener.FlagContext("listen"),
    web.SetRouteHandlerOption(routes),                        // 路由注册
    web.SetExceptionHandlerOption(exceptionHandler),          // 全局异常处理
    web.SetMuxRouteHandlerOption(muxHandler),                 // 直接操作 Gorilla Mux
    web.SetIgnoreLastSlashOption(true),                       // 忽略路径末尾的 /
    web.SetHttpReadTimeoutOption(10 * time.Second),           // 读超时
    web.SetHttpWriteTimeoutOption(30 * time.Second),          // 写超时
    web.SetHttpIdleTimeoutOption(120 * time.Second),          // 空闲超时
    web.SetMultipartFormMaxMemoryOption(32 << 20),            // 表单最大内存（32MB）
)
```

## 事件系统

Glacier 提供了发布/订阅模式的事件系统，用于模块间解耦通信。

### 定义事件

```go
type UserCreatedEvent struct {
    UserID   int
    Username string
}

// 实现 AsyncEvent 接口，使事件异步处理（可选）
func (e UserCreatedEvent) Async() bool { return true }
```

### 注册监听器

```go
ins.Provider(event.Provider(
    func(cc infra.Resolver, listener event.Listener) {
        // 监听器函数的参数类型决定它监听哪种事件
        listener.Listen(func(evt UserCreatedEvent) {
            log.Infof("新用户注册: %s", evt.Username)
            // 发送欢迎邮件等...
        })
    },
    // 使用内存事件存储（异步模式，队列长度 100）
    event.SetStoreOption(func(cc infra.Resolver) event.Store {
        return event.NewMemoryEventStore(true, 100)
    }),
))
```

### 发布事件

通过依赖注入获取 `event.Publisher`，在应用的任何地方发布事件：

```go
// 在 Handler 中发布
func(ctx web.Context, publisher event.Publisher) web.Response {
    publisher.Publish(UserCreatedEvent{UserID: 1, Username: "test"})
    return ctx.JSON(web.M{"status": "ok"})
}

// 在异步任务中发布
ins.Async(func(publisher event.Publisher) {
    publisher.Publish(UserCreatedEvent{UserID: 1, Username: "test"})
})
```

### 事件存储后端

- **内存后端**（内置）：`event.NewMemoryEventStore(async, queueSize)`
- **Redis 后端**：[redis-event-store](https://github.com/mylxsw/redis-event-store)，提供事件持久化支持，避免应用异常退出时事件丢失

## 定时任务

使用 `scheduler.Provider` 注册定时任务，基于 cron 表达式调度：

```go
ins.Provider(scheduler.Provider(
    func(cc infra.Resolver, creator scheduler.JobCreator) {
        // 每 10 秒执行一次
        creator.MustAdd("cleanup-job", "@every 10s", func() {
            log.Info("执行清理任务...")
        })

        // 每天凌晨 2 点执行，且服务启动时先执行一次
        creator.MustAddAndRunOnServerReady("daily-report", "0 2 * * *", func() {
            log.Info("生成日报...")
        })

        // 使用 WithoutOverlap 防止任务重叠：
        // 如果上一次执行尚未完成，本次调度将被跳过
        creator.MustAdd("slow-job", "@every 30s",
            scheduler.WithoutOverlap(func() {
                // 可能耗时较长的任务
            }).SkipCallback(func() {
                log.Warning("slow-job 被跳过：上一次尚未完成")
            }),
        )
    },
))
```

### 分布式锁

在集群部署时，可通过分布式锁确保定时任务在所有节点中只执行一次：

```go
scheduler.Provider(
    func(cc infra.Resolver, creator scheduler.JobCreator) { ... },
    scheduler.SetLockManagerOption(func(cc infra.Resolver) scheduler.LockManagerBuilder {
        return func(name string) scheduler.LockManager {
            // 返回分布式锁实现（如基于 Redis 的锁）
            return redisLock.New(redisClient, name, 10*time.Minute)
        }
    }),
)
```

> 参考实现：[mylxsw/distribute-locks](https://github.com/mylxsw/distribute-locks)

## 日志

Glacier 定义了 `infra.Logger` 接口用于日志抽象，支持多种日志后端：

```go
// 使用标准库日志（内置）
ins.WithLogger(log.StdLogger())

// 使用标准库日志，但过滤 DEBUG 级别
ins.WithLogger(log.StdLogger(log.DEBUG))

// 使用 asteria 日志框架（默认）
// import asteria "github.com/mylxsw/asteria/log"
ins.WithLogger(asteria.Module("glacier"))
```

也可以封装任意第三方日志库，只需实现 `infra.Logger` 接口：

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
    Critical(v ...interface{})    // 关键性错误，触发应用退出
    Criticalf(format string, v ...interface{})
}
```

## 平滑退出

Glacier 自动监听系统信号（SIGINT、SIGTERM），收到信号后按顺序执行所有注册的关闭处理函数。

```go
ins := app.Create("1.0", 3)
// 设置平滑退出超时时间（超时后强制退出）
ins.WithShutdownTimeoutFlag(5 * time.Second)

// 在 Provider 中注册关闭处理函数
func (Provider) Boot(resolver infra.Resolver) {
    resolver.MustResolve(func(gf infra.Graceful) {
        gf.AddShutdownHandler(func() {
            // 关闭数据库连接、释放资源等
        })
    })
}
```

## 完整示例

以下展示了一个包含 Web 服务、定时任务、事件系统的完整应用结构：

```
myapp/
├── main.go               # 应用入口
├── config/
│   └── config.go          # 配置定义
├── api/
│   ├── provider.go        # Web 模块 Provider
│   └── controller/
│       └── user.go        # 用户控制器
└── job/
    └── provider.go        # 定时任务 Provider
```

**main.go**

```go
func main() {
    app.MustStart("1.0", 3, func(ins *app.App) error {
        ins.WithLogger(log.StdLogger())
        ins.WithYAMLFlag("conf")                           // 支持 --conf 加载 YAML 配置文件
        ins.WithShutdownTimeoutFlag(5 * time.Second)
        ins.AddStringFlag("listen", ":8080", "HTTP 监听地址")

        // 注册配置（PreBind 确保在所有 Provider 之前注入）
        ins.PreBind(func(binder infra.Binder) {
            binder.MustSingleton(func(c infra.FlagContext) *config.Config {
                return &config.Config{Listen: c.String("listen")}
            })
        })

        // 注册各功能模块
        ins.Provider(api.ServiceProvider{})
        ins.Provider(job.ServiceProvider{})
        ins.Provider(event.Provider(
            func(cc infra.Resolver, listener event.Listener) {
                listener.Listen(func(evt UserCreatedEvent) {
                    log.Infof("新用户: %s", evt.Username)
                })
            },
            event.SetStoreOption(func(cc infra.Resolver) event.Store {
                return event.NewMemoryEventStore(true, 100)
            }),
        ))

        return nil
    })
}
```

**api/provider.go**

```go
type ServiceProvider struct{}

func (s ServiceProvider) Aggregates() []infra.Provider {
    return []infra.Provider{
        web.Provider(
            listener.FlagContext("listen"),
            web.SetRouteHandlerOption(s.routes),
            web.SetExceptionHandlerOption(func(ctx web.Context, err interface{}) web.Response {
                return ctx.JSONWithCode(web.M{"error": fmt.Sprintf("%v", err)}, 500)
            }),
        ),
    }
}

func (s ServiceProvider) routes(cc infra.Resolver, router web.Router, mw web.RequestMiddleware) {
    router.WithMiddleware(mw.AccessLog(log.Default()), mw.CORS("*")).
        Controllers("/api", controller.NewUserController(cc))
}

func (s ServiceProvider) Register(binder infra.Binder) {}
```

> 更多示例代码请参考 [example](https://github.com/mylxsw/glacier/tree/main/example) 目录。

## 执行流程

![执行流程](./arch.svg)

## 相关项目

**集成与扩展**

- [Eloquent ORM](https://github.com/mylxsw/eloquent) — 受 Laravel 启发的 Go ORM 框架
- [redis-event-store](https://github.com/mylxsw/redis-event-store) — 基于 Redis 的事件持久化后端
- [distribute-locks](https://github.com/mylxsw/distribute-locks) — 基于 Redis 的分布式锁
- [giris](https://github.com/mylxsw/giris) — [Iris Web Framework](https://www.iris-go.com/) 适配

**使用 Glacier 构建的项目**

- [WebDAV Server](https://github.com/mylxsw/webdav-server) — 支持 LDAP 的 WebDAV 服务器
- [Adanos Alert](https://github.com/mylxsw/adanos-alert) — 开源告警平台
- [Healthcheck](https://github.com/mylxsw/healthcheck) — 健康检查告警服务
- [Sync](https://github.com/mylxsw/sync) — 跨服务器文件同步
- [Tech Share](https://github.com/mylxsw/tech-share) — 团队技术分享管理
- [Universal Exporter](https://github.com/mylxsw/universal-exporter) — 通用 Prometheus Exporter
- [Graphviz Server](https://github.com/mylxsw/graphviz-server) — Graphviz 图形生成 Web 服务
- [MySQL Guard](https://github.com/mylxsw/mysql-guard) — MySQL 长事务检测与死锁告警
- [Password Server](https://github.com/mylxsw/password-server) — 随机密码生成服务
