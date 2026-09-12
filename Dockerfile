# Traefik interprets the source with Yaegi; this image is only a file volume.
FROM scratch
COPY --chmod=0644 go.mod *.go .traefik.yml LICENSE /
COPY --chmod=0644 .assets/ /.assets/
