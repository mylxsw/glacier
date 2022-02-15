package listener

import (
	"errors"
	"net"

	"github.com/mylxsw/glacier/infra"
)

// defaultBuilder 默认的 http listener 构建器，监听 127.0.0.1:8080 端口
type defaultBuilder struct {
	listenAddr string
}

// Default 创建默认的 http listener 构建器，监听 127.0.0.1:8080 端口
func Default(listenAddr string) infra.ListenerBuilder {
	return defaultBuilder{listenAddr: listenAddr}
}

func (e defaultBuilder) Build(infra.Resolver) (net.Listener, error) {
	return net.Listen("tcp", e.listenAddr)
}

// flagContextBuilder 基于 FlagContext 程序参数的 http listener 构建器
type flagContextBuilder struct {
	flagName string
}

// FlagContext 创建基于 FlagContext 程序参数的 http listener 构建器
func FlagContext(flagName string) infra.ListenerBuilder {
	return &flagContextBuilder{flagName: flagName}
}

func (builder *flagContextBuilder) Build(cc infra.Resolver) (net.Listener, error) {
	listenAddr := cc.MustGet((*infra.FlagContext)(nil)).(infra.FlagContext).String(builder.flagName)
	if listenAddr == "" {
		return nil, errors.New("listen addr is required")
	}

	return net.Listen("tcp", listenAddr)
}

type existedBuilder struct {
	listener net.Listener
}

// Exist 使用已经创建过的 listener
func Exist(listener net.Listener) infra.ListenerBuilder {
	return existedBuilder{listener: listener}
}

func (e existedBuilder) Build(infra.Resolver) (net.Listener, error) {
	return e.listener, nil
}
