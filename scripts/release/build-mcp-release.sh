#!/usr/bin/env bash
set -euo pipefail

usage() {
	cat <<'USAGE'
Usage: build-mcp-release.sh --version VERSION --output DIR [--platform GOOS/GOARCH ...]

Build Timich MCP release bundles and checksums.

Options:
  --version VERSION       Stable release version. Accepts 0.1.0 or v0.1.0.
  --output DIR            Directory for release artifacts.
  --platform GOOS/GOARCH  Target platform. May be repeated.
                          Defaults to linux/amd64, linux/arm64, darwin/amd64,
                          darwin/arm64, and windows/amd64.
USAGE
}

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
version=""
output=""
platforms=()

while [ "$#" -gt 0 ]; do
	case "$1" in
		--version)
			if [ "$#" -lt 2 ]; then
				usage >&2
				exit 2
			fi
			version="${2#v}"
			shift 2
			;;
		--output)
			if [ "$#" -lt 2 ]; then
				usage >&2
				exit 2
			fi
			output="$2"
			shift 2
			;;
		--platform)
			if [ "$#" -lt 2 ]; then
				usage >&2
				exit 2
			fi
			platforms+=("$2")
			shift 2
			;;
		-h|--help)
			usage
			exit 0
			;;
		*)
			echo "unknown argument: $1" >&2
			usage >&2
			exit 2
			;;
	esac
done

if [ -z "$version" ] || [ -z "$output" ]; then
	usage >&2
	exit 2
fi

if ! [[ "$version" =~ ^[0-9]+[.][0-9]+[.][0-9]+$ ]]; then
	echo "stable release version must look like 0.1.0 or v0.1.0: $version" >&2
	exit 2
fi

if [ "${#platforms[@]}" -eq 0 ]; then
	platforms=("linux/amd64" "linux/arm64" "darwin/amd64" "darwin/arm64" "windows/amd64")
fi

cd "$repo_root"

export GOCACHE="${GOCACHE:-$repo_root/build/go-build-cache}"

output_parent="$(dirname "$output")"
output_name="$(basename "$output")"
mkdir -p "$output_parent"
output_parent_abs="$(cd "$output_parent" && pwd)"
output_abs="$output_parent_abs/$output_name"
build_root="$repo_root/build/release"
stage_root="$build_root/stage"

rm -rf "$build_root" "$output_abs"
mkdir -p "$stage_root" "$output_abs"

commit="${TIMICH_COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo unknown)}"
built_at="${TIMICH_BUILT_AT:-$(date -u +"%Y-%m-%dT%H:%M:%SZ")}"

sha256_file() {
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$1" | awk '{print $1}'
	else
		shasum -a 256 "$1" | awk '{print $1}'
	fi
}

for platform in "${platforms[@]}"; do
	if [[ "$platform" != */* ]]; then
		echo "platform must look like GOOS/GOARCH: $platform" >&2
		exit 2
	fi
	dist_os="${platform%/*}"
	dist_arch="${platform#*/}"
	dist_name="timich-mcp_${version}_${dist_os}_${dist_arch}"
	stage="$stage_root/$dist_name"
	archive="$output_abs/$dist_name.tar.gz"
	binary_name="timich-mcp"

	if [ "$dist_os" = "windows" ]; then
		binary_name="timich-mcp.exe"
	fi

	rm -rf "$stage"
	mkdir -p "$stage"

	GOOS="$dist_os" GOARCH="$dist_arch" CGO_ENABLED=0 \
		go build -trimpath -buildvcs=false \
		-ldflags "-X main.version=$version -X main.commit=$commit -X main.builtAt=$built_at" \
		-o "$stage/$binary_name" \
		./cmd/timich-mcp

	cat > "$stage/VERSION" <<VERSION
TIMICH_MCP_VERSION=$version
TIMICH_COMMIT=$commit
TIMICH_BUILT_AT=$built_at
GOOS=$dist_os
GOARCH=$dist_arch
VERSION

	cat > "$stage/BUILDINFO.json" <<BUILDINFO
{
  "mcpVersion": "$version",
  "commit": "$commit",
  "builtAt": "$built_at",
  "goos": "$dist_os",
  "goarch": "$dist_arch"
}
BUILDINFO

	cp README.md "$stage/README.md"

	tar -C "$stage_root" -czf "$archive" "$dist_name"
	archive_sha="$(sha256_file "$archive")"
	printf '%s  %s\n' "$archive_sha" "$dist_name.tar.gz" > "$archive.sha256"
done

printf 'Wrote release artifacts to %s\n' "$output_abs"
