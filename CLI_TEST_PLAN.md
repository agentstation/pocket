# Pocket Unified CLI Test Plan

## Overview

This document outlines the comprehensive testing strategy for the unified Pocket CLI. The plan ensures that the migration from two binaries to one maintains functionality, performance, and user experience.

## Test Categories

### 1. Unit Tests

#### Command Parsing Tests
- Verify each command is parsed correctly
- Test flag combinations and validation
- Ensure error messages for invalid inputs

```go
func TestRunCommandParsing(t *testing.T) {
    tests := []struct {
        name    string
        args    []string
        wantErr bool
        flags   map[string]interface{}
    }{
        {
            name: "basic run",
            args: []string{"run", "workflow.yaml"},
            wantErr: false,
            flags: map[string]interface{}{
                "verbose": false,
                "dry-run": false,
            },
        },
        {
            name: "run with flags",
            args: []string{"run", "workflow.yaml", "--verbose", "--dry-run"},
            wantErr: false,
            flags: map[string]interface{}{
                "verbose": true,
                "dry-run": true,
            },
        },
    }
}
```

#### Command Execution Tests
- Mock dependencies (file system, plugin loader)
- Test command logic in isolation
- Verify output formatting

### 2. Integration Tests

#### End-to-End Workflow Tests
```go
func TestRunWorkflowIntegration(t *testing.T) {
    // Create test workflow file
    workflow := `
name: test-workflow
start: echo
nodes:
  - name: echo
    type: echo
    config:
      message: "test"
`
    // Run command
    output, err := executeCommand("run", tempFile)
    assert.NoError(t, err)
    assert.Contains(t, output, "test")
}
```

#### Plugin Management Tests
- Install/remove plugins with real files
- Verify plugin discovery and loading
- Test plugin execution through CLI

### 3. Backward Compatibility Tests

#### Legacy Command Formats
```go
func TestBackwardCompatibility(t *testing.T) {
    // Old format should still work
    output1, _ := executeCommand("run", "workflow.yaml", "-v")
    output2, _ := executeCommand("run", "workflow.yaml", "--verbose")
    assert.Equal(t, output1, output2)
}
```

#### Migration Path Tests
- Verify deprecation warnings appear correctly
- Test that old pocket-plugins commands map to new ones
- Ensure output format remains consistent

### 4. Performance Tests

#### Command Startup Time
```go
func BenchmarkCLIStartup(b *testing.B) {
    for i := 0; i < b.N; i++ {
        cmd := exec.Command("pocket", "version")
        cmd.Run()
    }
}
```

#### Binary Size Verification
```bash
# Test script to verify binary size
MAX_SIZE=30000000  # 30MB limit
ACTUAL_SIZE=$(stat -f%z pocket || stat -c%s pocket)
if [ $ACTUAL_SIZE -gt $MAX_SIZE ]; then
    echo "Binary too large: $ACTUAL_SIZE bytes"
    exit 1
fi
```

### 5. User Experience Tests

#### Help Text Validation
- Verify all commands have help text
- Check examples are provided
- Ensure consistency in terminology

```go
func TestHelpText(t *testing.T) {
    commands := []string{"run", "nodes", "scripts", "plugins"}
    for _, cmd := range commands {
        output, _ := executeCommand(cmd, "--help")
        assert.Contains(t, output, "Usage:")
        assert.Contains(t, output, "Examples:")
        assert.Contains(t, output, "Flags:")
    }
}
```

#### Error Message Quality
```go
func TestErrorMessages(t *testing.T) {
    tests := []struct {
        name     string
        args     []string
        wantErr  string
    }{
        {
            name:    "missing workflow file",
            args:    []string{"run"},
            wantErr: "Error: requires a workflow file",
        },
        {
            name:    "invalid workflow file",
            args:    []string{"run", "nonexistent.yaml"},
            wantErr: "Error: workflow file not found",
        },
    }
}
```

## Test Data Structure

```
cmd/pocket/testdata/
├── workflows/
│   ├── simple.yaml         # Basic workflow
│   ├── complex.yaml        # Multi-node workflow
│   ├── invalid.yaml        # Invalid syntax
│   └── plugin-based.yaml   # Uses plugins
├── scripts/
│   ├── valid.lua          # Valid Lua script
│   ├── invalid.lua        # Syntax errors
│   └── with-metadata.lua  # Full metadata
├── plugins/
│   ├── test-plugin/       # Mock plugin
│   │   ├── manifest.yaml
│   │   └── plugin.wasm
│   └── invalid-plugin/    # Invalid plugin
└── configs/
    ├── default.yaml       # Default config
    └── custom.yaml        # Custom settings
```

