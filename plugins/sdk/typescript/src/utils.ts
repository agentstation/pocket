/**
 * Utility functions for plugin development
 */

/**
 * Deep clone an object
 */
export function deepClone<T>(obj: T): T {
  if (obj === null || typeof obj !== 'object') {
    return obj;
  }

  if (obj instanceof Date) {
    return new Date(obj.getTime()) as T;
  }

  if (obj instanceof Array) {
    return obj.map(item => deepClone(item)) as T;
  }

  if (obj instanceof Object) {
    const clonedObj: any = {};
    for (const key in obj) {
      if (Object.prototype.hasOwnProperty.call(obj, key)) {
        clonedObj[key] = deepClone(obj[key]);
      }
    }
    return clonedObj;
  }

  return obj;
}

/**
 * Deep merge objects
 */
export function deepMerge<T extends object>(target: T, ...sources: Partial<T>[]): T {
  if (!sources.length) return target;
  
  const source = sources.shift();
  if (!source) return target;

  for (const key in source) {
    if (Object.prototype.hasOwnProperty.call(source, key)) {
      const sourceValue = source[key];
      const targetValue = target[key];

      if (isObject(sourceValue) && isObject(targetValue)) {
        target[key] = deepMerge(targetValue as any, sourceValue as any);
      } else {
        target[key] = sourceValue as T[Extract<keyof T, string>];
      }
    }
  }

  return deepMerge(target, ...sources);
}

/**
 * Check if value is a plain object
 */
export function isObject(value: any): value is object {
  return value !== null && typeof value === 'object' && !Array.isArray(value);
}

/**
 * Sleep for specified milliseconds
 */
export function sleep(ms: number): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms));
}

/**
 * Retry a function with exponential backoff
 */
export async function retry<T>(
  fn: () => Promise<T>,
  options: {
    attempts?: number;
    delay?: number;
    maxDelay?: number;
    factor?: number;
    onError?: (error: Error, attempt: number) => void;
  } = {}
): Promise<T> {
  const {
    attempts = 3,
    delay = 1000,
    maxDelay = 30000,
    factor = 2,
    onError,
  } = options;

  let lastError: Error;
  let currentDelay = delay;

  for (let attempt = 1; attempt <= attempts; attempt++) {
    try {
      return await fn();
    } catch (error) {
      lastError = error instanceof Error ? error : new Error(String(error));
      
      if (onError) {
        onError(lastError, attempt);
      }

      if (attempt === attempts) {
        throw lastError;
      }

      await sleep(currentDelay);
      currentDelay = Math.min(currentDelay * factor, maxDelay);
    }
  }

  throw lastError!;
}

/**
 * Create a debounced version of a function
 */
export function debounce<T extends (...args: any[]) => any>(
  fn: T,
  delay: number
): (...args: Parameters<T>) => void {
  let timeoutId: NodeJS.Timeout | null = null;

  return (...args: Parameters<T>) => {
    if (timeoutId) {
      clearTimeout(timeoutId);
    }

    timeoutId = setTimeout(() => {
      fn(...args);
      timeoutId = null;
    }, delay);
  };
}

/**
 * Create a throttled version of a function
 */
export function throttle<T extends (...args: any[]) => any>(
  fn: T,
  limit: number
): (...args: Parameters<T>) => void {
  let inThrottle = false;

  return (...args: Parameters<T>) => {
    if (!inThrottle) {
      fn(...args);
      inThrottle = true;
      setTimeout(() => {
        inThrottle = false;
      }, limit);
    }
  };
}

/**
 * Validate data against a JSON schema (simplified version)
 */
export function validateSchema(data: any, schema: any): { valid: boolean; errors: string[] } {
  const errors: string[] = [];

  function validate(value: any, schema: any, path: string = ''): void {
    if (schema.type) {
      const actualType = Array.isArray(value) ? 'array' : typeof value;
      if (schema.type !== actualType) {
        errors.push(`${path || 'root'}: expected ${schema.type}, got ${actualType}`);
        return;
      }
    }

    if (schema.required && Array.isArray(schema.required)) {
      for (const field of schema.required) {
        if (!(field in value)) {
          errors.push(`${path || 'root'}: missing required field '${field}'`);
        }
      }
    }

    if (schema.properties && typeof value === 'object' && !Array.isArray(value)) {
      for (const [key, subSchema] of Object.entries(schema.properties)) {
        if (key in value) {
          validate(value[key], subSchema, path ? `${path}.${key}` : key);
        }
      }
    }

    if (schema.items && Array.isArray(value)) {
      value.forEach((item, index) => {
        validate(item, schema.items, `${path}[${index}]`);
      });
    }

    if (schema.minimum !== undefined && typeof value === 'number') {
      if (value < schema.minimum) {
        errors.push(`${path || 'root'}: value ${value} is less than minimum ${schema.minimum}`);
      }
    }

    if (schema.maximum !== undefined && typeof value === 'number') {
      if (value > schema.maximum) {
        errors.push(`${path || 'root'}: value ${value} is greater than maximum ${schema.maximum}`);
      }
    }

    if (schema.minLength !== undefined && typeof value === 'string') {
      if (value.length < schema.minLength) {
        errors.push(`${path || 'root'}: string length ${value.length} is less than minimum ${schema.minLength}`);
      }
    }

    if (schema.maxLength !== undefined && typeof value === 'string') {
      if (value.length > schema.maxLength) {
        errors.push(`${path || 'root'}: string length ${value.length} is greater than maximum ${schema.maxLength}`);
      }
    }

    if (schema.pattern !== undefined && typeof value === 'string') {
      const regex = new RegExp(schema.pattern);
      if (!regex.test(value)) {
        errors.push(`${path || 'root'}: value does not match pattern ${schema.pattern}`);
      }
    }

    if (schema.enum !== undefined && Array.isArray(schema.enum)) {
      if (!schema.enum.includes(value)) {
        errors.push(`${path || 'root'}: value must be one of ${JSON.stringify(schema.enum)}`);
      }
    }
  }

  validate(data, schema);

  return {
    valid: errors.length === 0,
    errors,
  };
}