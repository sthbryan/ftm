#!/usr/bin/env bash
# Wrap the Wails Linux binary into an AppImage.
#
# One file that runs on any distro with glibc >= the build host's, with the
# icon and .desktop entry inside it, so launchers stop showing a generic icon.
# GTK and WebKitGTK are bundled by linuxdeploy, which is why this must run on
# the oldest distro we intend to support (the workflow pins ubuntu-22.04).
#
# Output: <out-dir>/ftm-desktop-linux-x86_64.AppImage
#
# Usage: package-appimage.sh <binary> [out-dir]
set -euo pipefail

BIN="${1:?usage: package-appimage.sh <binary> [out-dir]}"
OUT_DIR="${2:-$(dirname "$BIN")}"
VERSION="${VERSION:-$(sed -nE 's/^var Version = "([^"]+)"$/\1/p' internal/version/version.go)}"

export ARCH="${ARCH:-x86_64}"
# GitHub runners have no FUSE, so the AppImage tools must self-extract instead.
export APPIMAGE_EXTRACT_AND_RUN=1

BIN="$(cd "$(dirname "$BIN")" && pwd)/$(basename "$BIN")"
mkdir -p "$OUT_DIR"
OUT_DIR="$(cd "$OUT_DIR" && pwd)"

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

APPDIR="$WORK/AppDir"
mkdir -p "$APPDIR/usr/bin" "$APPDIR/usr/share/applications"

cp "$BIN" "$APPDIR/usr/bin/ftm-desktop"
chmod +x "$APPDIR/usr/bin/ftm-desktop"

cat > "$APPDIR/usr/share/applications/ftm-desktop.desktop" <<DESKTOP
[Desktop Entry]
Type=Application
Name=Foundry Tunnel Manager
GenericName=Tunnel Manager
Comment=Share your Foundry VTT world with your players
Exec=ftm-desktop
Icon=ftm-desktop
Categories=Network;Game;
Keywords=foundry;vtt;tunnel;
Terminal=false
StartupWMClass=ftm-desktop
DESKTOP

for size in 64 128 256 512; do
  dir="$APPDIR/usr/share/icons/hicolor/${size}x${size}/apps"
  mkdir -p "$dir"
  if command -v convert >/dev/null 2>&1; then
    convert desktop/icon.png -resize "${size}x${size}" "$dir/ftm-desktop.png"
  else
    cp desktop/icon.png "$dir/ftm-desktop.png"
  fi
done

TOOLS="$WORK/tools"
mkdir -p "$TOOLS"
base="https://github.com/linuxdeploy/linuxdeploy/releases/download/continuous"
gtk="https://raw.githubusercontent.com/linuxdeploy/linuxdeploy-plugin-gtk/master/linuxdeploy-plugin-gtk.sh"

curl -fsSL -o "$TOOLS/linuxdeploy" "$base/linuxdeploy-${ARCH}.AppImage"
curl -fsSL -o "$TOOLS/linuxdeploy-plugin-gtk.sh" "$gtk"
chmod +x "$TOOLS/linuxdeploy" "$TOOLS/linuxdeploy-plugin-gtk.sh"
export PATH="$TOOLS:$PATH"

WEBKIT_LIBEXEC="$APPDIR/usr/lib/webkit2gtk-4.1"
mkdir -p "$WEBKIT_LIBEXEC"

found_helpers=0
for candidate in /usr/lib/*/webkit2gtk-4.1 /usr/libexec/webkit2gtk-4.1 /usr/lib/webkit2gtk-4.1; do
  if [ -f "$candidate/WebKitNetworkProcess" ]; then
    cp -a "$candidate/." "$WEBKIT_LIBEXEC/"
    echo "Bundled WebKit helpers from $candidate"
    found_helpers=1
    break
  fi
done

if [ "$found_helpers" -eq 0 ]; then
  echo "Could not find WebKitNetworkProcess on this host; install libwebkit2gtk-4.1-0" >&2
  exit 1
fi

"$TOOLS/linuxdeploy" \
  --appdir "$APPDIR" \
  --executable "$APPDIR/usr/bin/ftm-desktop" \
  --desktop-file "$APPDIR/usr/share/applications/ftm-desktop.desktop" \
  --icon-file "$APPDIR/usr/share/icons/hicolor/256x256/apps/ftm-desktop.png" \
  --deploy-deps-only "$WEBKIT_LIBEXEC" \
  --plugin gtk

WK_LINK="/tmp/ftm-webkit-4.1"
for wk_lib in "$APPDIR"/usr/lib/libwebkit2gtk-4.1.so.0*; do
  [ -e "$wk_lib" ] || continue
  python3 "$(dirname "$0")/relocate-webkit.py" "$wk_lib" "$WK_LINK"
done

mkdir -p "$APPDIR/apprun-hooks"
cat > "$APPDIR/apprun-hooks/webkit.sh" <<HOOK
export WEBKIT_LINK="$WK_LINK"
export WEBKIT_TARGET="\$APPDIR/usr/lib/webkit2gtk-4.1"
export WEBKIT_DISABLE_COMPOSITING_MODE=1
if [ -L "\$WEBKIT_LINK" ] || [ -e "\$WEBKIT_LINK" ]; then
  link_uid="\$(stat -c %u "\$WEBKIT_LINK" 2>/dev/null || echo unknown)"
  if [ ! -L "\$WEBKIT_LINK" ] || [ "\$link_uid" != "\$(id -u)" ]; then
    echo "ftm: \$WEBKIT_LINK exists and is not owned by you; cannot set up WebKit helpers." >&2
    exit 1
  fi
fi
ln -sfn "\$WEBKIT_TARGET" "\$WEBKIT_LINK"
export WEBKIT_INJECTED_BUNDLE_PATH="\$WEBKIT_TARGET/injected-bundle"
HOOK

OUTPUT_NAME="ftm-desktop-linux-${ARCH}.AppImage"
(
  cd "$WORK"
  VERSION="$VERSION" "$TOOLS/linuxdeploy" --appdir "$APPDIR" --output appimage
)

built="$(find "$WORK" -maxdepth 1 -name '*.AppImage' -print -quit)"
[ -n "$built" ] || { echo "linuxdeploy produced no AppImage" >&2; exit 1; }

mv "$built" "$OUT_DIR/$OUTPUT_NAME"
chmod +x "$OUT_DIR/$OUTPUT_NAME"
rm -f "$BIN"

echo "Created $OUT_DIR/$OUTPUT_NAME"
