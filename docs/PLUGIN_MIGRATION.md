# Plugin Migration Guide

Guide for migrating from built-in nodes to plugins or between plugin versions.

## From Built-in Nodes to Plugins

### Why Migrate?

- **Language Choice**: Use TypeScript, Rust, or Go instead of only Go
- **Isolation**: Plugins run in sandboxed environments
- **Distribution**: Share plugins without modifying Pocket core
- **Hot Reload**: Update plugins without restarting (future feature)

### Migration Steps

1. **Identify Node Logic**

   Built-in node:
   ```go
   node := pocket.NewNode[Input, Output]("processor",
       pocket.WithExec(func(ctx context.Context, in Input) (Output, error) {
           // Your logic here
           return processData(in), nil
       }),
   )
   ```

2. **Create Plugin Structure**

   ```bash
   my-plugin/
   ├── manifest.yaml
   ├── src/
   │   └── index.ts
   ├── package.json
   └── tsconfig.json
   ```

3. **Port Logic to Plugin**

   TypeScript plugin:
   ```typescript
   class ProcessorNode extends PluginNode<Input, Output, Config> {
     readonly type = 'processor';
     
     async exec(prepData: any, config: Config): Promise<Output> {
       // Your logic here (ported from Go)
       return processData(prepData);
     }
   }
   ```

4. **Define Manifest**

   ```yaml
   name: my-processor
   version: 1.0.0
   runtime: wasm
   binary: plugin.wasm
   
   nodes:
     - type: processor
       category: transform
       description: Process data
   ```

5. **Build and Test**

   ```bash
   npm run build
   pocket-plugins validate .
   pocket-plugins install .
   ```

## Plugin Version Migration

### Semantic Versioning

Follow semantic versioning for plugins:
- **Major**: Breaking changes (3.0.0)
- **Minor**: New features, backward compatible (2.1.0)
- **Patch**: Bug fixes (2.0.1)

### Breaking Changes

When making breaking changes:

1. **Update Version**:
   ```yaml
   # Old: version: 1.2.3
   version: 2.0.0
   ```

2. **Document Changes**:
   ```markdown
   ## Breaking Changes in v2.0.0
   
   - Changed input schema: `data` field is now required
   - Renamed config option `enableCache` to `cache.enabled`
   - Output format changed from array to object
   ```

3. **Provide Migration Path**:
   ```typescript
   // Support old format temporarily
   if (Array.isArray(input)) {
     console.warn('Array input is deprecated, use {data: [...]}');
     input = { data: input };
   }
   ```

### Backward Compatibility

Maintain compatibility when possible:

```typescript
class MyNode extends PluginNode {
  async prep(input: any, config: any, store: Store) {
    // Handle both old and new formats
    const normalizedInput = this.normalizeInput(input);
    const normalizedConfig = this.normalizeConfig(config);
    
    return { input: normalizedInput, config: normalizedConfig };
  }
  
  private normalizeInput(input: any): NewInput {
    // Handle v1 format
    if (input.oldField !== undefined) {
      return {
        newField: input.oldField,
        // ... map other fields
      };
    }
    
    // Already v2 format
    return input;
  }
}
```

## Schema Evolution

### Adding Fields

Safe to add optional fields:

```yaml
# v1.0.0
inputSchema:
  type: object
  properties:
    text:
      type: string
  required: ["text"]

# v1.1.0 - Safe addition
inputSchema:
  type: object
  properties:
    text:
      type: string
    language:  # New optional field
      type: string
      default: "en"
  required: ["text"]
```

### Changing Types

Requires major version bump:

```yaml
# v1.x.x
outputSchema:
  type: array
  items:
    type: string

# v2.0.0 - Breaking change
outputSchema:
  type: object
  properties:
    results:
      type: array
    metadata:
      type: object
```

### Deprecation Strategy

1. **Mark Deprecated** (minor version):
   ```typescript
   if (config.oldOption !== undefined) {
     console.warn('config.oldOption is deprecated, use newOption');
     config.newOption = config.oldOption;
   }
   ```

2. **Remove in Major Version**:
   ```typescript
   // v2.0.0 - Removed support for oldOption
   if (config.oldOption !== undefined) {
     throw new Error('config.oldOption was removed in v2.0.0, use newOption');
   }
   ```

## Testing Migration

### Test Suite

Create tests for both versions:

```typescript
describe('Plugin Migration', () => {
  it('handles v1 input format', () => {
    const v1Input = ['item1', 'item2'];
    const result = plugin.call({
      node: 'processor',
      function: 'prep',
      input: v1Input
    });
    expect(result.success).toBe(true);
  });
  
  it('handles v2 input format', () => {
    const v2Input = { data: ['item1', 'item2'] };
    const result = plugin.call({
      node: 'processor',
      function: 'prep',
      input: v2Input
    });
    expect(result.success).toBe(true);
  });
});
```

### Compatibility Matrix

Document supported versions:

| Plugin Version | Pocket Version | Notes |
|---------------|----------------|-------|
| 1.0.0 - 1.5.0 | >= 1.0.0      | Initial release |
| 2.0.0 - 2.x.x | >= 1.2.0      | Breaking changes in schema |
| 3.0.0+        | >= 2.0.0      | Requires new plugin API |

## Migration Tools

### Schema Converter

Create migration utilities:

```typescript
export class MigrationHelper {
  static migrateV1ToV2(v1Data: V1Format): V2Format {
    return {
      version: '2.0.0',
      data: this.transformData(v1Data),
      metadata: this.generateMetadata(v1Data)
    };
  }
  
  static validateMigration(original: V1Format, migrated: V2Format): boolean {
    // Verify no data loss
    return this.compareData(original, migrated.data);
  }
}
```

### Batch Migration

For multiple workflows:

```bash
# Future CLI feature
pocket-plugins migrate ./workflows --from-version 1.0 --to-version 2.0
```

## Best Practices

1. **Version Lock**: Lock plugin versions in production
2. **Test Thoroughly**: Test migration paths before deploying
3. **Gradual Rollout**: Migrate workflows incrementally
4. **Rollback Plan**: Keep old plugin versions available
5. **Clear Documentation**: Document all breaking changes
6. **Migration Scripts**: Provide automated migration when possible
7. **Deprecation Warnings**: Give users time to migrate

## Common Issues

### "Plugin not found"
- Check version compatibility
- Verify installation path
- Confirm manifest version

### "Schema validation failed"
- Review schema changes
- Check required fields
- Validate data types

### "Timeout exceeded"
- New version may be slower
- Adjust timeout in manifest
- Optimize algorithm

## Support

- File issues: [GitHub Issues](https://github.com/agentstation/pocket/issues)
- Community: [Discord](https://discord.gg/pocket) (future)
- Documentation: [Plugin Docs](./PLUGINS.md)