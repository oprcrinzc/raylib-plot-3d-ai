---
name: raylib-go-android-workflow
description: >-
  Use this skill when developing, creating, checking, compiling, building, verifying,
  or iteratively editing Raylib-Go applications for Android (ARM64). Covers the
  complete create-check-build-verify-edit loop: project scaffolding (NativeActivity,
  manifests, assets), pre-build static checks and environment verification, 16KB
  page-aligned CGO cross-compilation (Android 15/16 Baklava), APK packaging and signing,
  binary alignment verification, and rapid edit-debug loops, completely decoupled
  from project-specific datasets.
---

# Raylib-Go Android Workflow (Create-Check-Build-Check-Edit Loop)

This skill provides a generalized, reproducible developer workflow for building and shipping **Raylib-Go applications on Android ARM64**, with strict compliance for **Android 15 & 16 (Baklava) 16KB memory page alignment**.

---

## Quick Reference: The Inner Loop

Phase | Description | Key Command / Script
:--- | :--- | :---
**1. CREATE** | Scaffold or initialize project structure | `./scripts/scaffold_project.sh <dir> <package> <title>`
**2. CHECK (Pre)** | Validate toolchain and code correctness | `./scripts/check_env.sh` & `go vet ./...`
**3. BUILD** | Compile ARM64 `.so` with 16KB alignment, package & sign APK | `./scripts/build_apk.sh --name <app_name>`
**4. CHECK (Post)** | Verify 16KB page boundary & multi-scheme signatures | `./scripts/verify_apk.sh <app.apk>`
**5. EDIT & DEBUG** | Rapid iterative edits, fast rebuilds & device logging | [Inner Loop Guide](./references/inner_loop.md)

---

## 1. Phase 1: CREATE (Project Scaffolding)

Every Raylib-Go Android application requires a minimal project structure:

```text
my-raylib-app/
├── main.go                     # Entrypoint with rl.SetMain(main)
├── AndroidManifest.xml         # Declares NativeActivity & lib_name="main"
├── res/
│   ├── values/strings.xml      # App display label
│   └── drawable/icon.png       # App launcher icon
├── assets/                     # (Optional) Textures, audio, fonts, shaders
└── go.mod                      # Go module requiring github.com/gen2brain/raylib-go/raylib
```

### Automated Scaffolding
Run the scaffolding script to generate a complete starter project:
```bash
./.agents/skills/raylib-go-android-workflow/scripts/scaffold_project.sh . com.example.myapp "My App" sensorLandscape
```

### Mandatory Android Initialization (`main.go`)
Android uses `android.app.NativeActivity`. The Go runtime **must** register the main function callback in `init()`:
```go
package main

import (
    "runtime"
    rl "github.com/gen2brain/raylib-go/raylib"
)

func init() {
    rl.SetMain(main) // CRITICAL: Registers Go main loop with Raylib Android C backend
}

func main() {
    rl.SetConfigFlags(rl.FlagVsyncHint)
    rl.InitWindow(0, 0, "My App") // (0,0) uses native full screen resolution on Android
    defer rl.CloseWindow()
    rl.SetTargetFPS(60)

    for !rl.WindowShouldClose() {
        // Android Back Button handling
        if runtime.GOOS == "android" && rl.IsKeyPressed(rl.KeyBack) {
            break
        }

        rl.BeginDrawing()
        rl.ClearBackground(rl.RayWhite)
        rl.DrawText("Hello Raylib Android!", 40, 100, 24, rl.DarkGray)
        rl.EndDrawing()
    }
}
```

### Manifest Requirements (`AndroidManifest.xml`)
- `<application android:hasCode="false" ...>`: Prevents Android from expecting Java bytecode.
- `<meta-data android:name="android.app.lib_name" android:value="main" />`: Directs `NativeActivity` to `libmain.so`.
- Full reference template: [AndroidManifest.xml.template](./templates/AndroidManifest.xml.template).

---

## 2. Phase 2: CHECK (Pre-Build Validation)

Before triggering binary compilation, run pre-flight checks:

### 1. Verify Toolchain & Prerequisites
```bash
./.agents/skills/raylib-go-android-workflow/scripts/check_env.sh
```
Verifies Go, ARM64 Clang, AAPT, Zipalign, Apksigner, Patchelf, `android.jar`, and debug keystore.

### 2. Static Code Check & Unit Tests
```bash
# Verify Go syntax, types, and suspicious constructs
go vet ./...

# Run unit tests on host
go test -v ./...
```

---

## 3. Phase 3: BUILD (16KB Page-Aligned ARM64 Pipeline)

Modern Android (15 & 16 / Baklava) requires native libraries and APK zip entries to be **16KB page-aligned** (16384 bytes).

### Automated One-Shot Build
```bash
./.agents/skills/raylib-go-android-workflow/scripts/build_apk.sh --name myapp
```

