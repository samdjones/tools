#!/usr/bin/env bash
#
# pin-digest.sh — resolve Docker image tag aliases to immutable sha256 digests
# for deterministic, auditable builds. Uses skopeo only.
#
#   pin-digest.sh REF                  resolve only: print "repo:tag@sha256:…"
#   pin-digest.sh --digest-only REF    print just "sha256:…"
#   pin-digest.sh -f Dockerfile        pin/update every resolvable FROM line in place
#   pin-digest.sh -f Dockerfile -n     dry run: show changes, write nothing
#
set -euo pipefail

usage() {
	cat >&2 <<'EOF'
pin-digest.sh — pin Docker base images to immutable sha256 digests (skopeo only)

Usage:
  pin-digest.sh REF                  Resolve REF and print "repo:tag@sha256:…" to stdout
  pin-digest.sh --digest-only REF    Print just the "sha256:…" digest
  pin-digest.sh -f FILE [-n]         Rewrite every resolvable FROM line in a Dockerfile
                                     in place. -n / --dry-run shows changes without writing.
  pin-digest.sh -h | --help          Show this help

Notes:
  * The digest resolved is the multi-arch index digest, so cross-platform builds
    still select the correct image.
  * FROM lines are written as "repo:tag@sha256:…" — the tag stays readable while the
    digest enforces determinism. Re-running re-resolves the tag (idempotent / bump).
  * Audit history is the Dockerfile's git diff.
EOF
}

die() {
	echo "pin-digest.sh: $*" >&2
	exit 1
}

# Resolve a "repo[:tag][@digest]" reference (optionally docker://-prefixed) to its
# index digest via skopeo. Echoes "repo<TAB>tag<TAB>sha256:…".
resolve() {
	local ref="$1" name repo tag digest last
	ref="${ref#docker://}"
	name="${ref%%@*}" # drop any existing @sha256:…
	last="${name##*/}"
	if [[ "$last" == *:* ]]; then
		repo="${name%:*}"
		tag="${name##*:}"
	else
		repo="$name"
		tag="latest"
	fi
	digest="$(skopeo inspect "docker://${repo}:${tag}" --format '{{.Digest}}')" ||
		die "failed to resolve docker://${repo}:${tag} (bad ref, auth, or network?)"
	[[ "$digest" == sha256:* ]] || die "unexpected digest for ${repo}:${tag}: ${digest}"
	printf '%s\t%s\t%s\n' "$repo" "$tag" "$digest"
}

# ---- argument parsing -------------------------------------------------------
mode="resolve"
digest_only=0
dry_run=0
file=""
ref=""

while [[ $# -gt 0 ]]; do
	case "$1" in
	-h | --help)
		usage
		exit 0
		;;
	--digest-only)
		digest_only=1
		shift
		;;
	-f | --file)
		mode="file"
		file="${2:-}"
		[[ -n "$file" ]] || die "$1 requires a path argument"
		shift 2
		;;
	-n | --dry-run)
		dry_run=1
		shift
		;;
	-*)
		die "unknown option: $1 (try --help)"
		;;
	*)
		[[ -z "$ref" ]] || die "unexpected extra argument: $1"
		ref="$1"
		shift
		;;
	esac
done

command -v skopeo >/dev/null 2>&1 || die "skopeo not found on PATH"

# ---- resolve mode -----------------------------------------------------------
if [[ "$mode" == "resolve" ]]; then
	[[ -n "$ref" ]] || die "no image reference given (try --help)"
	IFS=$'\t' read -r repo tag digest < <(resolve "$ref")
	if [[ "$digest_only" -eq 1 ]]; then
		printf '%s\n' "$digest"
	else
		printf '%s:%s@%s\n' "$repo" "$tag" "$digest"
	fi
	exit 0
fi

# ---- file mode --------------------------------------------------------------
[[ -z "$ref" ]] || die "cannot combine a REF argument with -f"
[[ -f "$file" ]] || die "no such file: $file"

# Pass 1: collect build-stage names (FROM … AS <name>), lowercased.
declare -A stages=()
while IFS= read -r line || [[ -n "$line" ]]; do
	read -ra toks <<<"$line"
	[[ "${toks[0]:-}" == "FROM" || "${toks[0]:-}" == "from" ]] || continue
	for ((i = 1; i < ${#toks[@]} - 1; i++)); do
		if [[ "${toks[i]}" == "AS" || "${toks[i]}" == "as" ]]; then
			stages["${toks[i + 1],,}"]=1
		fi
	done
done <"$file"

# Pass 2: resolve and build the rewritten file in memory. Abort before writing
# if any resolution fails (handled by `die` inside resolve()).
output=""
declare -a summary=()
changed=0

while IFS= read -r line || [[ -n "$line" ]]; do
	read -ra toks <<<"$line"
	if [[ "${toks[0]:-}" != "FROM" && "${toks[0]:-}" != "from" ]]; then
		output+="$line"$'\n'
		continue
	fi

	# Locate the image token: first token after FROM that is not a --flag.
	idx=1
	while [[ "$idx" -lt "${#toks[@]}" && "${toks[idx]}" == --* ]]; do
		((idx++))
	done
	if [[ "$idx" -ge "${#toks[@]}" ]]; then
		output+="$line"$'\n'
		continue
	fi
	image="${toks[idx]}"
	imgname="${image%%@*}"
	imglast="${imgname##*/}"

	# Decide whether to skip.
	skip=""
	if [[ "$image" == "scratch" ]]; then
		skip="scratch"
	elif [[ -n "${stages[${image,,}]:-}" ]]; then
		skip="build-stage reference"
	elif [[ "$image" == *@* && "$imglast" != *:* ]]; then
		skip="digest-pinned without a tag (no tag to re-resolve)"
	fi
	if [[ -n "$skip" ]]; then
		echo "skip: ${image} (${skip})" >&2
		output+="$line"$'\n'
		continue
	fi

	IFS=$'\t' read -r repo tag digest < <(resolve "$image")
	newimage="${repo}:${tag}@${digest}"
	if [[ "$newimage" != "$image" ]]; then
		((changed++)) || true
		summary+=("${repo}:${tag}  ${image} -> ${newimage}")
	fi
	toks[idx]="$newimage"
	output+="${toks[*]}"$'\n'
done <"$file"

if [[ "$changed" -eq 0 ]]; then
	echo "no changes: all FROM lines already pinned to current digests" >&2
	exit 0
fi

if [[ "$dry_run" -eq 1 ]]; then
	echo "--- dry run: would update $file ---" >&2
	printf '%s\n' "${summary[@]}"
	exit 0
fi

# Atomic write preserving the original file mode.
tmp="$(mktemp "${file}.XXXXXX")"
trap 'rm -f "$tmp"' EXIT
printf '%s' "$output" >"$tmp"
chmod --reference="$file" "$tmp" 2>/dev/null || true
mv "$tmp" "$file"
trap - EXIT

echo "updated $file:" >&2
printf '%s\n' "${summary[@]}"
