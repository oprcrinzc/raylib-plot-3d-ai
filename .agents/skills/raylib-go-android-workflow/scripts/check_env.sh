#!/bin/bash
# ==============================================================================
# Raylib-Go Android Workflow: Toolchain & Environment Sanity Checker
# Validates host compilers, Android SDK utilities, and 16KB alignment readiness.
# ==============================================================================
set -euo pipefail

echo "============================================================"
echo " Raylib-Go Android Workflow: Toolchain & Environment Check"
echo "============================================================"

PASS_COUNT=0
WARN_COUNT=0
FAIL_COUNT=0

check_cmd() {
    local name="$1"
    local cmd="$2"
    if command -v "${cmd}" >/dev/null 2>&1; then
        local path
        path=$(command -v "${cmd}")
        echo " [PASS] ${name}: found at ${path}"
        PASS_COUNT=$((PASS_COUNT + 1))
        return 0
    else
        echo " [FAIL] ${name}: '${cmd}' not found in PATH!"
        FAIL_COUNT=$((FAIL_COUNT + 1))
        return 1
    fi
}

check_file() {
    local name="$1"
    local path="$2"
    if [ -f "${path}" ]; then
        echo " [PASS] ${name}: found at ${path}"
        PASS_COUNT=$((PASS_COUNT + 1))
        return 0
    else
        echo " [FAIL] ${name}: file not found at ${path}"
        FAIL_COUNT=$((FAIL_COUNT + 1))
        return 1
    fi
}

# 1. Host Go Toolchain
echo "--- 1. Go Toolchain ---"
if command -v go >/dev/null 2>&1; then
    go_ver=$(go version)
    echo " [PASS] Go: ${go_ver}"
    PASS_COUNT=$((PASS_COUNT + 1))
else
    echo " [FAIL] Go compiler not found in PATH"
    FAIL_COUNT=$((FAIL_COUNT + 1))
fi

# 2. C Compiler for CGO (Android ARM64)
echo "--- 2. CGO C Compiler (ARM64 Clang) ---"
CLANG_PATH="${CC:-/data/data/com.termux/files/usr/bin/clang}"
if [ -x "${CLANG_PATH}" ]; then
    clang_ver=$("${CLANG_PATH}" --version 2>&1 | head -n 1)
    echo " [PASS] Clang: ${clang_ver} (${CLANG_PATH})"
    PASS_COUNT=$((PASS_COUNT + 1))
elif command -v clang >/dev/null 2>&1; then
    CLANG_PATH=$(command -v clang)
    clang_ver=$("${CLANG_PATH}" --version 2>&1 | head -n 1)
    echo " [WARN] Clang: default system clang found (${CLANG_PATH}), verify target support"
    WARN_COUNT=$((WARN_COUNT + 1))
else
    echo " [FAIL] ARM64 Clang not found at ${CLANG_PATH}"
    FAIL_COUNT=$((FAIL_COUNT + 1))
fi

# 3. Android SDK Platform Jar
echo "--- 3. Android Platform SDK ---"
ANDROID_JAR="${ANDROID_JAR:-/root/android-sdk/android-34/android.jar}"
if [ ! -f "${ANDROID_JAR}" ]; then
    # Search for alternative android.jar
    ALT_JAR=$(find /root/android-sdk -name "android.jar" 2>/dev/null | sort -V | tail -n 1 || true)
    if [ -n "${ALT_JAR}" ] && [ -f "${ALT_JAR}" ]; then
        ANDROID_JAR="${ALT_JAR}"
    fi
fi
check_file "android.jar" "${ANDROID_JAR}"

# 4. Android Build Tools & Binary Utilities
echo "--- 4. Android Packaging & Binary Tools ---"
check_cmd "AAPT" "aapt"
check_cmd "Zipalign" "zipalign"
check_cmd "Apksigner" "apksigner"
check_cmd "Patchelf" "patchelf"
check_cmd "Keytool" "keytool"
check_cmd "Zip" "zip"
check_cmd "Unzip" "unzip"

# 5. Keystore
echo "--- 5. Signing Keystore ---"
KEYSTORE_PATH="${KEYSTORE_PATH:-/root/debug.keystore}"
if [ -f "${KEYSTORE_PATH}" ]; then
    echo " [PASS] Keystore found at ${KEYSTORE_PATH}"
    PASS_COUNT=$((PASS_COUNT + 1))
else
    echo " [WARN] Keystore not found at ${KEYSTORE_PATH} (will be auto-generated on first build)"
    WARN_COUNT=$((WARN_COUNT + 1))
fi

echo "============================================================"
echo " Summary: ${PASS_COUNT} Passed, ${WARN_COUNT} Warnings, ${FAIL_COUNT} Failures"
echo "============================================================"

if [ "${FAIL_COUNT}" -gt 0 ]; then
    echo "Result: Environment is INCOMPLETE for Android builds."
    exit 1
else
    echo "Result: Environment is READY for Raylib-Go Android builds!"
    exit 0
fi
