#!/bin/sh
set -eu

arch=${1:?architecture required}
destination=${2:?destination directory required}
case "$arch" in
    amd64) digest=dfe4c9602d6cc21da0f8c5fc289b75ecfcc7e4862d53a7c7aced37e2a115c42e ;;
    arm64) digest=5b51916deb83f361ea6544f8cbb9f72f2db552ff0acbadff3797862793fc0472 ;;
    *) exit 0 ;;
esac

mkdir -p "$destination"
binary="$destination/vk-turn-proxy-linux-$arch"
temporary="$binary.tmp"
trap 'rm -f "$temporary"' EXIT HUP INT TERM
curl -fsSL --retry 3 -o "$temporary" "https://github.com/cacggghp/vk-turn-proxy/releases/download/v1.8.3/server-linux-$arch"
actual=$(sha256sum "$temporary" | cut -d ' ' -f 1)
if [ "$actual" != "$digest" ]; then
    echo "vk-turn-proxy SHA-256 mismatch for $arch" >&2
    exit 1
fi
chmod 755 "$temporary"
mv "$temporary" "$binary"
