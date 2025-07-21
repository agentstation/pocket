# Plugin SDK API Reference

Complete API reference for the Pocket Plugin SDK.

## Table of Contents

1. [Core Types](#core-types)
2. [Plugin Class](#plugin-class)
3. [PluginNode Class](#pluginnode-class)
4. [Interfaces](#interfaces)
5. [Memory Management](#memory-management)
6. [Utility Functions](#utility-functions)

## Core Types

### Metadata

Plugin metadata structure:

```typescript
interface Metadata {
  name: string;
  version: string;
  description: string;
  author: string;
  license?: string;
  runtime: 'wasm';
  binary: string;
  nodes: NodeDefinition[];
  permissions: Permissions;
  requirements: Requirements;
}
```

### NodeDefinition

Defines a node type exported by the plugin:

```typescript
interface NodeDefinition {
  type: string;              // Unique node type identifier
  category: string;          // Category (e.g., 'transform', 'ai', 'data')
  description: string;       // Human-readable description
  configSchema?: JSONSchema; // Configuration schema
  inputSchema?: JSONSchema;  // Input data schema
  outputSchema?: JSONSchema; // Output data schema
  examples?: Example[];      // Usage examples
}
```

### Permissions

Security permissions required by the plugin:

```typescript
interface Permissions {
  memory?: string;      // Max memory (e.g., "10MB")
  timeout?: number;     // Max execution time in ms
  env?: string[];       // Allowed environment variables
  filesystem?: string[]; // Allowed filesystem paths
  network?: string[];    // Allowed network endpoints (future)
}
```

### Requirements

Plugin requirements:

```typescript
interface Requirements {
  pocket: string;  // Pocket version requirement (e.g., ">=1.0.0")
  memory?: string; // Minimum memory requirement
  cpu?: string;    // CPU requirements (future)
}
```

## Plugin Class

Main plugin class for registering nodes and metadata:

```typescript
class Plugin {
  readonly metadata: Metadata;
  readonly nodes: Map<string, PluginNode>;

  constructor(metadata: Metadata);

  /**
   * Register a node with the plugin
   */
  register(node: PluginNode): void;

  /**
   * Get a registered node by type
   */
  getNode(type: string): PluginNode | undefined;

  /**
   * Internal: Handle plugin calls from host
   */
  _call(request: Request): Promise<Response>;
}
```

### Usage Example

```typescript
const plugin = new Plugin({
  name: 'my-plugin',
  version: '1.0.0',
  description: 'My awesome plugin',
  author: 'Your Name',
  runtime: 'wasm',
  binary: 'plugin.wasm',
  nodes: [], // Will be populated by register()
  permissions: {
    memory: '10MB',
    timeout: 5000
  },
  requirements: {
    pocket: '>=1.0.0'
  }
});

plugin.register(new MyCustomNode());
plugin.register(new AnotherNode());

initializePlugin(plugin);
```

## PluginNode Class

Base class for implementing plugin nodes:

```typescript
abstract class PluginNode<TInput = any, TOutput = any, TConfig = any> {
  abstract readonly type: string;
  abstract readonly category: string;
  abstract readonly description: string;
  
  readonly configSchema?: JSONSchema;
  readonly inputSchema?: JSONSchema;
  readonly outputSchema?: JSONSchema;
  readonly examples?: Example[];

  /**
   * Preparation phase - validate inputs and prepare data
   */
  abstract prep(
    input: TInput,
    config: TConfig,
    store: Store
  ): Promise<any>;

  /**
   * Execution phase - core business logic
   */
  abstract exec(
    prepResult: any,
    config: TConfig
  ): Promise<TOutput>;

  /**
   * Post-processing phase - handle results and routing
   */
  abstract post(
    input: TInput,
    prepResult: any,
    execResult: TOutput,
    config: TConfig,
    store: Store
  ): Promise<{
    output: TOutput;
    next: string;
  }>;
}
```

### Implementation Example

```typescript
interface MyInput {
  data: string[];
  options: {
    mode: 'fast' | 'accurate';
  };
}

interface MyOutput {
  result: string[];
  metadata: {
    processedCount: number;
    duration: number;
  };
}

interface MyConfig {
  threshold: number;
  enableCache: boolean;
}

class MyNode extends PluginNode<MyInput, MyOutput, MyConfig> {
  readonly type = 'my-processor';
  readonly category = 'transform';
  readonly description = 'Processes data with custom logic';

  readonly configSchema = {
    type: 'object',
    properties: {
      threshold: {
        type: 'number',
        default: 0.5,
        minimum: 0,
        maximum: 1
      },
      enableCache: {
        type: 'boolean',
        default: true
      }
    }
  };

  async prep(input: MyInput, config: MyConfig, store: Store) {
    // Validate input
    if (!Array.isArray(input.data)) {
      throw new Error('Input data must be an array');
    }

    // Check cache if enabled
    if (config.enableCache) {
      const cached = store.get('cache:' + JSON.stringify(input));
      if (cached) {
        return { cached: true, data: cached };
      }
    }

    // Prepare data for processing
    return {
      cached: false,
      data: input.data.filter(item => item.length > 0),
      startTime: Date.now()
    };
  }

  async exec(prepResult: any, config: MyConfig): Promise<MyOutput> {
    const { data, startTime } = prepResult;
    
    // Core processing logic
    const result = data.map(item => 
      processItem(item, config.threshold)
    );

    return {
      result,
      metadata: {
        processedCount: result.length,
        duration: Date.now() - startTime
      }
    };
  }

  async post(
    input: MyInput,
    prepResult: any,
    execResult: MyOutput,
    config: MyConfig,
    store: Store
  ) {
    // Cache result if enabled
    if (config.enableCache && !prepResult.cached) {
      store.set('cache:' + JSON.stringify(input), execResult);
    }

    // Determine routing
    const next = execResult.metadata.processedCount > 0 
      ? 'success' 
      : 'empty';

    return {
      output: execResult,
      next
    };
  }
}
```

## Interfaces

### Store

Key-value store for plugin state:

```typescript
interface Store {
  /**
   * Get a value by key
   */
  get(key: string): any | undefined;

  /**
   * Set a value
   */
  set(key: string, value: any): void;

  /**
   * Delete a key
   */
  delete(key: string): boolean;

  /**
   * Clear all data
   */
  clear(): void;

  /**
   * Check if key exists
   */
  has(key: string): boolean;

  /**
   * Get all keys
   */
  keys(): string[];
}
```

### Request

Internal request structure from host:

```typescript
interface Request {
  node: string;           // Node type
  function: 'prep' | 'exec' | 'post';
  config?: any;           // Node configuration
  input?: any;            // Input for prep
  prepResult?: any;       // Result from prep (for exec)
  execResult?: any;       // Result from exec (for post)
}
```

### Response

Internal response structure to host:

```typescript
interface Response {
  success: boolean;
  output?: any;
  error?: string;
  next?: string;  // Routing decision from post
}
```

### JSONSchema

Standard JSON Schema for validation:

```typescript
interface JSONSchema {
  type?: string | string[];
  properties?: { [key: string]: JSONSchema };
  items?: JSONSchema;
  required?: string[];
  enum?: any[];
  minimum?: number;
  maximum?: number;
  minLength?: number;
  maxLength?: number;
  pattern?: string;
  default?: any;
  description?: string;
  // ... other JSON Schema properties
}
```

### Example

Plugin example structure:

```typescript
interface Example {
  name: string;
  description?: string;
  input: any;
  config?: any;
  output: any;
  next?: string;
}
```

## Memory Management

Functions for WASM memory management:

### __pocket_alloc

Allocate memory for data transfer:

```typescript
export function __pocket_alloc(size: number): number
```

- **Parameters**: 
  - `size`: Number of bytes to allocate
- **Returns**: Pointer to allocated memory
- **Usage**: Called by host to allocate memory for passing data

### __pocket_free

Free allocated memory:

```typescript
export function __pocket_free(ptr: number, size: number): void
```

- **Parameters**:
  - `ptr`: Pointer to memory to free
  - `size`: Size of memory block
- **Usage**: Called by host to free memory after use

### __pocket_call

Main entry point for plugin calls:

```typescript
export function __pocket_call(ptr: number, size: number): number
```

- **Parameters**:
  - `ptr`: Pointer to request data
  - `size`: Size of request data
- **Returns**: Pointer to response data
- **Usage**: Called by host to invoke plugin functionality

## Utility Functions

### initializePlugin

Initialize the plugin for the host environment:

```typescript
function initializePlugin(plugin: Plugin): void
```

- **Parameters**:
  - `plugin`: The plugin instance to initialize
- **Usage**: Must be called after registering all nodes

### deepClone

Deep clone an object:

```typescript
function deepClone<T>(obj: T): T
```

- **Parameters**:
  - `obj`: Object to clone
- **Returns**: Deep copy of the object

### deepMerge

Deep merge objects:

```typescript
function deepMerge<T extends object>(target: T, ...sources: Partial<T>[]): T
```

- **Parameters**:
  - `target`: Target object
  - `sources`: Objects to merge into target
- **Returns**: Merged object

### retry

Retry an operation with exponential backoff:

```typescript
async function retry<T>(
  fn: () => Promise<T>,
  options?: {
    maxAttempts?: number;
    delay?: number;
    backoff?: number;
    onError?: (error: Error, attempt: number) => void;
  }
): Promise<T>
```

- **Parameters**:
  - `fn`: Function to retry
  - `options`: Retry configuration
- **Returns**: Result of successful operation

### debounce

Create a debounced function:

```typescript
function debounce<T extends (...args: any[]) => any>(
  fn: T,
  delay: number
): T & { cancel: () => void }
```

- **Parameters**:
  - `fn`: Function to debounce
  - `delay`: Delay in milliseconds
- **Returns**: Debounced function with cancel method

### throttle

Create a throttled function:

```typescript
function throttle<T extends (...args: any[]) => any>(
  fn: T,
  limit: number
): T
```

- **Parameters**:
  - `fn`: Function to throttle
  - `limit`: Minimum time between calls in ms
- **Returns**: Throttled function

### memoize

Memoize a function:

```typescript
function memoize<T extends (...args: any[]) => any>(
  fn: T,
  keyFn?: (...args: Parameters<T>) => string
): T
```

- **Parameters**:
  - `fn`: Function to memoize
  - `keyFn`: Optional function to generate cache key
- **Returns**: Memoized function

## Error Handling

### PluginError

Base error class for plugin errors:

```typescript
class PluginError extends Error {
  constructor(
    message: string,
    public code?: string,
    public details?: any
  ) {
    super(message);
    this.name = 'PluginError';
  }
}
```

### Common Error Codes

- `INVALID_INPUT`: Input validation failed
- `INVALID_CONFIG`: Configuration validation failed
- `TIMEOUT`: Operation timed out
- `MEMORY_LIMIT`: Memory limit exceeded
- `PERMISSION_DENIED`: Permission not granted
- `NOT_FOUND`: Resource not found
- `INTERNAL_ERROR`: Internal plugin error

### Error Handling Example

```typescript
async prep(input: Input, config: Config, store: Store) {
  try {
    if (!input.data) {
      throw new PluginError(
        'Input data is required',
        'INVALID_INPUT',
        { field: 'data' }
      );
    }

    // Processing...
  } catch (error) {
    if (error instanceof PluginError) {
      throw error;
    }
    
    // Wrap unknown errors
    throw new PluginError(
      'Unexpected error in prep phase',
      'INTERNAL_ERROR',
      { originalError: error.message }
    );
  }
}
```

## Best Practices

1. **Type Safety**: Use TypeScript generics for type-safe implementations
2. **Validation**: Validate all inputs in the prep phase
3. **Pure Exec**: Keep exec phase pure with no side effects
4. **Error Messages**: Provide clear, actionable error messages
5. **Resource Cleanup**: Clean up resources in error cases
6. **Documentation**: Document all config options and schemas
7. **Examples**: Provide comprehensive examples
8. **Testing**: Test all phases independently