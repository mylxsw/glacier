# Glaicer Framework

[TOC]

Glacier 是一款支持依赖注入的应用开发框架。

## Feature

- **依赖注入**：通过依赖注入的方式来管理对象的依赖，支持单例、原型对象创建
- **模块化**：通过 Provider 特性，轻松实现应用的模块化
- **内置 Web 开发支持**：Glacier 内置了对 Web 应用开发的支持，提供了功能丰富的 API 简化 web 开发

## Usage

创建一个新的项目，使用下面的命令安装 Glacier 开发框架

```bash
go get github.com/mylxsw/glacier/starter/application
```

示例:

```go
app := application.Create("v1.0")
app.AddStringFlag("listen", ":8080", "http listen address")

app.Provider(web.Provider(
	listener.FlagContext("listen"),
	web.SetRouteHandlerOption(func(cc infra.Resolver, router web.Router, mw web.RequestMiddleware) {
		router.Get("/", func(ctx web.Context) web.Response {
			return ctx.JSON(web.M{})
		})
	}),
))

application.MustRun(app)
```

## Core Concept

### Dependency Injection

Glacier 框架充分利用了 [container](https://github.com/mylxsw/container) 项目提供的依赖注入支持，在应用提供了功能强大的依赖注入特性。

在使用依赖注入特性时，无论是 `Binder` 还是 `Resolver`，都会有一个名为 `initialize interface{}` 的参数，该参数类型为 `interface{}`，但是实际使用时，它的类型为符合一定规则的函数，后面在 `Binder` 和 `Resolver` 部分将会详细说明。

#### Binder

`infra.Binder` 是一个抽象化之后的接口，用于将实例的创建方法绑定到容器，提供了以下常用方法

- `Prototype(initialize interface{}) error` 原型绑定，每次访问绑定的实例都会基于 `initialize` 函数重新创建新的实例
- `Singleton(initialize interface{}) error` 单例绑定，每次访问绑定的实例都是同一个，只会在第一次访问的时候创建初始实例
- `BindValue(key string, value interface{}) error` 将一个具体的值绑定到 `key`

`Prototype` 和 `Singleton` 方法参数 `initialize interface{}` 支持以下两种形式

- 形式1：`func(注入参数列表...) (绑定类型, error)`

	```go
	binder.Singleton(func() (*sql.DB, error) {
		return sql.Open("mysql", "user:pwd@tcp(ip:3306)/dbname")
	})
	```

- 形式2：`func(注入参数列表...) 绑定类型`

	```go
	binder.Singleton(func() UserRepo { return &userRepoImpl{} })
	binder.Singleton(func(db *sql.DB) UserRepo { 
		// 这里我们创建的 userRepoImpl 对象，依赖 sql.DB 对象，只需要在函数
		// 参数中，将依赖列举出来，容器会自动完成这些对象的创建
		return &userRepoImpl{db: db} 
	})
	```

#### Resolver

`infra.Resolver` 是一个抽象化的接口，用于通过依赖注入的方式获取实例，提供了以下常用方法

- `Resolve(callback interface{}) error` 
- `Call(callback interface{}) ([]interface{}, error)`
- `AutoWire(object interface{}) error` 自动对结构体对象进行依赖注入，object 必须是结构体对象的指针。自动注入字段（公开和私有均支持）需要添加 `autowire` tag，支持以下两种
	- autowire:"@" 根据字段的类型来注入
	- autowire:"自定义key" 根据自定义的key来注入（查找名为 key 的绑定）

- `Get(key interface{}) (interface{}, error)`


### Provider

在 Glacier 应用开发框架中，Provider 是应用模块化的核心，每个独立的功能模块通过 Provider 完成实例初始化，每个 Provider 都需要实现 `infra.Provider` 接口。 在每个功能模块中，我们通常会创建一个名为 provider.go 的文件，在该文件中创建一个 provider 实现

```
type Provider struct{}

func (p Provider) Register(cc infra.Binder) {
	
}
```

**Provider** 接口只有一个必须实现的方法 `Register(cc infra.Binder)`，该方法用于注册当前模块的对象到 Container 中，实现依赖注入的支持。以下是 `cc infra.Binder` 支持的常用方法

- `Prototype(initialize interface{}) error`
- `Singleton(initialize interface{}) error`
- `BindValue(key string, value interface{}) error`

#### ProviderBoot

#### DaemonProvider

#### ProviderAggregate

#### Service
#### ModuleLoadPolicy 

**Provider** 支持按需加载，要使用此功能，只需要让 **Provider** 实现对象实现 **ModuleLoadPolicy** 接口即可。

```go
type ModuleLoadPolicy interface {
	// ShouldLoad 如果返回 true，则加载该 Provider，否则跳过
	ShouldLoad(c FlagContext) bool
}
```

**ModuleLoadPolicy** 接口的 `ShouldLoad` 方法用于控制该 **Provider** 是否加载。

#### DaemonProvider

#### ProviderAggregate

#### Priority

实现 `infra.Priority` 接口的 Provider、Service，会按照 `priority()` 方法的返回值大小依次加载，值越大，加载顺序越靠后，默认的优先级为 `1000`。

### Web Framework

#### Usage

#### Route

#### Request

#### Response

#### Session

#### Cookie

#### Exception

### Event

#### 本地内存作为事件存储后端

#### Redis 作为事件存储后端

[redis-event-store](https://github.com/mylxsw/redis-event-store)

### Crontab

### Log

### Collection

### Eloquent ORM

### Others

#### 平滑关闭

[graceful](https://github.com/mylxsw/graceful)

## Third-party integration

- [giris](https://github.com/mylxsw/giris): [Iris Web Framework](https://www.iris-go.com/) 适配

## Examples

- [WebDAV Server](https://github.com/mylxsw/webdav-server) 一款支持 LDAP 作为用户数据库的 WebDAV 服务器
- [Adanos Alert](https://github.com/mylxsw/adanos-alert) 一个功能强大的开源告警平台，通过事件聚合机制，为监控系统提供钉钉、邮件、HTTP、JIRA、语音电话等告警方式的支持
- [Healthcheck](https://github.com/mylxsw/healthcheck) 为应用服务提供健康检查告警支持
- [Sync](https://github.com/mylxsw/sync) 跨服务器文件同步服务
- [Tech Share](https://github.com/mylxsw/tech-share) 一个用于中小型团队内部技术分享管理的 Web 应用
- [Universal Exporter](https://github.com/mylxsw/universal-exporter) 一个通用的 Prometheus 维度工具，目前支持从数据库中查询生成 Metric 数据
- [Graphviz Server](https://github.com/mylxsw/graphviz-server) 一个 Web 服务，封装了对 Graphviz 的接口调用，实现通过 Web API 的方式生成 Graphviz 图形
- [MySQL Guard](https://github.com/mylxsw/mysql-guard) 用于 MySQL 长事务检测杀死和死锁告警
- [Password Server](https://github.com/mylxsw/password-server) 一个生成随机密码的简单 web 服务器