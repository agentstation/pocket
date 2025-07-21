package script

import (
	"encoding/json"
	"strings"

	"github.com/Shopify/go-lua"
)

// setupSandbox creates a safe Lua environment.
func setupSandbox(l *lua.State) {
	// Load only safe libraries
	lua.Require(l, "_G", lua.BaseOpen, true)
	l.Pop(1)
	lua.Require(l, "string", lua.StringOpen, true)
	l.Pop(1)
	lua.Require(l, "table", lua.TableOpen, true)
	l.Pop(1)
	lua.Require(l, "math", lua.MathOpen, true)
	l.Pop(1)

	// Limited os library - only safe functions
	lua.Require(l, "os", lua.OSOpen, true)
	l.Pop(1)
	// Remove dangerous os functions
	l.Global("os")
	l.PushNil()
	l.SetField(-2, "execute")
	l.PushNil()
	l.SetField(-2, "exit")
	l.PushNil()
	l.SetField(-2, "getenv")
	l.PushNil()
	l.SetField(-2, "remove")
	l.PushNil()
	l.SetField(-2, "rename")
	l.PushNil()
	l.SetField(-2, "setlocale")
	l.PushNil()
	l.SetField(-2, "tmpname")
	l.Pop(1)

	// Remove other dangerous functions
	l.PushNil()
	l.SetGlobal("dofile")
	l.PushNil()
	l.SetGlobal("loadfile")
	l.PushNil()
	l.SetGlobal("load")
	l.PushNil()
	l.SetGlobal("loadstring")
	l.PushNil()
	l.SetGlobal("require")

	// Add safe utilities
	l.Register("json_encode", jsonEncode)
	l.Register("json_decode", jsonDecode)
	l.Register("str_trim", strTrim)
	l.Register("str_split", strSplit)
	l.Register("str_contains", strContains)
	l.Register("str_replace", strReplace)
	l.Register("type_of", typeOf)
}

// pushValue converts a Go value to Lua.
func pushValue(l *lua.State, v interface{}) {
	switch val := v.(type) {
	case nil:
		l.PushNil()
	case bool:
		l.PushBoolean(val)
	case int:
		l.PushInteger(val)
	case int64:
		l.PushInteger(int(val))
	case float64:
		l.PushNumber(val)
	case string:
		l.PushString(val)
	case []interface{}:
		l.NewTable()
		for i, item := range val {
			l.PushInteger(i + 1)
			pushValue(l, item)
			l.SetTable(-3)
		}
	case map[string]interface{}:
		l.NewTable()
		for k, v := range val {
			l.PushString(k)
			pushValue(l, v)
			l.SetTable(-3)
		}
	default:
		// Try to convert to JSON as fallback
		if data, err := json.Marshal(val); err == nil {
			l.PushString(string(data))
		} else {
			l.PushNil()
		}
	}
}

// pullValue converts a Lua value to Go.
func pullValue(l *lua.State, idx int) interface{} {
	switch l.TypeOf(idx) {
	case lua.TypeNil:
		return nil
	case lua.TypeBoolean:
		return l.ToBoolean(idx)
	case lua.TypeNumber:
		n, _ := l.ToNumber(idx)
		return n
	case lua.TypeString:
		s, _ := l.ToString(idx)
		return s
	case lua.TypeTable:
		// First, push the table to the top of the stack for easier manipulation
		l.PushValue(idx)

		// Check if it's an array or object
		isArray := true
		maxIndex := 0

		l.PushNil()
		for l.Next(-2) {
			if l.TypeOf(-2) != lua.TypeNumber {
				isArray = false
				l.Pop(2)
				break
			}
			n, _ := l.ToNumber(-2)
			i := int(n)
			if i > maxIndex {
				maxIndex = i
			}
			l.Pop(1) // Remove value, keep key for next iteration
		}

		if isArray && maxIndex > 0 {
			// Array
			arr := make([]interface{}, maxIndex)
			for i := 1; i <= maxIndex; i++ {
				l.PushInteger(i)
				l.Table(-2) // Table is at -2
				arr[i-1] = pullValue(l, -1)
				l.Pop(1)
			}
			l.Pop(1) // Remove the table copy
			return arr
		}

		// Object
		obj := make(map[string]interface{})
		l.PushNil()
		for l.Next(-2) {
			key, _ := l.ToString(-2)
			value := pullValue(l, -1)
			obj[key] = value
			l.Pop(1) // Remove value, keep key for next iteration
		}
		l.Pop(1) // Remove the table copy
		return obj
	default:
		return nil
	}
}

// Lua utility functions

func jsonEncode(l *lua.State) int {
	value := pullValue(l, 1)
	data, err := json.Marshal(value)
	if err != nil {
		l.PushNil()
		l.PushString(err.Error())
		return 2
	}
	l.PushString(string(data))
	return 1
}

func jsonDecode(l *lua.State) int {
	str := lua.CheckString(l, 1)
	var value interface{}
	if err := json.Unmarshal([]byte(str), &value); err != nil {
		l.PushNil()
		l.PushString(err.Error())
		return 2
	}
	pushValue(l, value)
	return 1
}

func strTrim(l *lua.State) int {
	str := lua.CheckString(l, 1)
	l.PushString(strings.TrimSpace(str))
	return 1
}

func strSplit(l *lua.State) int {
	str := lua.CheckString(l, 1)
	sep := lua.CheckString(l, 2)
	parts := strings.Split(str, sep)

	l.NewTable()
	for i, part := range parts {
		l.PushInteger(i + 1)
		l.PushString(part)
		l.SetTable(-3)
	}
	return 1
}

func strContains(l *lua.State) int {
	str := lua.CheckString(l, 1)
	substr := lua.CheckString(l, 2)
	l.PushBoolean(strings.Contains(str, substr))
	return 1
}

func strReplace(l *lua.State) int {
	str := lua.CheckString(l, 1)
	old := lua.CheckString(l, 2)
	newStr := lua.CheckString(l, 3)

	// Optional count parameter
	count := -1
	if l.Top() >= 4 {
		count = lua.CheckInteger(l, 4)
	}

	l.PushString(strings.Replace(str, old, newStr, count))
	return 1
}

func typeOf(l *lua.State) int {
	t := l.TypeOf(1)
	switch t {
	case lua.TypeNil:
		l.PushString("nil")
	case lua.TypeBoolean:
		l.PushString("boolean")
	case lua.TypeNumber:
		l.PushString("number")
	case lua.TypeString:
		l.PushString("string")
	case lua.TypeTable:
		l.PushString("table")
	case lua.TypeFunction:
		l.PushString("function")
	default:
		l.PushString("unknown")
	}
	return 1
}
