/**
 * Javy wrapper for Pocket plugins
 * 
 * This module provides synchronous wrappers for async plugin operations
 * to work with Javy's synchronous execution model.
 */

import { Plugin, Request, Response } from './index';

declare const Javy: {
  IO: {
    readSync(fd: number): Uint8Array;
    writeSync(fd: number, data: Uint8Array): void;
  };
};

let pluginInstance: Plugin | null = null;

/**
 * Initialize the plugin for Javy runtime
 */
export function initializeJavyPlugin(plugin: Plugin): void {
  pluginInstance = plugin;
}

/**
 * Synchronous call handler for Javy
 * 
 * Note: This uses a simplified synchronous approach.
 * Complex async operations may need to be restructured.
 */
function callSync(request: Request): Response {
  if (!pluginInstance) {
    return {
      success: false,
      error: 'Plugin not initialized',
    };
  }

  try {
    const node = pluginInstance.nodes.get(request.node);
    if (!node) {
      return {
        success: false,
        error: `Node not found: ${request.node}`,
      };
    }

    // Create a simple synchronous store
    const store = {
      data: new Map<string, any>(),
      get(key: string): any | undefined {
        return this.data.get(key);
      },
      set(key: string, value: any): void {
        this.data.set(key, value);
      },
      delete(key: string): boolean {
        return this.data.delete(key);
      },
      clear(): void {
        this.data.clear();
      },
    };

    // For Javy, we need to handle async operations differently
    // This is a simplified approach - real implementation might need
    // to restructure async operations or use a different pattern
    switch (request.function) {
      case 'prep': {
        // Assume prep is synchronous for this example
        const prepResult = (node as any).prepSync?.(
          request.input,
          request.config,
          store
        );
        return { success: true, output: prepResult };
      }

      case 'exec': {
        const execResult = (node as any).execSync?.(
          request.prepResult,
          request.config
        );
        return { success: true, output: execResult };
      }

      case 'post': {
        const postResult = (node as any).postSync?.(
          request.input,
          request.prepResult,
          request.execResult,
          request.config,
          store
        );
        return {
          success: true,
          output: postResult.output,
          next: postResult.next,
        };
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

// Javy entry point - reads from stdin and writes to stdout
if (typeof Javy !== 'undefined') {
  try {
    // Read input from stdin
    const input = Javy.IO.readSync(0);
    const inputStr = new TextDecoder().decode(input);
    
    if (!inputStr || inputStr.trim() === '') {
      // No input, return metadata
      const metadata = pluginInstance ? pluginInstance.metadata : {};
      const output = JSON.stringify(metadata);
      Javy.IO.writeSync(1, new TextEncoder().encode(output));
    } else {
      // Process request
      const request = JSON.parse(inputStr) as Request;
      const response = callSync(request);
      const output = JSON.stringify(response);
      Javy.IO.writeSync(1, new TextEncoder().encode(output));
    }
  } catch (error) {
    const errorResponse = {
      success: false,
      error: error instanceof Error ? error.message : String(error),
    };
    Javy.IO.writeSync(1, new TextEncoder().encode(JSON.stringify(errorResponse)));
  }
}

/**
 * Export functions for direct calling (used by non-Javy environments)
 */
export const javyExports = {
  call: callSync,
  getMetadata: () => pluginInstance?.metadata || {},
};