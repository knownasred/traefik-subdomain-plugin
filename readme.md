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

## Plugin image for Kubernetes

[Plugin image](.github/workflows/image.yml) builds a minimal `scratch` image for
Linux amd64 and arm64. Its root contains the Go source, `go.mod`, `.traefik.yml`,
license, and icon. Traefik loads the source with Yaegi; the image has no executable
or entrypoint. Tests and development files are excluded.

Pushes to `master` publish to `ghcr.io/knownasred/traefik-subdomain-plugin` with
`latest`, `master`, and `sha-<full-commit>` tags. Pushing a `v*` tag also publishes
that exact tag (for example, `v1.0.0`) and a commit tag. Pull requests build without
publishing; the workflow can also be run manually. Publishing uses the built-in
`GITHUB_TOKEN` with `packages: write`, so no registry secret is needed.
Forks publish under their own repository name.

Merge [the image volume example](examples/kubernetes-image-volume.yml) into your
existing Traefik Deployment. It mounts the image root at
`/plugins-local/src/github.com/knownasred/traefik-subdomain-plugin` and enables
the `kepaTenant` local plugin. Keep your existing Traefik arguments and volumes.
If you use a static config file, the equivalent plugin settings are in
[traefik-static.yml](examples/traefik-static.yml). The middleware in
[kubernetes.yml](examples/kubernetes.yml) then uses that plugin.

Your cluster and container runtime must support
[Kubernetes image volumes](https://kubernetes.io/docs/tasks/configure-pod-container/image-volumes/),
with the `ImageVolume` feature enabled where required. The volume is read-only.
Make the GHCR package public for anonymous pulls, or configure the Traefik Pod's
`imagePullSecrets` for a private package. For reproducible deployments, replace
`latest` with the image digest (`ghcr.io/knownasred/traefik-subdomain-plugin@sha256:...`)
reported by the build. Updating a tag does not update running Pods; recreate them
to mount a new image.

To build the file image locally:

```sh
docker build -t kepa-plugin:local .
```

## Development

Implementation: `demo.go` (retaining the template filename). Only the standard
library is used; configuration is validated and the regex compiled at startup.
The Go package is named `traefik_subdomain_plugin`, matching the module's final
path component with hyphens replaced by underscores, as Traefik's loader expects.
Tests import it without an alias so a package-name mismatch fails validation.
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
