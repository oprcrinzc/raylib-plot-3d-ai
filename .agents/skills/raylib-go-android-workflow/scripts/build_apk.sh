#!/bin/bash
# ==============================================================================
# Raylib-Go Android Workflow: Generalized APK Build Pipeline
# Enforces Android 15 & 16 (Baklava) 16KB ELF Memory Page Alignment
# ==============================================================================
set -euo pipefail

# Default configuration
PROJECT_DIR="$(pwd)"
APK_BASENAME="${APK_NAME:-app}"
APK_BASENAME="${APK_BASENAME%.apk}" # strip .apk extension if supplied
OUT_DIR="${OUT_DIR:-${PROJECT_DIR}}"
CC_COMPILER="${CC:-/data/data/com.termux/files/usr/bin/clang}"
ANDROID_JAR="${ANDROID_JAR:-/root/android-sdk/android-34/android.jar}"
KEYSTORE_PATH="${KEYSTORE_PATH:-/root/debug.keystore}"
KEYSTORE_PASS="${KEYSTORE_PASS:-android}"
KEY_ALIAS="${KEY_ALIAS:-androiddebugkey}"
RUN_TESTS=1
CLEAN_BUILD=0

# Parse arguments
while [[ $# -gt 0 ]]; do
    case "$1" in
        --name|-n)
            APK_BASENAME="${2%.apk}"
            shift 2
            ;;
        --no-test)
            RUN_TESTS=0
            shift
            ;;
        --clean|-c)
            CLEAN_BUILD=1
            shift
            ;;
        --out|-o)
            OUT_DIR="$2"
            shift 2
            ;;
        --help|-h)
            echo "Usage: $0 [OPTIONS]"
            echo "Options:"
            echo "  --name, -n <name>     Base name for the APK (default: 'app')"
            echo "  --no-test             Skip 'go test ./...'"
            echo "  --clean, -c           Clean previous build artifacts before starting"
            echo "  --out, -o <dir>       Output directory for the final APK (default: current dir)"
            echo "  --help, -h            Show this help message"
            exit 0
            ;;
        *)
            echo "Unknown argument: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

cd "${PROJECT_DIR}"

if [ ! -f "AndroidManifest.xml" ]; then
    echo "ERROR: AndroidManifest.xml not found in ${PROJECT_DIR}!"
    echo "Make sure you run this script from your Raylib-Go project root."
    exit 1
fi

if [ ! -d "res" ]; then
    echo "ERROR: 'res/' directory not found in ${PROJECT_DIR}!"
    exit 1
fi

UNALIGNED_APK="${OUT_DIR}/${APK_BASENAME}.unaligned.apk"
ALIGNED_APK="${OUT_DIR}/${APK_BASENAME}.aligned.apk"
FINAL_APK="${OUT_DIR}/${APK_BASENAME}.apk"

if [ "${CLEAN_BUILD}" -eq 1 ]; then
    echo "Cleaning temporary build artifacts..."
    rm -f libmain.so "${UNALIGNED_APK}" "${ALIGNED_APK}" "${FINAL_APK}"
    rm -rf lib/
fi

# Step 1: Pre-build test check
if [ "${RUN_TESTS}" -eq 1 ]; then
    echo "============================================================"
    echo " [1/6] Running Go Tests & Checks..."
    echo "============================================================"
    CGO_ENABLED=1 GOOS=android GOARCH=arm64 CC="${CC_COMPILER}" go vet ./...
    if ls ./*_test.go >/dev/null 2>&1 || [ -d "test" ]; then
        go test -v ./...
    else
        echo " No unit tests found, vet passed."
    fi
else
    echo "============================================================"
    echo " [1/6] Skipping Unit Tests (--no-test requested)"
    echo "============================================================"
fi

# Step 2: Compile CGO ARM64 Shared Library with 16KB Page Size
echo "============================================================"
echo " [2/6] Compiling Raylib-Go ARM64 Shared Library (16KB Page Alignment)..."
echo "============================================================"
CGO_ENABLED=1 GOOS=android GOARCH=arm64 \
CC="${CC_COMPILER}" \
go build -v -buildmode=c-shared \
-ldflags="-s -w -extldflags=-Wl,-soname,libmain.so,-wrap,fopen,-z,max-page-size=16384" \
-o libmain.so .

# Step 3: Sanitize ELF Header (Strip RUNPATH/RPATH)
echo "============================================================"
echo " [3/6] Sanitizing ELF Headers with patchelf..."
echo "============================================================"
patchelf --remove-rpath libmain.so

# Step 4: Package Resources with AAPT
echo "============================================================"
echo " [4/6] Packaging Resources with AAPT..."
echo "============================================================"
rm -f "${UNALIGNED_APK}" "${ALIGNED_APK}"

AAPT_EXTRA_FLAGS=()
if [ -d "assets" ]; then
    echo " Found 'assets/' directory, including in APK packaging..."
    AAPT_EXTRA_FLAGS+=("-A" "assets")
fi

aapt package -f -M AndroidManifest.xml -S res "${AAPT_EXTRA_FLAGS[@]}" -I "${ANDROID_JAR}" -F "${UNALIGNED_APK}"

# Step 5: Embed Native Library Uncompressed (zip -0) & Zipalign 16KB
echo "============================================================"
echo " [5/6] Embedding Library Uncompressed & 16KB Zipaligning..."
echo "============================================================"
mkdir -p lib/arm64-v8a
cp -f libmain.so lib/arm64-v8a/libmain.so
zip -0 -u "${UNALIGNED_APK}" lib/arm64-v8a/libmain.so

zipalign -f 16384 "${UNALIGNED_APK}" "${ALIGNED_APK}"

# Step 6: Multi-Scheme Signing (v1, v2, v3)
echo "============================================================"
echo " [6/6] Signing with apksigner (Schemes v1, v2, v3)..."
echo "============================================================"
if [ ! -f "${KEYSTORE_PATH}" ]; then
    echo " Keystore not found. Generating default debug keystore at ${KEYSTORE_PATH}..."
    keytool -genkeypair -keystore "${KEYSTORE_PATH}" -storepass "${KEYSTORE_PASS}" \
        -alias "${KEY_ALIAS}" -keypass "${KEYSTORE_PASS}" -keyalg RSA -keysize 2048 \
        -validity 10000 -dname "CN=Android Debug,O=Android,C=US"
fi

apksigner sign --ks "${KEYSTORE_PATH}" --ks-pass "pass:${KEYSTORE_PASS}" \
    --ks-key-alias "${KEY_ALIAS}" --key-pass "pass:${KEYSTORE_PASS}" \
    --v1-signing-enabled true \
    --v2-signing-enabled true \
    --v3-signing-enabled true \
    --out "${FINAL_APK}" "${ALIGNED_APK}"

# Clean up intermediate unaligned/aligned/idsig files
rm -f "${UNALIGNED_APK}" "${ALIGNED_APK}" "${FINAL_APK}.idsig"

echo ""
echo "============================================================"
echo " Build Success!"
echo " Output: ${FINAL_APK} ($(ls -lh "${FINAL_APK}" | awk '{print $5}'))"
echo "============================================================"
