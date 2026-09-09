# Tenant header middleware

A dependency-free Go/Yaegi middleware for Traefik. It replaces the NGINX
`configuration-snippet`: `alice.app.example.com` becomes `X-Tenant: alice` upstream.

## Configuration

```yaml
spec:
  plugin:
    tenantHeader:
      hostRegex: '^([^.]+)\.app\.example\.com$'
      headerName: X-Tenant
      captureGroup: 1
```

This example uses `app.example.com` as the tenant domain. `captureGroup` is a positive, 1-based regex capture index.
Invalid regexes, capture indices, and header names fail at startup. `Host` cannot
be the destination header. Use Go regex syntax; anchor patterns to the whole host.

The middleware reads the request Host, removes its port, lowercases it, and strips
a trailing DNS dot. It ignores `X-Forwarded-Host`. It overwrites incoming tenant
headers; for unmatched hosts or empty captures it removes the header, matching
NGINX's empty `proxy_set_header` behavior. Other request fields and the response
pass through unchanged. Attach it before authentication middleware that consumes
the tenant header. Applications must still authorize access to the selected tenant.

## Development

Implementation: `demo.go` (retaining the template filename). Only the standard
library is used; configuration is validated and the regex compiled at startup.
Tests cover host normalization, custom configuration, spoofed headers, nonmatches,
invalid configuration, and downstream behavior under both Go and Yaegi.

`make yaegi_test` creates a temporary GOPATH and links this checkout at the module
path from `go.mod`, then removes the temporary directory on exit. Yaegi resolves
test imports through GOPATH, so this also works in forks whose checkout path does
not match the module name, without changing your existing GOPATH.

## Nix development environment

With Nix flakes enabled, enter the pinned development shell:

```sh
nix develop
make test
make yaegi_test
```

The shell provides Go, Yaegi, gopls, and GNU Make from nixpkgs 26.05 on Linux
(x86_64 and aarch64). Yaegi is marked broken on macOS in nixpkgs.
Run `nix flake check` to execute both test targets in an isolated Nix build.
Go uses the toolchain pinned by `flake.lock`, with CGO disabled to match CI.
The shell creates an ignored `.nix-go/` GOPATH with a link to this checkout
so Yaegi can resolve the plugin's imports from any checkout location.
The legacy golangci-lint configuration is not included in this shell;
`make lint` requires a separate compatible installation.

When adding these files to Git, include both `flake.nix` and `flake.lock`
so Nix can discover the flake and reproduce its dependencies.
