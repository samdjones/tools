# docker-tools

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

## yolo-run.sh / Containerfile.yolo

Run a Claude Code agent with permission prompts disabled (`--dangerously-skip-permissions`)
inside a locked-down rootless Podman container.

```sh
./yolo-run.sh ~/code/project                 # agent in that dir (builds image on first use)
./yolo-run.sh -p 3000 ~/code/project         # also expose container :3000 on host 127.0.0.1:3000
./yolo-run.sh . -- bash                      # shell in the sandbox instead of the agent
./yolo-run.sh --rebuild [--browser] .        # new Claude Code release / add Chromium
```

First run: `./yolo-run.sh -- claude`, then `/login` (or export `ANTHROPIC_API_KEY`). The agent's
home persists at `~/.local/share/yolo-agent/home`; your real `$HOME` is never mounted.

| Concern | Control |
|---|---|
| Filesystem | Only `DIR` (and a worktree's shared `.git`) is writable; read-only root fs |
| Git | `.git/hooks` and `.git/config` read-only; no SSH keys/credentials, so no push |
| Privileges | non-root, `--cap-drop=ALL`, `no-new-privileges`, memory/CPU/PID limits |
| Network | outbound internet only; host loopback + gateway unreachable; `--offline` for none |
| Local debugging | servers run inside the container on `localhost`; `-p` to view from host |

Known gap: other LAN hosts are reachable (pasta can't filter them). Base image is pinned by
digest in `Containerfile.yolo`.
