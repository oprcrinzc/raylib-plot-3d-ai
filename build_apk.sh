#!/bin/bash
set -e

echo "=== 1. Compiling Raylib-Go for Android ARM64 ==="
CGO_ENABLED=1 GOOS=android GOARCH=arm64 CC=/data/data/com.termux/files/usr/bin/clang \
go build -v -buildmode=c-shared \
-ldflags="-s -w -extldflags=-Wl,-soname,libmain.so,-wrap,fopen,-z,max-page-size=16384" \
-o libmain.so .

echo "=== 2. Cleaning ELF RUNPATH ==="
patchelf --remove-rpath libmain.so

echo "=== 3. Packaging Resources with AAPT ==="
rm -f iris3d.unaligned.apk iris3d.aligned.apk iris3d.apk
aapt package -f -M AndroidManifest.xml -S res -I /root/android-sdk/android-34/android.jar -F iris3d.unaligned.apk

echo "=== 4. Embedding 16KB-ready ARM64 Shared Library ==="
mkdir -p lib/arm64-v8a
cp -f libmain.so lib/arm64-v8a/libmain.so
zip -0 -u iris3d.unaligned.apk lib/arm64-v8a/libmain.so

echo "=== 5. Zipaligning with 16KB page alignment (Android 15/16 Baklava) ==="
zipalign -f 16384 iris3d.unaligned.apk iris3d.aligned.apk

echo "=== 6. Signing APK with apksigner (v1 + v2 + v3 scheme) ==="
if [ ! -f /root/debug.keystore ]; then
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

echo "=== 7. Verification ==="
apksigner verify -v iris3d.apk
zipalign -c -v 16384 iris3d.apk | grep libmain.so

# Copy to Termux home and phone Downloads for immediate installation
cp -f iris3d.apk /data/data/com.termux/files/home/iris3d.apk 2>/dev/null || true
cp -f iris3d.apk /sdcard/Download/iris3d.apk 2>/dev/null || true

echo "=== BUILD COMPLETE: $(pwd)/iris3d.apk ($(ls -lh iris3d.apk | awk '{print $5}')) ==="
echo "=== Copied to: /sdcard/Download/iris3d.apk and ~/iris3d.apk ==="
