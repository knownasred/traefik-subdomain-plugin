# KEPA tenant header middleware

A dependency-free Go/Yaegi middleware for Traefik. It replaces the NGINX
`configuration-snippet`: `alice.app.kepa.ch` becomes `X-Tenant: alice` upstream.

## Configuration

```yaml
spec:
  plugin:
    kepaTenant:
      hostRegex: '^([^.]+)\.app\.kepa\.ch$'
      headerName: X-Tenant
      captureGroup: 1
```

These are the defaults. `captureGroup` is a positive, 1-based regex capture index.
Invalid regexes, capture indices, and header names fail at startup. `Host` cannot
be the destination header. Use Go regex syntax; anchor patterns to the whole host.

The middleware reads the request Host, removes its port, lowercases it, and strips
a trailing DNS dot. It ignores `X-Forwarded-Host`. It overwrites incoming tenant
headers; for unmatched hosts or empty captures it removes the header, matching
NGINX's empty `proxy_set_header` behavior. Other request fields and the response
pass through unchanged. Attach it before authentication middleware that consumes
the tenant header. Applications must still authorize access to the selected tenant.

## NGINX migration

| NGINX annotation | Traefik configuration |
| --- | --- |
| `proxy-body-size: "256m"` | Native `buffering.maxRequestBodyBytes: 268435456` |
| `proxy-buffering: "on"` | Native Buffering middleware, with different buffering behavior |
| `proxy-buffer-size: "32k"` | No direct equivalent. NGINX sizes the first upstream response buffer, including headers; Traefik's `memResponseBodyBytes` controls when response bodies spill to disk. |
| `server-alias: "*.app.kepa.ch"` | Router `HostRegexp` for a single subdomain; configure DNS and TLS separately |
| `rewrite-target: /$2` | Native `replacePathRegex`, using the original path regex and replacement `/${2}` |
| `configuration-snippet` | This plugin |

The original Ingress path was not provided. The example assumes `/api(/|$)(.*)`,
so `/api/items` becomes `/items`. Replace both the rewrite regex and router path
match with your actual path, or omit rewriting if unnecessary. Add any original
non-wildcard hosts to the router separately.

Traefik buffers whole bodies and can spill to disk. Provision temporary disk for
concurrent uploads/responses and review buffering for streaming endpoints.
The sample retains 1 MiB memory thresholds; `32k` is not an equivalent setting.

References: [Buffering](https://doc.traefik.io/traefik/reference/routing-configuration/http/middlewares/buffering/),
[ReplacePathRegex](https://doc.traefik.io/traefik/reference/routing-configuration/http/middlewares/replacepathregex/),
[plugin installation](https://plugins.traefik.io/install).

## Installation

This checkout retains the template module path `github.com/traefik/plugindemo`.
Use the local installation below. Downloading that upstream module would install
the original demo, not this plugin.

1. Mount this checkout at `/plugins-local/src/github.com/traefik/plugindemo` in
   the Traefik container, with Traefik's working directory set to `/`.
2. Merge [examples/traefik-static.yml](examples/traefik-static.yml) into Traefik's
   static configuration and restart it. The plugin key is `kepaTenant`.
3. Install Traefik's Kubernetes CRDs and enable the CRD provider with appropriate
   RBAC. The examples use Traefik v3 rule syntax.
4. Edit [examples/kubernetes.yml](examples/kubernetes.yml) for your namespace,
   Service, port, path, entry point, and TLS secret, then run
   `kubectl apply -f examples/kubernetes.yml`.

For remote installation, change `go.mod`, test imports, `.traefik.yml`, and example
module paths to your actual repository. Publish a version tag and configure
`experimental.plugins.kepaTenant` with that moduleName and version instead of
`experimental.localPlugins`. See the plugin installation reference for catalog
requirements. Nothing has been published or deployed by this change.

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
