#!/usr/bin/env bash
#
# pin-digest.sh — resolve a Docker image tag alias to its immutable sha256 digest
# for deterministic, auditable builds. Uses skopeo only.
#
#   pin-digest.sh REF                  print "repo:tag@sha256:…" (ready to paste into FROM)
#   pin-digest.sh --digest-only REF    print just "sha256:…"
#   pin-digest.sh --meta REF           also print image metadata (created, platform, labels)
#
set -euo pipefail

usage() {
	cat >&2 <<'EOF'
pin-digest.sh — resolve a Docker image tag to an immutable sha256 digest (skopeo only)

Usage:
  pin-digest.sh REF                  Print "repo:tag@sha256:…" to stdout (paste into FROM)
  pin-digest.sh --digest-only REF    Print just the "sha256:…" digest
  pin-digest.sh --meta REF           Also print image metadata (created, platform, labels)
                                     to stderr; stdout stays the clean pasteable reference
  pin-digest.sh -h | --help          Show this help

Notes:
  * The digest resolved is the multi-arch index digest, so cross-platform builds
    still select the correct image. Pin in a Dockerfile with '@', e.g.
    FROM gcr.io/distroless/python3-debian13:nonroot@sha256:…
  * Metadata (--meta) is read from the image config: the creation time and labels such
    as org.opencontainers.image.{version,revision,source}. Some images (e.g. distroless)
    zero the timestamp for reproducibility, shown as a 1970 epoch with a note.
EOF
}

die() {
	echo "pin-digest.sh: $*" >&2
	exit 1
}

# Resolve a "repo[:tag][@digest]" reference (optionally docker://-prefixed) to its
# index digest via skopeo, in a single inspect call.
# Echoes "repo<TAB>tag<TAB>sha256:…<TAB>created<TAB>os/arch".
resolve() {
	local ref="$1" name repo tag last out
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
	out="$(skopeo inspect "docker://${repo}:${tag}" \
		--format '{{.Digest}}{{"\t"}}{{.Created}}{{"\t"}}{{.Os}}/{{.Architecture}}')" ||
		die "failed to resolve docker://${repo}:${tag} (bad ref, auth, or network?)"
	[[ "$out" == sha256:* ]] || die "unexpected inspect output for ${repo}:${tag}: ${out}"
	printf '%s\t%s\t%s\n' "$repo" "$tag" "$out"
}

# Print a human-readable metadata block for repo:tag to stderr. Labels are fetched
# with a Go-template range so a nil label map simply yields nothing.
print_meta() {
	local repo="$1" tag="$2" created="$3" platform="$4" note="" labels
	[[ "$created" == 1970-01-01\ 00:00:00\ * ]] && note="  (zeroed for reproducible build)"
	{
		echo "  metadata for ${repo}:${tag}"
		echo "    created:  ${created}${note}"
		echo "    platform: ${platform}"
		labels="$(skopeo inspect "docker://${repo}:${tag}" \
			--format '{{range $k, $v := .Labels}}    label:    {{$k}}={{$v}}
{{end}}' 2>/dev/null)"
		[[ -n "${labels//[$' \t\n']/}" ]] && printf '%s\n' "$labels"
	} >&2
}

# ---- argument parsing -------------------------------------------------------
digest_only=0
show_meta=0
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
	--meta)
		show_meta=1
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
[[ -n "$ref" ]] || die "no image reference given (try --help)"

IFS=$'\t' read -r repo tag digest created platform < <(resolve "$ref")
[[ "$show_meta" -eq 1 ]] && print_meta "$repo" "$tag" "$created" "$platform"
if [[ "$digest_only" -eq 1 ]]; then
	printf '%s\n' "$digest"
else
	printf '%s:%s@%s\n' "$repo" "$tag" "$digest"
fi
