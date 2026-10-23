#!/bin/sh
# Pulls, verifies and exports the gadgets listed in upstream.txt into an OCI
# layout at $1. Runs inside the Dockerfile's gadgets stage, with ig on PATH
# and the IG version from go.mod in $IG_VERSION.
set -eu
out=$1
refs=""
for line in $(grep -v '^#' "$(dirname "$0")/upstream.txt"); do
	tag=${line%@*}
	case "$tag" in
	*:"$IG_VERSION") ;;
	*) echo "gadget $tag does not match IG version $IG_VERSION from go.mod" >&2; exit 1 ;;
	esac
	pulled=$(ig image pull "$tag" 2>&1 | sed -n 's/^Successfully pulled .*@//p')
	if [ "$pulled" != "${line#*@}" ]; then
		echo "gadget $tag has digest $pulled, pinned ${line#*@}" >&2
		exit 1
	fi
	ig image verify "$tag"
	refs="$refs $tag"
done
ig image export $refs /tmp/gadgets.tar
mkdir -p "$out"
tar xf /tmp/gadgets.tar -C "$out"
