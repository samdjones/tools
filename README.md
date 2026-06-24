# tools

## pin-digest.sh

Resolve a Docker image tag alias to its immutable `sha256` digest for deterministic,
auditable builds. Requires only [skopeo](https://github.com/containers/skopeo).

A floating tag such as `gcr.io/distroless/python3-debian13:nonroot` moves over time, so
the same Dockerfile can pull different base images on different days. Pinning the digest
makes the build reproducible; the resolved digest is the **multi-arch index** digest, so
cross-platform builds still select the correct image.

```sh
# Resolve a tag to a ready-to-paste reference
./pin-digest.sh gcr.io/distroless/python3-debian13:nonroot
# -> gcr.io/distroless/python3-debian13:nonroot@sha256:886011…

# Just the digest
./pin-digest.sh --digest-only gcr.io/distroless/python3-debian13:nonroot

# Also show image metadata (created time, platform, OCI labels) on stderr
./pin-digest.sh --meta gcr.io/distroless/python3-debian13:nonroot
```

Use the result in a Dockerfile, keeping the tag readable while the digest enforces
determinism:

```dockerfile
FROM gcr.io/distroless/python3-debian13:nonroot@sha256:886011…
```

Pin with `@`, not `:` — a bare `:sha256:…` would be parsed as a tag and fail. `--meta`
prints to stderr (creation time, platform, and labels such as
`org.opencontainers.image.{version,revision,source}`), so stdout stays a clean pasteable
reference; some images (e.g. distroless) deliberately zero the timestamp for
reproducibility, which is flagged. Audit history is your Dockerfile's git diff.
