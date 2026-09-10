#!/bin/bash
# ==============================================================================
# Raylib-Go Android Workflow: Project Scaffolding Tool
# Generates minimal Android build scaffolding for a Raylib-Go application.
# ==============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SKILL_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

TARGET_DIR="${1:-.}"
PACKAGE_NAME="${2:-com.example.raylibapp}"
APP_TITLE="${3:-Raylib App}"
ORIENTATION="${4:-sensorLandscape}"

mkdir -p "${TARGET_DIR}"
cd "${TARGET_DIR}"
TARGET_ABS="$(pwd)"

echo "============================================================"
echo " Scaffolding Raylib-Go Android Project"
echo " Target Directory : ${TARGET_ABS}"
echo " Package Name     : ${PACKAGE_NAME}"
echo " App Title        : ${APP_TITLE}"
echo " Screen Mode      : ${ORIENTATION}"
echo "============================================================"

# 1. Create directory structure
mkdir -p res/values res/drawable assets

# 2. Generate strings.xml
if [ ! -f "res/values/strings.xml" ]; then
    echo " Generating res/values/strings.xml..."
    sed "s|\${APP_TITLE}|${APP_TITLE}|g" "${SKILL_ROOT}/templates/strings.xml.template" > res/values/strings.xml
else
    echo " res/values/strings.xml already exists, skipping."
fi

# 3. Generate AndroidManifest.xml
if [ ! -f "AndroidManifest.xml" ]; then
    echo " Generating AndroidManifest.xml..."
    sed -e "s|\${PACKAGE_NAME}|${PACKAGE_NAME}|g" \
        -e "s|\${ORIENTATION}|${ORIENTATION}|g" \
        "${SKILL_ROOT}/templates/AndroidManifest.xml.template" > AndroidManifest.xml
else
    echo " AndroidManifest.xml already exists, skipping."
fi

# 4. Generate Launcher Icon if missing
if [ ! -f "res/drawable/icon.png" ]; then
    echo " Providing default launcher icon..."
    if [ -f "/root/3dandroid/res/drawable/icon.png" ]; then
        cp "/root/3dandroid/res/drawable/icon.png" res/drawable/icon.png
    fi
fi

# 5. Generate main.go if missing
if [ ! -f "main.go" ]; then
    echo " Generating template main.go..."
    cp "${SKILL_ROOT}/templates/main.go.template" main.go
else
    echo " main.go already exists, keeping existing code."
fi

# 6. Initialize go.mod if missing
if [ ! -f "go.mod" ]; then
    echo " Initializing go.mod..."
    go mod init "${PACKAGE_NAME}"
    echo " Adding raylib-go dependency..."
    go get github.com/gen2brain/raylib-go/raylib@latest || true
else
    echo " go.mod already exists, skipping."
fi

# 7. Add local build scripts for developer convenience
cp -f "${SCRIPT_DIR}/build_apk.sh" ./build_apk.sh
chmod +x ./build_apk.sh

cp -f "${SCRIPT_DIR}/verify_apk.sh" ./verify_apk.sh
chmod +x ./verify_apk.sh

echo "============================================================"
echo " Project Scaffolding Complete!"
echo " Next steps:"
echo "   1. Edit main.go to implement your graphics logic."
echo "   2. Run './build_apk.sh' to compile and package."
echo "   3. Run './verify_apk.sh <app.apk>' to verify 16KB alignment."
echo "============================================================"
