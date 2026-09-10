# Raylib-Go Android Build Flags & 16KB Alignment Reference

This guide details the exact compilation flags, linker arguments, and binary packaging specifications required to build stable, high-performance Raylib-Go applications for modern Android ARM64 devices (including Android 15 and 16 Baklava).

---

## 1. The Android 15/16 (Baklava) 16KB Page Requirement

Modern ARM64 Android devices (such as the Nothing Phone 3a, modern Pixel devices, and MediaTek/Snapdragon platforms) use **16KB memory page sizes** instead of the legacy 4KB page size.

### Failure Mode (4KB vs 16KB)
When a native ELF shared library (`.so`) is compiled with standard 4KB segment boundaries:
1. Android dynamic linker (`/linker64`) attempts to load the ELF segments with `mmap()`.
2. Because the segment alignments in the ELF headers or inside the uncompressed ZIP entry are not multiples of 16384 bytes, `mmap()` cannot cleanly map page tables.
3. Android immediately terminates the process before any Go or Raylib code executes:
   ```text
   dlopen failed: "/data/app/.../lib/arm64/libmain.so" has bad ELF alignment
   ```

### Full Two-Stage Alignment Compliance
Compliance requires **both** steps:
1. **ELF Binary Alignment**: Compile `libmain.so` with `-Wl,-z,max-page-size=16384`.
2. **ZIP Container Alignment**: Store `libmain.so` uncompressed (`zip -0`) and align with `zipalign -f 16384`.

---

## 2. Compiler & Linker Flag Breakdown

The standard CGO compilation command:
```bash
CGO_ENABLED=1 GOOS=android GOARCH=arm64 \
CC=/data/data/com.termux/files/usr/bin/clang \
go build -v -buildmode=c-shared \
-ldflags="-s -w -extldflags=-Wl,-soname,libmain.so,-wrap,fopen,-z,max-page-size=16384" \
-o libmain.so .
```

Parameter | Purpose & Rationale
:--- | :---
`CGO_ENABLED=1` | Enables CGO bindings required to link Go code with Raylib's C headers and Android OpenGL ES / EGL libraries.
`GOOS=android` | Targets the Android Bionic C runtime rather than GNU glibc or standard Linux.
`GOARCH=arm64` | Targets 64-bit ARM (`aarch64`), standard on all modern Android devices.
`CC=.../clang` | Points to the Android-targeted Clang compiler configured with Bionic headers and NDK sysroot.
`-buildmode=c-shared` | Instructs the Go toolchain to emit a C-compatible shared object (`.so`) with exported symbol tables rather than a standard executable binary.
`-s -w` | Strips symbol tables (`-s`) and DWARF debug information (`-w`), reducing the `.so` binary footprint from ~15 MB down to ~3.5 MB.
`-Wl,-soname,libmain.so` | Sets `DT_SONAME` in the ELF header to `libmain.so`. Android's `NativeActivity` looks for this name as specified in `AndroidManifest.xml` (`android:value="main"`).
`-Wl,-wrap,fopen` | Intercepts standard C `fopen()` calls. On Android, assets packaged inside the APK must be routed through the Android asset manager rather than raw Linux filesystem calls.
`-Wl,-z,max-page-size=16384` | Forces GNU/LLD linker to align all `PT_LOAD` ELF segments to 16KB boundaries for Android 15/16 Baklava compatibility.

---

## 3. ELF Sanitization with `patchelf`

Modern Android dynamic linkers reject ELF binaries containing `DT_RUNPATH` or `DT_RPATH` tags under modern target API levels:
```bash
patchelf --remove-rpath libmain.so
```
Running `patchelf --remove-rpath` strips residual build-host paths injected by the compiler, avoiding dynamic linker security aborts.

---

## 4. APK Assembly & Packaging Pipeline

### Step A: Resource Packaging (AAPT)
```bash
aapt package -f -M AndroidManifest.xml -S res [-A assets] \
  -I /root/android-sdk/android-34/android.jar \
  -F app.unaligned.apk
```
- `-f`: Overwrite existing output file.
- `-M`: Path to `AndroidManifest.xml`.
- `-S`: Path to resource directory (`res/`).
- `-A`: Optional path to application assets (`assets/` for textures, fonts, audio).
- `-I`: Path to Android platform definitions (`android.jar`).
- `-F`: Output unaligned APK container.

### Step B: Embedding Native Library Uncompressed
```bash
mkdir -p lib/arm64-v8a
cp -f libmain.so lib/arm64-v8a/libmain.so
zip -0 -u app.unaligned.apk lib/arm64-v8a/libmain.so
```
**CRITICAL**: The `-0` flag ensures the `.so` file is stored with zero compression. Uncompressed storage is required for direct `mmap()` execution from APK and for `zipalign` 16KB offsets to succeed.

### Step C: 16KB Zipaligning
```bash
zipalign -f 16384 app.unaligned.apk app.aligned.apk
```
Ensures all uncompressed data in the ZIP begins on an exact multiple of 16384 bytes.

### Step D: Multi-Scheme Signing (v1, v2, v3)
```bash
apksigner sign --ks /root/debug.keystore --ks-pass pass:android \
  --ks-key-alias androiddebugkey --key-pass pass:android \
  --v1-signing-enabled true \
  --v2-signing-enabled true \
  --v3-signing-enabled true \
  --out app.apk app.aligned.apk
```
Enabling v1, v2, and v3 schemes ensures universal compatibility from older Android versions up through Android 16.
