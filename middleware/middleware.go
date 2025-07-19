// Package middleware provides node enhancement patterns for cross-cutting concerns
// like logging, metrics, retries, and circuit breakers.
package middleware

import (
	"context"
	"reflect"

	"github.com/agentstation/pocket"
)

// Middleware modifies node behavior.
type Middleware func(pocket.Node) pocket.Node

// middlewareNode wraps a node to modify its behavior.
type middlewareNode struct {
	inner pocket.Node
	name  string
	prep  pocket.PrepFunc
	exec  pocket.ExecFunc
	post  pocket.PostFunc
}

func (m *middlewareNode) Name() string {
	return m.name
}

func (m *middlewareNode) Prep(ctx context.Context, store pocket.StoreReader, input any) (any, error) {
	if m.prep != nil {
		return m.prep(ctx, store, input)
	}
	return m.inner.Prep(ctx, store, input)
}

func (m *middlewareNode) Exec(ctx context.Context, prepResult any) (any, error) {
	if m.exec != nil {
		return m.exec(ctx, prepResult)
	}
	return m.inner.Exec(ctx, prepResult)
}

func (m *middlewareNode) Post(ctx context.Context, store pocket.StoreWriter, input, prepResult, execResult any) (output any, next string, err error) {
	if m.post != nil {
		return m.post(ctx, store, input, prepResult, execResult)
	}
	return m.inner.Post(ctx, store, input, prepResult, execResult)
}

func (m *middlewareNode) Connect(action string, next pocket.Node) pocket.Node {
	return m.inner.Connect(action, next)
}

func (m *middlewareNode) Successors() map[string]pocket.Node {
	return m.inner.Successors()
}

func (m *middlewareNode) InputType() reflect.Type {
	return m.inner.InputType()
}

func (m *middlewareNode) OutputType() reflect.Type {
	return m.inner.OutputType()
}

// Chain combines multiple middlewares into a single middleware.
// Middlewares are applied in reverse order (like function composition).
func Chain(middlewares ...Middleware) Middleware {
	return func(node pocket.Node) pocket.Node {
		for i := len(middlewares) - 1; i >= 0; i-- {
			node = middlewares[i](node)
		}
		return node
	}
}

// Apply applies middleware to a node.
func Apply(node pocket.Node, middlewares ...Middleware) pocket.Node {
	for _, mw := range middlewares {
		node = mw(node)
	}
	return node
}
