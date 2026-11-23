#!/usr/bin/env bash
#
# yolo-run.sh — run a Claude Code agent with permission prompts disabled inside a
# locked-down rootless Podman container. Only the target directory is writable.
#
#   yolo-run.sh [options] [DIR] [-- COMMAND...]
#
set -euo pipefail

IMAGE=${YOLO_IMAGE:-yolo-agent}
HOME_DIR=${YOLO_HOME:-$HOME/.local/share/yolo-agent/home}
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

usage() {
	cat <<'EOF'
yolo-run.sh — run Claude Code (--dangerously-skip-permissions) in a rootless Podman sandbox

Usage:
  yolo-run.sh [options] [DIR] [-- COMMAND...]

  DIR                 Project/worktree to expose read-write (default: current dir).
                      Mounted at the same path inside the container.
  COMMAND             Run instead of claude, e.g.  -- bash   or   -- npm test

Options:
  -p, --publish PORT  Publish container PORT on host 127.0.0.1:PORT for debugging
                      (also HOST:CONTAINER). Repeatable.
      --offline       No network at all.
      --mem SIZE      Memory limit (default 8g)
      --cpus N        CPU limit (default 4)
      --pids N        Process limit (default 1024)
      --rebuild       Rebuild the image (picks up a new Claude Code release)
      --browser       With --rebuild: include Chromium in the image
  -h, --help          Show this help

The image is built automatically on first use.

Isolation:
  * Filesystem: only DIR (+ the repo's shared .git dir for worktrees) is writable;
    root fs is read-only; your $HOME is NOT mounted. The agent has its own home at
    $YOLO_HOME (default ~/.local/share/yolo-agent/home), so log in once there:
        yolo-run.sh -- claude        (then /login)   or export ANTHROPIC_API_KEY
  * Git: .git/hooks and .git/config are mounted read-only (the agent can't plant
    hooks that later run on the host). No SSH keys or git credentials are passed,
    so the agent can commit but not push; push from the host.
  * Privileges: all capabilities dropped, no-new-privileges, non-root, resource limits.
  * Network: full outbound internet (packages, APIs). Host loopback and the host
    gateway are NOT reachable. Servers the agent starts are reachable inside the
    container on localhost; use --publish to view them from the host.
    LAN hosts are still reachable (use --offline if that matters).
EOF
}

die() { echo "yolo-run.sh: $*" >&2; exit 1; }

ports=() offline=0 mem=8g cpus=4 pids=1024 rebuild=0 browser=0 dir="" cmd=()
while (($#)); do
	case $1 in
	-h | --help) usage; exit 0 ;;
	-p | --publish) [[ ${2:-} ]] || die "$1 needs a value"; ports+=("$2"); shift 2 ;;
	--offline) offline=1; shift ;;
	--mem) mem=${2:?}; shift 2 ;;
	--cpus) cpus=${2:?}; shift 2 ;;
	--pids) pids=${2:?}; shift 2 ;;
	--rebuild) rebuild=1; shift ;;
	--browser) browser=1; shift ;;
	--) shift; cmd=("$@"); break ;;
	-*) die "unknown option: $1 (see --help)" ;;
	*) [[ -z $dir ]] || die "unexpected argument: $1"; dir=$1; shift ;;
	esac
done

command -v podman >/dev/null || die "podman not found"

if ((rebuild)) || ! podman image exists "$IMAGE"; then
	echo "Building $IMAGE…" >&2
	podman build -t "$IMAGE" -f "$SCRIPT_DIR/Containerfile.yolo" \
		--build-arg "WITH_BROWSER=$browser" "$SCRIPT_DIR" >&2
fi

dir=$(cd "${dir:-.}" && pwd -P)
[[ $dir != / && $dir != "$HOME" && $dir != "$(dirname "$HOME")" ]] \
	|| die "refusing to expose $dir; point at a project directory"
case $dir in
"$HOME"/.ssh* | "$HOME"/.aws* | "$HOME"/.gnupg* | "$HOME"/.claude* | "$HOME"/.config*)
	die "refusing to expose $dir" ;;
esac

mkdir -p "$HOME_DIR"
args=(
	run --rm --init -i
	--name "yolo-$(basename "$dir" | tr -c 'a-zA-Z0-9_.-' '-')-$$"
	--userns=keep-id:uid=1000,gid=1000
	--cap-drop=ALL --security-opt=no-new-privileges
	--read-only --tmpfs /tmp:rw,exec,size=2g --tmpfs /var/tmp:rw,size=512m
	--memory "$mem" --cpus "$cpus" --pids-limit "$pids"
	-v "$HOME_DIR:/home/node:rw"
	-v "$dir:$dir:rw" -w "$dir"
	-e TERM -e COLORTERM
)

# Network: pasta gives outbound internet; --no-map-gw stops the container reaching
# the host through the gateway address. Host loopback is not mapped by default.
if ((offline)); then
	args+=(--network=none)
else
	args+=(--network=pasta:--no-map-gw)
fi
for p in ${ports[@]+"${ports[@]}"}; do
	[[ $p == *:* ]] || p="$p:$p"
	args+=(-p "127.0.0.1:$p")
done

# Git: a worktree's .git file points at a shared git dir elsewhere; mount it at the
# same path so git works, but keep hooks and config read-only.
gitdir="$dir/.git"
if [[ -f $dir/.git ]]; then
	common=$(git -C "$dir" rev-parse --path-format=absolute --git-common-dir 2>/dev/null || true)
	if [[ -n $common && -d $common ]]; then
		args+=(-v "$common:$common:rw")
		gitdir=$common
	fi
fi
[[ ! -e $gitdir/hooks ]] || args+=(-v "$gitdir/hooks:$gitdir/hooks:ro")
[[ ! -e $gitdir/config ]] || args+=(-v "$gitdir/config:$gitdir/config:ro")

# Identity (not credentials) for commits; API key only if the caller set one.
n=$(git config user.name 2>/dev/null || true)
e=$(git config user.email 2>/dev/null || true)
[[ -z $n ]] || args+=(-e "GIT_AUTHOR_NAME=$n" -e "GIT_COMMITTER_NAME=$n")
[[ -z $e ]] || args+=(-e "GIT_AUTHOR_EMAIL=$e" -e "GIT_COMMITTER_EMAIL=$e")
[[ -z ${ANTHROPIC_API_KEY:-} ]] || args+=(-e ANTHROPIC_API_KEY)

[[ ! -t 0 || ! -t 1 ]] || args+=(-t)
args+=("$IMAGE")
((${#cmd[@]})) && args+=("${cmd[@]}")

echo "yolo-run: $dir (net: $([[ $offline == 1 ]] && echo none || echo internet-only), mem $mem, cpus $cpus)" >&2
exec podman "${args[@]}"
