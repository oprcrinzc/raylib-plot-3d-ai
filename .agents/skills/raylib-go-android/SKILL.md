---
name: raylib-go-android
description: >-
  Use this skill when developing, building, testing, or packaging Raylib-Go 3D/4D
  applications for Android (ARM64), including managing CGO cross-compilation,
  enforcing 16KB page alignment (Android 15/16 Baklava), handling touch gestures and
  orbit camera controls, extending datasets and colormaps, or troubleshooting APK deployment.
---

# Raylib-Go Android (3D/4D Visualizer) Skill

This skill provides step-by-step procedures, build pipelines, architecture references, and verification runbooks for developing, compiling, and deploying high-performance Raylib-Go 3D/4D applications on Android (ARM64), with strict adherence to Android 15/16 (Baklava) 16KB memory page alignment.

---

## Quick Reference: Key Workflows

Task | Action / Command | Runbook / Reference
:--- | :--- | :---
**Run Unit Tests** | `go test -v ./...` | [Development & Testing](#1-development--testing-workflow)
**Build & Package APK** | `./.agents/skills/raylib-go-android/scripts/build_and_verify.sh` | [Build Pipeline Guide](./references/build_pipeline.md)
**Verify APK (16KB + Sign)** | `./.agents/skills/raylib-go-android/scripts/verify_apk.sh iris3d.apk` | [Verification Guide](#3-verification-runbook)
**Architecture Reference** | View component interactions, touch dispatch, & 3D ray picking | [Architecture Guide](./references/architecture.md)
**Add a New Dataset** | Implement embedded or dynamic multi-dimensional dataset | [Dataset Example](./examples/add_dataset_example.go)

---

## 1. Development & Testing Workflow

Before building the Android shared library, always validate data structures, normalization algorithms, and colormap calculations on the host/PRoot environment.

### Run Unit Tests
```bash
go test -v ./...
```

### Dataset Validation Requirements
When modifying [`dataset.go`](file:///root/3dandroid/dataset.go) or adding new datasets:
1. Ensure all numeric features normalize cleanly to $[0.0, 1.0]$ via `Get01NormalizedValue(pointIdx, colIdx)`.
2. Guard against division by zero when min and max values are identical (handled by returning mid-range `0.5`).
3. Handle missing values, NaNs, and varied CSV delimiters (`comma`, `tab`, `semicolon`) automatically.

---

## 2. Android Build & Packaging Pipeline

Modern Android versions (Android 15 and 16 / Baklava) enforce **16KB ELF memory page alignment**. Compiling without 16KB alignment causes immediate dynamic linker crashes (`SIGSEGV` or `dlopen failed: unaligned segment`) on modern devices such as the Nothing Phone 3a and MediaTek ARM64 chipsets.

### Automated Build Script
Execute the bundled build script from the workspace root:
```bash
./build_apk.sh
# OR use the enhanced skill script:
./.agents/skills/raylib-go-android/scripts/build_and_verify.sh
```

### Pipeline Steps Overview
1. **CGO Cross-Compilation (`libmain.so`)**:
   ```bash
   CGO_ENABLED=1 GOOS=android GOARCH=arm64 CC=/data/data/com.termux/files/usr/bin/clang \
   go build -v -buildmode=c-shared \
   -ldflags="-s -w -extldflags=-Wl,-soname,libmain.so,-wrap,fopen,-z,max-page-size=16384" \
   -o libmain.so .
   ```
2. **ELF Header Sanitization**:
   Remove `DT_RUNPATH` / `DT_RPATH` to prevent Android linker rejection:
   ```bash
   patchelf --remove-rpath libmain.so
   ```
3. **Resource Packaging**:
   Package manifest, drawables, strings, and icons with AAPT:
   ```bash
   aapt package -f -M AndroidManifest.xml -S res -I /root/android-sdk/android-34/android.jar -F iris3d.unaligned.apk
   ```
4. **Embed Shared Library Uncompressed**:
   Native libraries must be stored uncompressed (`-0`) for the Android runtime to memory-map them directly:
   ```bash
   mkdir -p lib/arm64-v8a
   cp -f libmain.so lib/arm64-v8a/libmain.so
   zip -0 -u iris3d.unaligned.apk lib/arm64-v8a/libmain.so
   ```
5. **16KB Zipaligning**:
   Align all uncompressed data on 16384-byte boundaries:
   ```bash
   zipalign -f 16384 iris3d.unaligned.apk iris3d.aligned.apk
   ```
6. **Multi-Scheme Signing**:
   Sign with APK Signature Schemes v1, v2, and v3:
   ```bash
   apksigner sign --ks /root/debug.keystore --ks-pass pass:android \
     --ks-key-alias androiddebugkey --key-pass pass:android \
     --v1-signing-enabled true --v2-signing-enabled true --v3-signing-enabled true \
     --out iris3d.apk iris3d.aligned.apk
   ```

For an exhaustive breakdown of compiler flags, memory layouts, and troubleshooting, read [Build Pipeline Reference](./references/build_pipeline.md).

---

## 3. Verification Runbook

Always verify the output APK before deploying to a physical device:

### 1. Verify 16KB Page Alignment
Check that `libmain.so` is aligned at an offset that is an exact multiple of 16384:
```bash
zipalign -c -v 16384 iris3d.apk | grep libmain.so
```
**Expected Output**:
```text
Verifying alignment of iris3d.apk (16384)...
... lib/arm64-v8a/libmain.so (OK - compressed) OR (OK)
Verification successful
```

### 2. Verify APK Signatures
Ensure all signature schemes pass verification:
```bash
apksigner verify -v iris3d.apk
```
**Expected Output**:
```text
Verifies
Verified using v1 scheme (JAR signing): true
Verified using v2 scheme (APK Signature Scheme v2): true
Verified using v3 scheme (APK Signature Scheme v3): true
```

---

## 4. UI & Touch Interaction Guidelines

When modifying 3D rendering or UI overlay in [`renderer.go`](file:///root/3dandroid/renderer.go) or [`ui.go`](file:///root/3dandroid/ui.go):

1. **Touch Disambiguation**:
   - Always call `ui.IsPointerOnUI(...)` before passing touch events to [`camera3d.go`](file:///root/3dandroid/camera3d.go).
   - Require tap duration $< 350\text{ ms}$ and travel distance $< 20\text{ px}$ to trigger 3D raycast picking (`renderer.PickPoint`).
2. **Generous Hit Radius**:
   - Mobile touch screens lack sub-pixel precision. Ray picking must use a minimum hit radius of $2.2 \times \text{BasePointSize}$.
3. **Android Back Button (`rl.KeyBack`)**:
   - Check `rl.IsKeyPressed(rl.KeyBack)`:
     1. Close active modals if open (`ui.ActiveModal = ModalNone`).
     2. Deselect inspected point if selected (`renderer.SelectedPoint = -1`).
     3. Exit app loop only when at root view state.

---

## 5. Deployment Locations

On this device/environment, compiled APKs should be synced to:
- `/sdcard/Download/iris3d.apk` (directly accessible by the Android package installer)
- `/data/data/com.termux/files/home/iris3d.apk` (accessible in Termux shell)
