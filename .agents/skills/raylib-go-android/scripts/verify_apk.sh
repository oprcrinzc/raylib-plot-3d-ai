#!/bin/bash
# ==============================================================================
# APK Inspector & 16KB Alignment Verification Script
# Usage: ./verify_apk.sh [path_to_apk]
# ==============================================================================
set -euo pipefail

APK_TARGET="${1:-iris3d.apk}"

if [ ! -f "${APK_TARGET}" ]; then
    echo "Error: APK file '${APK_TARGET}' not found!"
    exit 1
fi

echo "============================================================"
echo " Inspecting APK: ${APK_TARGET}"
echo " File Size: $(ls -lh "${APK_TARGET}" | awk '{print $5}')"
echo "============================================================"

# 1. Check AAPT Badging & Metadata
echo "--- 1. Package Metadata (AAPT) ---"
aapt dump badging "${APK_TARGET}" | grep -E "package: name|sdkVersion|targetSdkVersion|application-label" || true

# 2. Check Native Libraries Embedded
echo "--- 2. Native Shared Libraries ---"
unzip -l "${APK_TARGET}" | grep "lib/" || echo "No lib/ directories found!"

# 3. Verify 16KB Page Alignment
echo "--- 3. 16KB Page Alignment Verification (16384 bytes) ---"
align_line=$(zipalign -c -v 4 "${APK_TARGET}" 2>/dev/null | grep "lib/arm64-v8a/libmain.so" || true)
if [ -z "${align_line}" ]; then
    echo "[FAIL] lib/arm64-v8a/libmain.so not found in APK!"
    exit 2
fi

offset=$(echo "${align_line}" | awk '{print $1}')
echo "Library offset: ${offset} bytes"

if [ $((offset % 16384)) -eq 0 ]; then
    echo "[PASS] libmain.so is aligned to a 16KB boundary (offset ${offset} = $((offset / 16384)) * 16384 bytes)!"
else
    echo "[FAIL] libmain.so offset (${offset}) is NOT aligned to 16KB (remainder: $((offset % 16384)))!"
    exit 2
fi

# 4. Verify Signatures
echo "--- 4. APK Signature Scheme Verification ---"
apksigner verify -v "${APK_TARGET}"
echo "[PASS] Signatures verified successfully!"
