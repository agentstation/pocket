/**
 * Pocket Plugin SDK for TypeScript
 * 
 * This SDK provides the base classes and interfaces for building
 * Pocket plugins in TypeScript that compile to WebAssembly.
 */

// Plugin metadata types
export interface PluginMetadata {
  name: string;
  version: string;
  description: string;
  author: string;
  license?: string;
  nodes: NodeDefinition[];
  permissions?: Permissions;
  requirements?: Requirements;
}

export interface NodeDefinition {
  type: string;
  category: string;
  description: string;
  configSchema?: object;
  inputSchema?: object;
  outputSchema?: object;
  examples?: Example[];
}

export interface Example {
  name: string;
  description?: string;
  config?: any;
  input?: any;
  output?: any;
}

export interface Permissions {
  network?: string[];
  env?: string[];
  filesystem?: string[];
  memory?: string;
  cpu?: string;
  timeout?: number;
}

export interface Requirements {
  pocket?: string;
  memory?: string;
}

// Request/Response types
export interface Request {
  node: string;
  function: 'prep' | 'exec' | 'post';
  config?: any;
  input?: any;
  prepResult?: any;
  execResult?: any;
}

export interface Response {
  success: boolean;
  error?: string;
  output?: any;
  next?: string;
}

// Node context interface
export interface NodeContext {
  config: any;
  store: Store;
}

// Store interface (read-only for prep, read-write for post)
export interface Store {
  get(key: string): any | undefined;
  set?(key: string, value: any): void;
  delete?(key: string): void;
}

// Base class for plugin nodes
export abstract class PluginNode<TInput = any, TOutput = any, TConfig = any> {
  abstract readonly type: string;
  abstract readonly category: string;
  abstract readonly description: string;
  
  readonly configSchema?: object;
  readonly inputSchema?: object;
  readonly outputSchema?: object;
  readonly examples?: Example[];

  /**
   * Prep phase - prepare and validate data
   * Has read-only access to store
   */
  async prep(input: TInput, config: TConfig, store: Store): Promise<any> {
    // Default implementation - just return input
    return input;
  }

  /**
   * Exec phase - main processing logic
   * No store access - pure function
   */
  abstract async exec(prepResult: any, config: TConfig): Promise<TOutput>;

  /**
   * Post phase - finalize and route
   * Has read-write access to store
   */
  async post(
    input: TInput,
    prepResult: any,
    execResult: TOutput,
    config: TConfig,
    store: Store
  ): Promise<{ output: TOutput; next: string }> {
    // Default implementation - return result and route to "done"
    return { output: execResult, next: 'done' };
  }
}

// Plugin class that manages nodes
export class Plugin {
  private nodes: Map<string, PluginNode> = new Map();
  
  constructor(private metadata: PluginMetadata) {}

  /**
   * Register a node with the plugin
   */
  register(node: PluginNode): void {
    this.nodes.set(node.type, node);
  }

  /**
   * Get plugin metadata
   */
  getMetadata(): PluginMetadata {
    // Update node definitions from registered nodes
    this.metadata.nodes = Array.from(this.nodes.values()).map(node => ({
      type: node.type,
      category: node.category,
      description: node.description,
      configSchema: node.configSchema,
      inputSchema: node.inputSchema,
      outputSchema: node.outputSchema,
      examples: node.examples,
    }));
    
    return this.metadata;
  }

  /**
   * Handle a request from the host
   */
  async handleRequest(request: Request): Promise<Response> {
    try {
      const node = this.nodes.get(request.node);
      if (!node) {
        return {
          success: false,
          error: `Unknown node type: ${request.node}`,
        };
      }

      // Create store (implementation depends on host bindings)
      const store = createStore();

      switch (request.function) {
        case 'prep': {
          const output = await node.prep(request.input, request.config, store);
          return { success: true, output };
        }

        case 'exec': {
          const output = await node.exec(request.prepResult, request.config);
          return { success: true, output };
        }

        case 'post': {
          const { output, next } = await node.post(
            request.input,
            request.prepResult,
            request.execResult,
            request.config,
            store
          );
          return { success: true, output, next };
        }

        default:
          return {
            success: false,
            error: `Unknown function: ${request.function}`,
          };
      }
    } catch (error) {
      return {
        success: false,
        error: error instanceof Error ? error.message : String(error),
      };
    }
  }
}

// Memory management for WASM
let memory: ArrayBuffer;
let allocatedChunks: Map<number, number> = new Map();

/**
 * Allocate memory for data transfer
 */
export function __pocket_alloc(size: number): number {
  if (!memory) {
    // Initialize with 1MB
    memory = new ArrayBuffer(1024 * 1024);
  }

  // Find free space (simple implementation)
  let ptr = 0;
  for (const [start, length] of allocatedChunks) {
    if (start > ptr + size) {
      break;
    }
    ptr = start + length;
  }

  if (ptr + size > memory.byteLength) {
    // Grow memory
    const newSize = Math.max(memory.byteLength * 2, ptr + size);
    const newMemory = new ArrayBuffer(newSize);
    new Uint8Array(newMemory).set(new Uint8Array(memory));
    memory = newMemory;
  }

  allocatedChunks.set(ptr, size);
  return ptr;
}

/**
 * Free allocated memory
 */
export function __pocket_free(ptr: number, size: number): void {
  allocatedChunks.delete(ptr);
}

/**
 * Main entry point for plugin calls
 */
export function __pocket_call(ptr: number, len: number): [number, number] {
  // Read input
  const input = new Uint8Array(memory, ptr, len);
  const decoder = new TextDecoder();
  const requestStr = decoder.decode(input);
  const request: Request = JSON.parse(requestStr);

  // Handle request
  const responsePromise = globalPlugin?.handleRequest(request) ?? 
    Promise.resolve({ success: false, error: 'Plugin not initialized' });

  // Convert response to bytes
  return responsePromise.then(response => {
    const encoder = new TextEncoder();
    const responseStr = JSON.stringify(response);
    const responseBytes = encoder.encode(responseStr);

    // Allocate memory for response
    const responsePtr = __pocket_alloc(responseBytes.length);
    new Uint8Array(memory, responsePtr, responseBytes.length).set(responseBytes);

    return [responsePtr, responseBytes.length] as [number, number];
  }).catch(error => {
    const errorResponse: Response = {
      success: false,
      error: error instanceof Error ? error.message : String(error),
    };
    
    const encoder = new TextEncoder();
    const responseStr = JSON.stringify(errorResponse);
    const responseBytes = encoder.encode(responseStr);

    const responsePtr = __pocket_alloc(responseBytes.length);
    new Uint8Array(memory, responsePtr, responseBytes.length).set(responseBytes);

    return [responsePtr, responseBytes.length] as [number, number];
  });
}

// Global plugin instance
let globalPlugin: Plugin | null = null;

/**
 * Initialize the plugin
 */
export function initializePlugin(plugin: Plugin): void {
  globalPlugin = plugin;
}

// Store implementation (placeholder - actual implementation depends on host bindings)
function createStore(): Store {
  const data = new Map<string, any>();
  
  return {
    get(key: string): any | undefined {
      return data.get(key);
    },
    set(key: string, value: any): void {
      data.set(key, value);
    },
    delete(key: string): void {
      data.delete(key);
    },
  };
}

// Export WASM memory
export { memory };

// Helper utilities
export * from './utils';