## Test Scenarios

### Scenario 1: New User Experience
1. Install pocket
2. Run `pocket` without arguments → See helpful usage
3. Run `pocket run examples/hello.yaml` → Success
4. Run `pocket nodes` → See available nodes
5. Run `pocket plugins list` → See no plugins
6. Install a plugin → Verify success
7. Use plugin in workflow → Verify execution

### Scenario 2: Migration from pocket-plugins
1. User has existing plugins installed
2. Run old `pocket-plugins list` → See deprecation warning
3. Run new `pocket plugins list` → See same plugins
4. Verify plugins still work in workflows
5. Test all migrated commands work identically

### Scenario 3: Script Developer Workflow
1. Create Lua script in ~/.pocket/scripts/
2. Run `pocket scripts` → See new script
3. Run `pocket scripts validate <script>` → Pass
4. Run `pocket scripts run <script>` → Execute
5. Use script in workflow → Verify integration

### Scenario 4: CI/CD Integration
1. Run `pocket run workflow.yaml --dry-run` → Validate only
2. Check exit codes for success/failure
3. Parse JSON output with `--output json`
4. Verify quiet mode for automated systems

## Automated Test Suite

### Continuous Integration Tests
```yaml
# .github/workflows/cli-tests.yml
name: CLI Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ${{ matrix.os }}
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
        go: [1.21, 1.22]
    
    steps:
    - name: Unit Tests
      run: go test ./cmd/pocket/...
    
    - name: Integration Tests
      run: go test -tags=integration ./cmd/pocket/...
    
    - name: Build Binary
      run: make build
    
    - name: Smoke Tests
      run: |
        ./pocket version
        ./pocket nodes
        ./pocket run examples/hello.yaml
    
    - name: Binary Size Check
      run: scripts/check-binary-size.sh
```

### Local Test Commands
```bash
# Run all tests
make test

# Run specific test categories
make test-unit
make test-integration
make test-compatibility
make test-performance

# Run with coverage
make test-coverage

# Run specific test
go test -run TestRunCommand ./cmd/pocket/
```

## Test Coverage Goals

- Unit Tests: >90% coverage
- Integration Tests: Cover all major workflows
- Command Coverage: 100% of commands tested
- Error Paths: >80% of error cases tested
- Platform Coverage: Linux, macOS, Windows

## Regression Test Suite

### Critical Path Tests
These tests MUST pass before any release:

1. **Basic Workflow Execution**
   ```bash
   pocket run examples/hello.yaml
   pocket run examples/conditional.yaml
   pocket run examples/parallel.yaml
   ```

2. **Node Discovery**
   ```bash
   pocket nodes | grep -c "echo"  # Should be > 0
   pocket nodes info echo         # Should show details
   ```

3. **Script Management**
   ```bash
   pocket scripts                 # List scripts
   pocket scripts validate <path> # Validate syntax
   ```

4. **Plugin Operations**
   ```bash
   pocket plugins list            # Show plugins
   pocket plugins install <path>  # Install plugin
   pocket plugins remove <name>   # Remove plugin
   ```

## Manual Testing Checklist

### Pre-Release Testing
- [ ] Install from source (`go install`)
- [ ] Run all example workflows
- [ ] Test on fresh system (no existing config)
- [ ] Test migration from pocket-plugins
- [ ] Verify help text is accurate
- [ ] Check binary size is reasonable
- [ ] Test on all supported platforms
- [ ] Verify no missing dependencies
- [ ] Check for data races (`go test -race`)
- [ ] Validate performance benchmarks

### User Acceptance Testing
- [ ] Give to beta users for feedback
- [ ] Document any confusing aspects
- [ ] Verify error messages are helpful
- [ ] Check command response times
- [ ] Test in real-world scenarios

## Success Metrics

1. **All tests pass** on supported platforms
2. **No performance regression** vs. separate binaries
3. **Binary size** < 30MB
4. **Command latency** < 100ms for simple operations
5. **Test coverage** > 85% overall
6. **Zero data races** detected
7. **Backward compatibility** maintained 100%

## Monitoring Post-Release

1. Track error reports in GitHub issues
2. Monitor binary download sizes
3. Collect user feedback on new structure
4. Performance metrics from real usage
5. Success rate of migrations from old CLIs