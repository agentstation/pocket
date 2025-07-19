package testutil

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/agentstation/pocket"
)

// Assert provides test assertions.
type Assert struct {
	t *testing.T
}

// NewAssert creates a new assert helper.
func NewAssert(t *testing.T) *Assert {
	return &Assert{t: t}
}

// Equal asserts that two values are equal.
func (a *Assert) Equal(expected, actual any, msgAndArgs ...any) {
	a.t.Helper()
	if !reflect.DeepEqual(expected, actual) {
		a.fail(fmt.Sprintf("Expected: %v\nActual: %v", expected, actual), msgAndArgs...)
	}
}

// NotEqual asserts that two values are not equal.
func (a *Assert) NotEqual(expected, actual any, msgAndArgs ...any) {
	a.t.Helper()
	if reflect.DeepEqual(expected, actual) {
		a.fail(fmt.Sprintf("Expected values to be different, but both were: %v", actual), msgAndArgs...)
	}
}

// Nil asserts that a value is nil.
func (a *Assert) Nil(value any, msgAndArgs ...any) {
	a.t.Helper()
	if !isNil(value) {
		a.fail(fmt.Sprintf("Expected nil, but got: %v", value), msgAndArgs...)
	}
}

// NotNil asserts that a value is not nil.
func (a *Assert) NotNil(value any, msgAndArgs ...any) {
	a.t.Helper()
	if isNil(value) {
		a.fail("Expected non-nil value, but got nil", msgAndArgs...)
	}
}

// True asserts that a value is true.
func (a *Assert) True(value bool, msgAndArgs ...any) {
	a.t.Helper()
	if !value {
		a.fail("Expected true, but got false", msgAndArgs...)
	}
}

// False asserts that a value is false.
func (a *Assert) False(value bool, msgAndArgs ...any) {
	a.t.Helper()
	if value {
		a.fail("Expected false, but got true", msgAndArgs...)
	}
}

// Error asserts that an error occurred.
func (a *Assert) Error(err error, msgAndArgs ...any) {
	a.t.Helper()
	if err == nil {
		a.fail("Expected error, but got nil", msgAndArgs...)
	}
}

// NoError asserts that no error occurred.
func (a *Assert) NoError(err error, msgAndArgs ...any) {
	a.t.Helper()
	if err != nil {
		a.fail(fmt.Sprintf("Expected no error, but got: %v", err), msgAndArgs...)
	}
}

// Contains asserts that a string contains a substring.
func (a *Assert) Contains(s, substr string, msgAndArgs ...any) {
	a.t.Helper()
	if !contains(s, substr) {
		a.fail(fmt.Sprintf("Expected %q to contain %q", s, substr), msgAndArgs...)
	}
}

// NotContains asserts that a string does not contain a substring.
func (a *Assert) NotContains(s, substr string, msgAndArgs ...any) {
	a.t.Helper()
	if contains(s, substr) {
		a.fail(fmt.Sprintf("Expected %q to not contain %q", s, substr), msgAndArgs...)
	}
}

// Len asserts the length of a collection.
func (a *Assert) Len(collection any, length int, msgAndArgs ...any) {
	a.t.Helper()
	actual := getLen(collection)
	if actual != length {
		a.fail(fmt.Sprintf("Expected length %d, but got %d", length, actual), msgAndArgs...)
	}
}

// Empty asserts that a collection is empty.
func (a *Assert) Empty(collection any, msgAndArgs ...any) {
	a.t.Helper()
	if getLen(collection) != 0 {
		a.fail(fmt.Sprintf("Expected empty collection, but got length %d", getLen(collection)), msgAndArgs...)
	}
}

// NotEmpty asserts that a collection is not empty.
func (a *Assert) NotEmpty(collection any, msgAndArgs ...any) {
	a.t.Helper()
	if getLen(collection) == 0 {
		a.fail("Expected non-empty collection, but got empty", msgAndArgs...)
	}
}

// Eventually asserts that a condition becomes true within a timeout.
func (a *Assert) Eventually(condition func() bool, timeout time.Duration, msgAndArgs ...any) {
	a.t.Helper()

	deadline := time.Now().Add(timeout)
	interval := timeout / 100
	if interval < time.Millisecond {
		interval = time.Millisecond
	}

	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(interval)
	}

	a.fail("Condition did not become true within timeout", msgAndArgs...)
}