### Step-by-Step Manual Pipeline
1. **CGO Cross-Compilation (`libmain.so`)**:
   ```bash
   CGO_ENABLED=1 GOOS=android GOARCH=arm64 \
   CC=/data/data/com.termux/files/usr/bin/clang \
   go build -v -buildmode=c-shared \
   -ldflags="-s -w -extldflags=-Wl,-soname,libmain.so,-wrap,fopen,-z,max-page-size=16384" \
   -o libmain.so .
   ```
2. **Sanitize ELF Headers (Strip RUNPATH/RPATH)**:
   ```bash
   patchelf --remove-rpath libmain.so
   ```
3. **Package Resources & Manifest**:
   ```bash
   aapt package -f -M AndroidManifest.xml -S res -I /root/android-sdk/android-34/android.jar -F myapp.unaligned.apk
   ```
4. **Embed Shared Library Uncompressed (`zip -0`)**:
   ```bash
   mkdir -p lib/arm64-v8a
   cp -f libmain.so lib/arm64-v8a/libmain.so
   zip -0 -u myapp.unaligned.apk lib/arm64-v8a/libmain.so
   ```
5. **16KB Zipaligning**:
   ```bash
   zipalign -f 16384 myapp.unaligned.apk myapp.aligned.apk
   ```
6. **Multi-Scheme Signing (v1, v2, v3)**:
   ```bash
   apksigner sign --ks /root/debug.keystore --ks-pass pass:android \
     --ks-key-alias androiddebugkey --key-pass pass:android \
     --v1-signing-enabled true --v2-signing-enabled true --v3-signing-enabled true \
     --out myapp.apk myapp.aligned.apk
   ```

Exhaustive compiler flag documentation: [Build Flags Reference](./references/build_flags.md).

---

## 4. Phase 4: CHECK (Post-Build Verification)

Always inspect the generated APK before transferring to a test device:

### Automated APK Verification
```bash
./.agents/skills/raylib-go-android-workflow/scripts/verify_apk.sh myapp.apk
```

### Manual Verification Checks
1. **16KB Page Alignment Offset**:
   ```bash
   zipalign -c -v 4 myapp.apk | grep "lib/arm64-v8a/libmain.so"
   ```
   *Verification Rule*: The starting byte offset must be divisible by 16384 (`offset % 16384 == 0`).
2. **Signature Schemes**:
   ```bash
   apksigner verify -v myapp.apk
   ```
   Ensure `v2 scheme: true` and `v3 scheme: true`.
3. **AAPT Badging**:
   ```bash
   aapt dump badging myapp.apk | grep -E "package:|application-label"
   ```

---

## 5. Phase 5: EDIT LOOP (Fast Iteration & Debugging)

### Fast Incremental Rebuilds
When modifying Go code without changing Android resources:
```bash
# Quick build without re-running tests or cleaning
./.agents/skills/raylib-go-android-workflow/scripts/build_apk.sh --name myapp --no-test
```

### Runtime Debugging via Logcat
Monitor Raylib, Go, and system logs:
```bash
# In adb shell or Termux:
logcat -c
logcat -s "Raylib:*" "GoLog:*" "DEBUG:*" "AndroidRuntime:*"
```
In Go code, emit logs with `rl.TraceLog(rl.LogInfo, "...")`, which routes directly to Android's logcat under tag `Raylib`.

### Troubleshooting Matrix
Problem | Root Cause | Resolution
:--- | :--- | :---
`dlopen failed: ... has bad ELF alignment` | Library compiled without 16KB page size. | Ensure `-z,max-page-size=16384` is in `-extldflags` and verify with `verify_apk.sh`.
Black screen or immediate silent exit | Missing `rl.SetMain(main)`. | Add `func init() { rl.SetMain(main) }` to `main.go`.
Dynamic Linker rejects RUNPATH | Residual RPATH injected by host compiler. | Run `patchelf --remove-rpath libmain.so`.
`ClassNotFoundException: NativeActivity` | `hasCode` set to true or missing NativeActivity. | Add `android:hasCode="false"` in `<application>` tag of `AndroidManifest.xml`.
Library `main` not found | SONAME mismatch. | Verify `-Wl,-soname,libmain.so` and `<meta-data android:name="android.app.lib_name" android:value="main" />`.

For in-depth gesture input patterns and lifecycle management, see [Inner Loop Guide](./references/inner_loop.md).

---

## Bundled Tools Index

- [Environment Checker](./scripts/check_env.sh) (`check_env.sh`)
- [Project Scaffolder](./scripts/scaffold_project.sh) (`scaffold_project.sh`)
- [Build Pipeline](./scripts/build_apk.sh) (`build_apk.sh`)
- [APK Verifier](./scripts/verify_apk.sh) (`verify_apk.sh`)
- [Main Go Template](./templates/main.go.template) (`main.go.template`)
- [Manifest Template](./templates/AndroidManifest.xml.template) (`AndroidManifest.xml.template`)
- [Build Flags Reference](./references/build_flags.md)
- [Inner Loop Guide](./references/inner_loop.md)
