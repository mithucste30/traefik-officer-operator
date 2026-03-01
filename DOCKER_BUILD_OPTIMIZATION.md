# Docker Build Optimization

## Overview

This project uses an optimized Docker build strategy that significantly reduces build times, especially for multi-architecture builds.

## Strategy

### Traditional Approach (Slow)
```
GoReleaser builds binaries → Docker compiles Go code for each platform
- AMD64: ~2 minutes (fast cross-compilation)
- ARM64: ~10+ minutes (slow QEMU emulation)
Total: ~12+ minutes
```

### Optimized Approach (Fast)
```
GoReleaser builds binaries → Docker copies pre-built binaries
- AMD64: ~30 seconds (just file copy)
- ARM64: ~2-3 minutes (QEMU + file copy)
Total: ~3-4 minutes (~70% faster!)
```

## How It Works

1. **GoReleaser Phase**
   - Cross-compiles binaries for all platforms (linux/amd64, linux/arm64, darwin, windows)
   - Uses Go's native cross-compilation (very fast)
   - Creates standalone binaries

2. **Docker Build Phase**
   - Uses `Dockerfile.releaser` instead of `Dockerfile`
   - Simply `COPY` the pre-built binary into the image
   - No compilation, no Go installation needed
   - Minimal layers, faster builds

## Dockerfiles

### For Development (Local Builds)
- `Dockerfile` - Full build from source
- `operator/Dockerfile` - Full build from source
- **Use these for**: Local development, testing

### For Releases (CI/CD)
- `Dockerfile.releaser` - Copy pre-built binary
- `operator/Dockerfile.releaser` - Copy pre-built binary
- **Use these for**: Production releases (managed by GoReleaser)

## Benefits

1. **Speed**: ~70% faster multi-arch builds
2. **Consistency**: Same binary in all artifacts (release archives + Docker images)
3. **Efficiency**: No redundant compilation
4. **Smaller Images**: No build tools in final images
5. **Better Caching**: Binary built once, reused in all images

## Example Timings

### Before Optimization
```
GoReleaser binaries:    1m 30s
Docker AMD64 build:     2m 00s
Docker ARM64 build:    10m 30s
─────────────────────────────
Total:                 ~14m
```

### After Optimization
```
GoReleaser binaries:    1m 30s
Docker AMD64 build:     0m 30s
Docker ARM64 build:     2m 30s
─────────────────────────────
Total:                  ~4m 30s (68% faster!)
```

## Configuration

GoReleaser is configured in `.goreleaser.yml`:
- `builds` section: Defines how to compile binaries
- `dockers` section: References `Dockerfile.releaser`
- `use: buildx`: Enables multi-arch builds
- `skip_build: true`: Skips in-Docker compilation

## Testing Locally

To test the release Dockerfile locally:

```bash
# Build binary locally
go build -o traefik-officer ./cmd/traefik-officer

# Build Docker image using release Dockerfile
docker build -f Dockerfile.releaser -t traefik-officer:test .

# Test the image
docker run --rm traefik-officer:test --version
```

## CI/CD Integration

The GitHub Actions workflow automatically:
1. Triggers on tag push (e.g., `v2.5.1`)
2. Runs GoReleaser to build binaries and Docker images
3. Creates GitHub release with all artifacts
4. Pushes multi-arch Docker images to registry

## Future Improvements

- Consider using `dockers_v2` when it becomes stable (new GoReleaser format)
- Add more platforms (ppc64le, s390x) if needed
- Implement SBOM (Software Bill of Materials) generation
