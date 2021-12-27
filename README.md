# glacier

[TOC]

glacier is a app framework for rapid service development

Usage:

```bash
go get github.com/mylxsw/glacier/starter/application
```

Demo:

```go
app := application.Create(fmt.Sprintf("%s (%s)", Version, GitCommit[:8]))

// 添加命令行参数 flags
//app.AddStringFlag("listen", ":19945", "http listen addr")
//app.AddBoolFlag("load-job", false, "")
//
// 注册 Provider，Service 等
//app.Provider(job.ServiceProvider{}, api.ServiceProvider{})
//app.Service(&service.DemoService{}, &service.Demo2Service{})
//
//app.Singleton(func(c infra.FlagContext) *config.Config {
//    return &config.Config{
//        Listen:   c.String("listen"),
//        LoadJob: c.Bool("load-job"),
//    }
//})

if err := app.Run(os.Args); err != nil {
    panic(err)
}
```

## 核心概念

### Provider

**Provider** 接口定义如下

```go
type Provider interface {
	Register(app Binder)
	Boot(app Resolver)
}
```


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

### Service

## 第三方框架整合

- [giris](https://github.com/mylxsw/giris): [Iris Web Framework](https://www.iris-go.com/) 适配
