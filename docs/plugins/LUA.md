# Lua Scripting Guide

Lua scripting in Pocket provides a safe, sandboxed environment for implementing custom business logic. This guide covers everything you need to know about writing Lua scripts for Pocket workflows.

## Table of Contents

- [Introduction](#introduction)
- [Getting Started](#getting-started)
- [Available APIs](#available-apis)
- [Security and Sandboxing](#security-and-sandboxing)
- [Examples](#examples)
- [Best Practices](#best-practices)
- [Debugging](#debugging)
- [Performance](#performance)

## Introduction

Lua is a lightweight, embeddable scripting language that's perfect for extending applications. In Pocket, Lua scripts run in a sandboxed environment with access to safe APIs for data processing.

### Why Lua?

- **Simple syntax** - Easy to learn and write
- **Fast execution** - Minimal overhead
- **Safe sandboxing** - No access to dangerous operations
- **Rich data structures** - Tables work well with JSON
- **Proven track record** - Used in many production systems

## Getting Started

### Basic Script Structure

Every Lua script in Pocket follows this pattern:

```lua
-- 1. Access input data (always available as 'input')
local data = input

-- 2. Process the data
local result = process_data(data)

-- 3. Return the result (becomes the node's output)
return result
```

### Your First Script

```yaml
nodes:
  - name: hello-lua
    type: lua
    config:
      script: |
        -- Access input
        local name = input.name or "World"
        
        -- Process
        local greeting = "Hello, " .. name .. "!"
        
        -- Return result
        return {
          message = greeting,
          timestamp = os.time()
        }
```

## Available APIs

### Global Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `input` | The input data passed to the node | `local items = input.items` |

### JSON Functions

#### json_encode(value)
Converts a Lua value to a JSON string.

```lua
local data = {name = "Alice", age = 30}
local json_str = json_encode(data)
-- Result: '{"name":"Alice","age":30}'
```

#### json_decode(string)
Parses a JSON string into a Lua value.

```lua
local json_str = '{"name":"Bob","scores":[85,92,78]}'
local data = json_decode(json_str)
-- Result: {name = "Bob", scores = {85, 92, 78}}
```

### String Functions

#### str_trim(string)
Removes leading and trailing whitespace.

```lua
local cleaned = str_trim("  hello world  ")
-- Result: "hello world"
```

#### str_split(string, delimiter)
Splits a string into an array.

```lua
local parts = str_split("apple,banana,orange", ",")
-- Result: {"apple", "banana", "orange"}
```

#### str_contains(string, substring)
Checks if a string contains a substring.

```lua
if str_contains(input.email, "@example.com") then
  return {domain = "internal"}
end
```

#### str_replace(string, old, new, [count])
Replaces occurrences of a substring.

```lua
local text = str_replace("Hello World", "World", "Lua")
-- Result: "Hello Lua"

-- Replace only first occurrence
local text = str_replace("foo bar foo", "foo", "baz", 1)
-- Result: "baz bar foo"
```

### Type Functions

#### type_of(value)
Returns the type of a value as a string.

```lua
type_of(42)          -- "number"
type_of("hello")     -- "string"
type_of({1,2,3})     -- "table"
type_of(true)        -- "boolean"
type_of(nil)         -- "nil"
```

### Standard Lua Libraries

The following standard Lua libraries are available:

- **Basic functions** - print, assert, error, pairs, ipairs, next, tonumber, tostring
- **String library** - string.upper, string.lower, string.sub, string.format, etc.
- **Table library** - table.insert, table.remove, table.sort, table.concat
- **Math library** - math.sin, math.cos, math.random, math.floor, math.ceil, etc.

## Security and Sandboxing

### What's Disabled

For security, the following are NOT available:

- ❌ `io` library - No file system access
- ❌ `os.execute` - No shell commands
- ❌ `require` - No loading external modules
- ❌ `dofile` / `loadfile` - No loading external scripts
- ❌ `debug` library - No debugging/introspection
- ❌ Network operations - No socket access

### What's Available

- ✅ Data processing and transformation
- ✅ String manipulation
- ✅ Mathematical operations
- ✅ JSON encoding/decoding
- ✅ Table operations
- ✅ Limited `os` functions (time, date)

### Timeout Protection

Scripts have a default timeout of 30 seconds. For long-running operations:

```yaml
- name: complex-processing
  type: lua
  config:
    script: "..."
    timeout: "2m"  # Increase timeout to 2 minutes
```

## Examples

### Data Filtering

```lua
-- Filter items based on criteria
local filtered = {}
for _, item in ipairs(input.items or {}) do
  if item.active and item.score >= 0.7 then
    table.insert(filtered, item)
  end
end

return {
  filtered = filtered,
  count = #filtered,
  total = #(input.items or {})
}
```

### Data Transformation

```lua
-- Transform order data
local order = input
local items = {}
local total = 0

for _, item in ipairs(order.line_items or {}) do
  local processed = {
    sku = item.product_id,
    name = str_trim(item.name),
    quantity = item.qty,
    unit_price = item.price,
    subtotal = item.qty * item.price
  }
  
  table.insert(items, processed)
  total = total + processed.subtotal
end

return {
  order_id = order.id,
  customer = order.customer_id,
  items = items,
  subtotal = total,
  tax = total * 0.08,
  total = total * 1.08,
  item_count = #items
}
```

### Conditional Logic

```lua
-- Route based on complex conditions
local score = 0

-- Calculate score based on multiple factors
if input.value > 1000 then score = score + 1 end
if input.customer_tier == "premium" then score = score + 2 end
if input.priority == "high" then score = score + 3 end
if #(input.items or {}) > 10 then score = score + 1 end

-- Determine action based on score
local action = "standard"
local reason = "default processing"

if score >= 6 then
  action = "expedite"
  reason = "high value customer with priority order"
elseif score >= 4 then
  action = "priority"
  reason = "meets priority criteria"
elseif input.value < 50 then
  action = "batch"
  reason = "low value order for batch processing"
end

return {
  action = action,
  score = score,
  reason = reason,
  original = input
}
```

### JSON Processing

```lua
-- Parse and transform JSON data
local raw_json = input.raw_data
local success, data = pcall(json_decode, raw_json)

if not success then
  return {
    error = "Invalid JSON",
    raw = raw_json
  }
end

-- Extract and transform nested data
local users = {}
for _, record in ipairs(data.records or {}) do
  if record.type == "user" then
    table.insert(users, {
      id = record.id,
      email = record.attributes.email,
      name = record.attributes.first_name .. " " .. record.attributes.last_name,
      active = record.attributes.status == "active"
    })
  end
end

-- Re-encode as JSON
return {
  users = users,
  users_json = json_encode(users),
  count = #users
}
```

### Error Handling

```lua
-- Safe error handling pattern
local function safe_divide(a, b)
  if b == 0 then
    return nil, "Division by zero"
  end
  return a / b, nil
end

local function process_metrics(data)
  local results = {}
  
  for _, metric in ipairs(data.metrics or {}) do
    local value, err = safe_divide(metric.numerator, metric.denominator)
    
    if err then
      table.insert(results, {
        name = metric.name,
        error = err,
        status = "failed"
      })
    else
      table.insert(results, {
        name = metric.name,
        value = value,
        status = "success"
      })
    end
  end
  
  return results
end

-- Main execution
local success, results = pcall(process_metrics, input)

if success then
  return {
    results = results,
    success = true
  }
else
  return {
    error = "Processing failed: " .. tostring(results),
    success = false
  }
end
```

## Best Practices

### 1. Always Check for nil

Lua uses nil for missing values. Always check:

```lua
-- Bad: Assumes items exists
for _, item in ipairs(input.items) do
  -- This crashes if input.items is nil
end

-- Good: Safe with default
for _, item in ipairs(input.items or {}) do
  -- This works even if input.items is nil
end
```

### 2. Use Local Variables

Local variables are faster and cleaner:

```lua
-- Bad: Global variables pollute namespace
count = 0
for i = 1, 10 do
  count = count + 1
end

-- Good: Local variables are contained
local count = 0
for i = 1, 10 do
  count = count + 1
end
```

### 3. Handle Type Conversions

Be explicit about type conversions:

```lua
-- Convert string to number safely
local num = tonumber(input.value)
if not num then
  return {error = "Invalid number: " .. tostring(input.value)}
end

-- Format numbers for display
local formatted = string.format("%.2f", num)
```

### 4. Structure Your Code

For complex scripts, use functions:

```lua
-- Define helper functions
local function validate_email(email)
  return str_contains(email, "@") and str_contains(email, ".")
end

local function process_user(user)
  return {
    id = user.id,
    email = string.lower(user.email),
    valid = validate_email(user.email)
  }
end

-- Main logic
local results = {}
for _, user in ipairs(input.users or {}) do
  table.insert(results, process_user(user))
end

return results
```

### 5. Return Meaningful Data

Always return structured data:

```lua
-- Bad: Just return a value
return #filtered_items

-- Good: Return context
return {
  items = filtered_items,
  count = #filtered_items,
  total = #original_items,
  filter_rate = #filtered_items / #original_items
}
```

## Debugging

### Print Debugging

While `print()` is available, output goes to logs:

```lua
print("Debug: Processing " .. #(input.items or {}) .. " items")

for i, item in ipairs(input.items or {}) do
  print(string.format("Item %d: %s = %g", i, item.name, item.value))
end
```

### Error Messages

Use meaningful error messages:

```lua
if not input.required_field then
  error("Missing required field: required_field")
end

if type(input.data) ~= "table" then
  error("Expected data to be a table, got: " .. type(input.data))
end
```

### Test Data Structure

Test your assumptions about data:

```lua
-- Debug helper to inspect data structure
local function inspect(value, indent)
  indent = indent or ""
  if type(value) == "table" then
    print(indent .. "{")
    for k, v in pairs(value) do
      print(indent .. "  " .. tostring(k) .. ": " .. type(v))
      if type(v) == "table" and indent == "" then
        inspect(v, indent .. "    ")
      end
    end
    print(indent .. "}")
  else
    print(indent .. type(value) .. ": " .. tostring(value))
  end
end

-- Use during development
inspect(input)
```

## Performance

### Tips for Fast Scripts

1. **Minimize table lookups**:
```lua
-- Slow: Multiple lookups
for i = 1, 1000 do
  process(input.data.items[i].value)
end

-- Fast: Cache reference
local items = input.data.items
for i = 1, 1000 do
  process(items[i].value)
end
```

2. **Use table.insert for arrays**:
```lua
-- Slow: Computing length each time
local results = {}
for i = 1, n do
  results[#results + 1] = process(i)
end

-- Fast: Let Lua handle it
local results = {}
for i = 1, n do
  table.insert(results, process(i))
end
```

3. **Avoid string concatenation in loops**:
```lua
-- Slow: Creates many intermediate strings
local s = ""
for i = 1, 1000 do
  s = s .. tostring(i) .. ","
end

-- Fast: Use table.concat
local parts = {}
for i = 1, 1000 do
  table.insert(parts, tostring(i))
end
local s = table.concat(parts, ",")
```

## Script File Organization

For complex scripts, use external files:

```yaml
nodes:
  - name: process
    type: lua
    config:
      file: "scripts/order_processor.lua"
```

Organize your scripts:
```
~/.pocket/scripts/
  ├── filters/
  │   ├── active_users.lua
  │   └── high_value_orders.lua
  ├── transforms/
  │   ├── normalize_data.lua
  │   └── calculate_metrics.lua
  └── validators/
      ├── order_validator.lua
      └── user_validator.lua
```

## See Also

- [Built-in Nodes Reference](../NODE_TYPES.md) - All available node types
- [Examples](examples/lua/) - Complete Lua script examples
- [Lua 5.2 Reference Manual](https://www.lua.org/manual/5.2/) - Official Lua documentation