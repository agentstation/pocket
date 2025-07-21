package script

import (
	"testing"

	"github.com/Shopify/go-lua"
)

func TestPushPullValue(t *testing.T) {
	l := lua.NewState()

	tests := []struct {
		name  string
		value interface{}
	}{
		{"nil", nil},
		{"bool true", true},
		{"bool false", false},
		{"int", 42},
		{"float", 3.14},
		{"string", "hello"},
		{"array", []interface{}{1, 2, 3}},
		{"map", map[string]interface{}{"key": "value", "num": 123}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Push value
			pushValue(l, tt.value)

			// Pull value back
			result := pullValue(l, -1)
			l.Pop(1)

			// For arrays and maps, we can't do direct equality
			// because they're different types after conversion
			switch v := tt.value.(type) {
			case []interface{}:
				arr, ok := result.([]interface{})
				if !ok {
					t.Errorf("Expected array, got %T", result)
					return
				}
				if len(arr) != len(v) {
					t.Errorf("Array length mismatch: got %d, want %d", len(arr), len(v))
				}
			case map[string]interface{}:
				m, ok := result.(map[string]interface{})
				if !ok {
					t.Errorf("Expected map, got %T", result)
					return
				}
				if len(m) != len(v) {
					t.Errorf("Map length mismatch: got %d, want %d", len(m), len(v))
				}
			case int:
				// Ints are converted to float64 in Lua
				if f, ok := result.(float64); !ok || f != float64(v) {
					t.Errorf("Got %v (%T), want %v", result, result, v)
				}
			default:
				if result != tt.value {
					t.Errorf("Got %v (%T), want %v (%T)", result, result, tt.value, tt.value)
				}
			}
		})
	}
}

func TestLuaUtilityFunctions(t *testing.T) {
	// Test json_encode
	t.Run("json_encode", func(t *testing.T) {
		l := lua.NewState()
		setupSandbox(l)

		l.NewTable()
		l.PushString("test")
		l.SetField(-2, "key")

		jsonEncode(l)
		result, _ := l.ToString(-1)
		l.Pop(1)

		if result != `{"key":"test"}` {
			t.Errorf("json_encode failed: got %s", result)
		}
	})

	// Test str_trim
	t.Run("str_trim", func(t *testing.T) {
		l := lua.NewState()
		setupSandbox(l)

		l.PushString("  hello world  ")
		strTrim(l)
		result, _ := l.ToString(-1)
		l.Pop(1)

		if result != "hello world" {
			t.Errorf("str_trim failed: got %q", result)
		}
	})

	// Test str_contains
	t.Run("str_contains", func(t *testing.T) {
		l := lua.NewState()
		setupSandbox(l)

		l.PushString("hello world")
		l.PushString("world")
		strContains(l)
		result := l.ToBoolean(-1)
		l.Pop(1)

		if !result {
			t.Errorf("str_contains failed: expected true")
		}
	})
}
