# tools

## pin-digest.sh

Pin Docker base images to immutable `sha256` digests for deterministic, auditable
builds. Requires only [skopeo](https://github.com/containers/skopeo).

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

# Pin every resolvable FROM line in a Dockerfile, in place
./pin-digest.sh -f Dockerfile

# Preview the changes without writing
./pin-digest.sh -f Dockerfile -n
```

`FROM` lines are rewritten as `repo:tag@sha256:…` — the tag stays human-readable while
the digest enforces determinism. Re-running re-resolves each tag, so the script is
idempotent and can be used to bump base images. `scratch`, build-stage references, and
lines already pinned without a tag are left untouched. Audit history is the Dockerfile's
git diff.
