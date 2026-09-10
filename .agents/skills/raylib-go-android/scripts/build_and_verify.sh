#!/bin/bash
# ==============================================================================
# Raylib-Go Android Build & Verification Script
# Enforces 16KB page alignment for Android 15 & 16 (Baklava) / ARM64
# ==============================================================================
set -euo pipefail

WORKSPACE_ROOT="/root/3dandroid"
cd "${WORKSPACE_ROOT}"

echo "============================================================"
echo " [1/7] Running Unit Tests..."
echo "============================================================"
go test -v ./...

echo "============================================================"
echo " [2/7] Compiling Raylib-Go ARM64 Shared Library (16KB Page)..."
echo "============================================================"
CGO_ENABLED=1 GOOS=android GOARCH=arm64 \
CC=/data/data/com.termux/files/usr/bin/clang \
go build -v -buildmode=c-shared \
-ldflags="-s -w -extldflags=-Wl,-soname,libmain.so,-wrap,fopen,-z,max-page-size=16384" \
-o libmain.so .

echo "============================================================"
echo " [3/7] Sanitizing ELF Headers (Stripping RUNPATH)..."
echo "============================================================"
patchelf --remove-rpath libmain.so

echo "============================================================"
echo " [4/7] Packaging Android Resources with AAPT..."
echo "============================================================"
rm -f iris3d.unaligned.apk iris3d.aligned.apk iris3d.apk
aapt package -f -M AndroidManifest.xml -S res -I /root/android-sdk/android-34/android.jar -F iris3d.unaligned.apk

echo "============================================================"
echo " [5/7] Embedding Uncompressed 16KB-Ready Native Library..."
echo "============================================================"
mkdir -p lib/arm64-v8a
cp -f libmain.so lib/arm64-v8a/libmain.so
zip -0 -u iris3d.unaligned.apk lib/arm64-v8a/libmain.so

echo "============================================================"
echo " [6/7] Zipaligning (16384-byte boundary for Android 15/16)..."
echo "============================================================"
zipalign -f 16384 iris3d.unaligned.apk iris3d.aligned.apk

echo "============================================================"
echo " [7/7] Signing with APK Signature Schemes v1, v2, v3..."
echo "============================================================"
if [ ! -f /root/debug.keystore ]; then
    echo "Generating new debug keystore..."
    keytool -genkeypair -keystore /root/debug.keystore -storepass android \
        -alias androiddebugkey -keypass android -keyalg RSA -keysize 2048 \
        -validity 10000 -dname "CN=Android Debug,O=Android,C=US"
fi

apksigner sign --ks /root/debug.keystore --ks-pass pass:android \
    --ks-key-alias androiddebugkey --key-pass pass:android \
    --v1-signing-enabled true \
    --v2-signing-enabled true \
    --v3-signing-enabled true \
    --out iris3d.apk iris3d.aligned.apk

echo ""
"${WORKSPACE_ROOT}/.agents/skills/raylib-go-android/scripts/verify_apk.sh" iris3d.apk

# Deploy to accessible locations
cp -f iris3d.apk /sdcard/Download/iris3d.apk 2>/dev/null || true
cp -f iris3d.apk /data/data/com.termux/files/home/iris3d.apk 2>/dev/null || true

echo ""
echo ">>> SUCCESS: iris3d.apk is built, 16KB-aligned, and signed! <<<"
echo ">>> Output: $(ls -lh iris3d.apk | awk '{print $5, $9}') <<<"
