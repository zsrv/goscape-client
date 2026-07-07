#!/usr/bin/env bash
# build.sh — reproducibly compile the pinned LostCityRS/Client-Java (225-clean)
# plus the IconDump harness, run it against the rev-225 client jag archives, and
# write the item-icon rasterizer goldens into cmd/icondump/testdata/.
#
# The one deviation from a literal client run: setBrightness's Math.random
# brightness jitter is patched out of a throwaway worktree copy of Pix3D.java so
# the palette (and every colour derived from it) is bit-reproducible at
# brightness 0.8. See README.md.
#
# Usage: ./build.sh [--keep]   (--keep leaves the temp worktree for inspection)
set -euo pipefail

PIN=cc3781de
CJ_REPO=${CJ_REPO:-/home/owner/Code/github.com/LostCityRS/Client-Java}
SERVER=${SERVER:-/home/owner/Code/github.com/LostCityRS/Server225_2}
JAGDIR=${JAGDIR:-$SERVER/engine/data/pack/client}
OBJPACK=${OBJPACK:-$SERVER/content/pack/obj.pack}

HERE=$(cd "$(dirname "$0")" && pwd)
OUT=${OUT:-$HERE/../../../cmd/icondump/testdata}
WORK=$(mktemp -d)
CJ="$WORK/cj"
CLASSES="$WORK/classes"

cleanup() {
	git -C "$CJ_REPO" worktree remove --force "$CJ" 2>/dev/null || true
	if [[ "${1:-}" != "keep" ]]; then rm -rf "$WORK"; fi
}
trap 'cleanup "${KEEP:-}"' EXIT
[[ "${1:-}" == "--keep" ]] && KEEP=keep

echo "Client-Java pin: $PIN (branch 225-clean)"
git -C "$CJ_REPO" worktree add --detach "$CJ" "$PIN"

# --- documented patch: remove the setBrightness brightness jitter ---
# At 225-clean the jitter local is var28 (it was var3 at the 254/274 pins).
PIX="$CJ/src/main/java/jagex2/graphics/Pix3D.java"
sed -i \
	's|double var28 = arg1 + (Math.random() \* 0.03D - 0.015D);|double var28 = arg1; // PATCHED (IconDump): Math.random jitter removed for reproducible goldens|' \
	"$PIX"
grep -q 'PATCHED (IconDump)' "$PIX" || { echo "ERROR: jitter patch did not apply (Pix3D.java changed?)"; exit 1; }

# --- compile the whole pinned client + the harness ---
cp "$HERE/IconDump.java" "$CJ/src/main/java/IconDump.java"
mkdir -p "$CLASSES"
find "$CJ/src/main/java" -name '*.java' > "$WORK/srcs.txt"
javac -d "$CLASSES" @"$WORK/srcs.txt"

# --- run ---
mkdir -p "$OUT"
java -Djava.awt.headless=true -cp "$CLASSES" IconDump \
	--jag-dir "$JAGDIR" --out "$OUT" --objpack "$OBJPACK"

echo "goldens written under: $OUT"