// Panics asserts that a function panics.
func (a *Assert) Panics(fn func(), msgAndArgs ...any) {
	a.t.Helper()

	defer func() {
		if r := recover(); r == nil {
			a.fail("Expected panic, but function completed normally", msgAndArgs...)
		}
	}()

	fn()
}

// NotPanics asserts that a function does not panic.
func (a *Assert) NotPanics(fn func(), msgAndArgs ...any) {
	a.t.Helper()

	defer func() {
		if r := recover(); r != nil {
			a.fail(fmt.Sprintf("Expected no panic, but got: %v", r), msgAndArgs...)
		}
	}()

	fn()
}

// InDelta asserts that two floats are within a delta.
func (a *Assert) InDelta(expected, actual, delta float64, msgAndArgs ...any) {
	a.t.Helper()

	diff := expected - actual
	if diff < 0 {
		diff = -diff
	}

	if diff > delta {
		a.fail(fmt.Sprintf("Expected %f ± %f, but got %f (diff: %f)", expected, delta, actual, diff), msgAndArgs...)
	}
}

// WithinDuration asserts that two times are within a duration.
func (a *Assert) WithinDuration(expected, actual time.Time, delta time.Duration, msgAndArgs ...any) {
	a.t.Helper()

	diff := expected.Sub(actual)
	if diff < 0 {
		diff = -diff
	}

	if diff > delta {
		a.fail(fmt.Sprintf("Expected %v ± %v, but got %v (diff: %v)", expected, delta, actual, diff), msgAndArgs...)
	}
}

// Helper functions

func (a *Assert) fail(message string, msgAndArgs ...any) {
	if len(msgAndArgs) > 0 {
		if format, ok := msgAndArgs[0].(string); ok && len(msgAndArgs) > 1 {
			message = fmt.Sprintf(format, msgAndArgs[1:]...) + "\n" + message
		} else if len(msgAndArgs) == 1 {
			message = fmt.Sprintf("%v\n%s", msgAndArgs[0], message)
		}
	}
	a.t.Fatal(message)
}

func isNil(value any) bool {
	if value == nil {
		return true
	}

	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return v.IsNil()
	}

	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || substr == "" || (s != "" && s[0:len(substr)] == substr) || contains(s[1:], substr))
}

func getLen(value any) int {
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice, reflect.String:
		return v.Len()
	default:
		panic(fmt.Sprintf("Cannot get length of type %T", value))
	}
}

// GraphAssert provides graph-specific assertions.
type GraphAssert struct {
	*Assert
}

// NewGraphAssert creates graph-specific assertions.
func NewGraphAssert(t *testing.T) *GraphAssert {
	return &GraphAssert{
		Assert: NewAssert(t),
	}
}

// GraphCompletes asserts that a graph completes successfully.
func (fa *GraphAssert) GraphCompletes(graph *pocket.Graph, input any) any {
	fa.t.Helper()

	ctx := context.Background()
	result, err := graph.Run(ctx, input)
	fa.NoError(err, "Graph execution failed")

	return result
}

// GraphFails asserts that a graph fails with an error.
func (fa *GraphAssert) GraphFails(graph *pocket.Graph, input any) error {
	fa.t.Helper()

	ctx := context.Background()
	_, err := graph.Run(ctx, input)
	fa.Error(err, "Expected graph to fail")

	return err
}

// NodeExecutes asserts that a node executes successfully.
func (fa *GraphAssert) NodeExecutes(node *pocket.Node, store pocket.Store, input any) any {
	fa.t.Helper()

	graph := pocket.NewGraph(node, store)
	return fa.GraphCompletes(graph, input)
}

// StoreContains asserts that a store contains a key.
func (fa *GraphAssert) StoreContains(store pocket.Store, key string) any {
	fa.t.Helper()

	ctx := context.Background()
	value, exists := store.Get(ctx, key)
	fa.True(exists, "Expected store to contain key: %s", key)

	return value
}

// StoreNotContains asserts that a store does not contain a key.
func (fa *GraphAssert) StoreNotContains(store pocket.Store, key string) {
	fa.t.Helper()

	ctx := context.Background()
	_, exists := store.Get(ctx, key)
	fa.False(exists, "Expected store to not contain key: %s", key)
}
