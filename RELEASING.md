# Releasing Pocket

This document describes the release process for Pocket maintainers.

## Prerequisites

- Push access to the repository
- `HOMEBREW_TAP_TOKEN` secret configured in GitHub (for automatic Homebrew updates)
- GoReleaser installed locally (for testing): `go install github.com/goreleaser/goreleaser@latest`

## Release Process

### 1. Prepare the Release

1. Ensure all PRs for the release are merged
2. Update the version in any documentation if needed
3. Run tests to ensure everything is working:
   ```bash
   make test-all
   make lint
   ```

### 2. Test the Release Locally

Before creating a real release, test the build process:

```bash
make release-test
```

This will:
- Build binaries for all platforms
- Create archives
- Generate checksums
- Test the GoReleaser configuration

Check the `dist/` directory to verify the artifacts look correct.

### 3. Create the Release

Create and push a version tag:

```bash
# For a new release
make release VERSION=v1.2.3

# For a pre-release
make release VERSION=v1.2.3-beta.1
```

This will:
1. Create an annotated git tag
2. Push the tag to GitHub
3. Trigger the GitHub Actions release workflow

### 4. Monitor the Release

1. Go to the [Actions tab](https://github.com/agentstation/pocket/actions)
2. Watch the "Release" workflow
3. The workflow will:
   - Run tests
   - Build binaries for all platforms
   - Create a GitHub release with:
     - Pre-built binaries for all platforms
     - Source code archives
     - SHA256 checksums
     - Auto-generated changelog
   - Update the Homebrew formula with bottle support (for stable releases only)
   - GoReleaser will automatically:
     - Update version and checksums
     - Add bottle definitions for each platform
     - Commit to homebrew-tap repository

### 5. Verify the Release

Once the workflow completes:

1. Check the [releases page](https://github.com/agentstation/pocket/releases)
2. Verify all artifacts are present
3. Test installation methods:
   ```bash
   # Test Homebrew bottle install (default - pre-built binary)
   brew update
   brew install agentstation/tap/pocket
   
   # Test Homebrew source build
   brew install --build-from-source agentstation/tap/pocket
   
   # Test install script
   curl -sSL https://raw.githubusercontent.com/agentstation/pocket/master/install.sh | bash
   
   # Test direct download
   curl -L https://github.com/agentstation/pocket/releases/latest/download/pocket-darwin-arm64.tar.gz -o test.tar.gz
   ```

### 6. Post-Release

1. Close the milestone for this release
2. Create a new milestone for the next release
3. Update the roadmap if needed
4. Announce the release (optional):
   - Twitter/X
   - Discord/Slack communities
   - Blog post for major releases

## Version Numbering

We follow [Semantic Versioning](https://semver.org/):

- `vMAJOR.MINOR.PATCH` for stable releases
- `vMAJOR.MINOR.PATCH-TAG.N` for pre-releases

Examples:
- `v1.0.0` - First stable release
- `v1.0.1` - Patch release (bug fixes)
- `v1.1.0` - Minor release (new features, backward compatible)
- `v2.0.0` - Major release (breaking changes)
- `v1.1.0-beta.1` - Beta release
- `v1.1.0-rc.1` - Release candidate

## Troubleshooting

### Release workflow fails

1. Check the [Actions logs](https://github.com/agentstation/pocket/actions)
2. Common issues:
   - Tests failing: Fix the tests before releasing
   - GoReleaser errors: Test locally with `make release-test`
   - Homebrew update fails: Check the `HOMEBREW_TAP_TOKEN` secret

### Homebrew formula not updating

The Homebrew formula is only updated for stable releases (not pre-releases).
If it's not updating:

1. Check that the `HOMEBREW_TAP_TOKEN` has write access to `agentstation/homebrew-tap`
2. Verify the release workflow completed successfully
3. Check the [homebrew-tap repository](https://github.com/agentstation/homebrew-tap) for the commit
4. GoReleaser should create a commit with message "Update pocket to vX.Y.Z"

### Homebrew bottle vs source builds

The formula supports both installation methods:
- **Bottles** (default): Pre-built binaries for faster installation
- **Source**: `brew install --build-from-source agentstation/tap/pocket`

If bottles aren't working:
1. Check that GoReleaser created the bottle block in the formula
2. Verify the binary URLs in the formula are correct
3. Test with `--build-from-source` as a fallback

### Need to delete a release

If you need to remove a release:

1. Delete the release on GitHub
2. Delete the tag: `git push origin :refs/tags/vX.Y.Z`
3. Fix the issue
4. Create a new release with a new version number

## Release Checklist

- [ ] All tests passing
- [ ] Changelog updated (if manual changes needed)
- [ ] Documentation updated
- [ ] Version tag created and pushed
- [ ] GitHub Actions workflow successful
- [ ] Release artifacts verified
- [ ] Homebrew formula updated (for stable releases)
- [ ] Installation methods tested
- [ ] Release announced (if applicable)