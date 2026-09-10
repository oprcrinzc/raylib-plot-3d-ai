#!/bin/bash
# ==============================================================================
# Raylib-Go Android Workflow: APK Inspector & 16KB Verification Runbook
# Validates 16KB boundary alignment (Android 15/16 Baklava) & APK signatures
# ==============================================================================
set -euo pipefail

APK_TARGET="${1:-}"

if [ -z "${APK_TARGET}" ]; then
    # Try finding any .apk in current directory
    APK_TARGET=$(ls -1 ./*.apk 2>/dev/null | grep -v -E "unaligned|aligned" | head -n 1 || true)
    if [ -z "${APK_TARGET}" ]; then
        echo "Usage: $0 <path_to_apk>"
        exit 1
    fi
fi

if [ ! -f "${APK_TARGET}" ]; then
    echo "ERROR: APK file '${APK_TARGET}' does not exist!"
    exit 1
fi

echo "============================================================"
echo " Raylib-Go Android Workflow: Verifying APK"
echo " Target APK : ${APK_TARGET}"
echo " File Size  : $(ls -lh "${APK_TARGET}" | awk '{print $5}')"
echo "============================================================"

# 1. Package Metadata & NativeActivity Check
echo "--- 1. Package & Activity Metadata (AAPT) ---"
if command -v aapt >/dev/null 2>&1; then
    BADGING=$(aapt dump badging "${APK_TARGET}" 2>/dev/null || true)
    echo "${BADGING}" | grep -E "package: name|sdkVersion|targetSdkVersion|application-label" || true
    if echo "${BADGING}" | grep -q "android.app.NativeActivity"; then
        echo " [PASS] android.app.NativeActivity configured as entry activity"
    else
        echo " [WARN] NativeActivity entry not explicitly matched in AAPT dump"
    fi
else
    echo " [SKIP] aapt command not available"
fi

# 2. Native Shared Libraries List
echo "--- 2. Native Shared Libraries in APK ---"
SO_FILES=$(unzip -l "${APK_TARGET}" | grep "lib/arm64-v8a/.*\.so" || true)
if [ -z "${SO_FILES}" ]; then
    echo " [FAIL] No ARM64 shared libraries found in lib/arm64-v8a/!"
    exit 2
else
    echo "${SO_FILES}"
fi

# 3. 16KB Memory Page Alignment Verification
echo "--- 3. 16KB Page Alignment Verification (Android 15/16 Baklava) ---"
ALIGN_OUTPUT=$(zipalign -c -v 4 "${APK_TARGET}" 2>/dev/null || true)

SO_ALIGNED_COUNT=0
SO_FAILED_COUNT=0

while read -r line; do
    if [ -n "${line}" ]; then
        offset=$(echo "${line}" | awk '{print $1}')
        so_name=$(echo "${line}" | awk '{print $2}')
        if [ $((offset % 16384)) -eq 0 ]; then
            echo " [PASS] ${so_name}: Offset ${offset} is aligned to 16KB boundary ($((offset / 16384)) * 16384)"
            SO_ALIGNED_COUNT=$((SO_ALIGNED_COUNT + 1))
        else
            echo " [FAIL] ${so_name}: Offset ${offset} is NOT aligned to 16KB! Remainder: $((offset % 16384))"
            SO_FAILED_COUNT=$((SO_FAILED_COUNT + 1))
        fi
    fi
done < <(echo "${ALIGN_OUTPUT}" | grep "lib/.*\.so" || true)

if [ "${SO_FAILED_COUNT}" -gt 0 ]; then
    echo "Result: 16KB alignment failed. Modern Android (15/16) devices will reject this APK."
    exit 3
elif [ "${SO_ALIGNED_COUNT}" -eq 0 ]; then
    echo " [FAIL] Could not verify alignment for any shared library in APK."
    exit 3
else
    echo " [PASS] All native shared libraries meet Android 15/16 16KB page alignment requirements."
fi

# 4. APK Signature Scheme Verification
echo "--- 4. APK Multi-Scheme Signature Verification ---"
if command -v apksigner >/dev/null 2>&1; then
    apksigner verify -v "${APK_TARGET}"
    echo " [PASS] APK signatures verified successfully."
else
    echo " [SKIP] apksigner command not available"
fi

echo "============================================================"
echo " VERIFICATION RESULT: APK IS READY FOR ANDROID DEPLOYMENT"
echo "============================================================"
