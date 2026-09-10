# Android 15 & 16 (Baklava) Build Pipeline & 16KB Alignment Guide

This reference document explains the mechanics, requirements, and troubleshooting steps for compiling Go and Raylib (`c-shared`) for modern Android ARM64 devices.

---

## 1. Why 16KB Page Alignment is Mandatory

Starting in **Android 15** and enforced strictly on **Android 16 (Baklava)**, the Android operating system supports and commonly runs on ARM64 hardware configured with **16KB memory page sizes** (instead of legacy 4KB pages), as seen on devices like the Nothing Phone 3a, modern Pixel devices, and newer MediaTek/Snapdragon platforms.

### The Problem with 4KB Binaries
When an ELF shared library (`.so`) is compiled with default 4KB page alignment:
- The virtual memory addresses of loadable ELF segments (`PT_LOAD`) are aligned to 4096 bytes.
- When `dlopen()` or Android's dynamic linker attempts to `mmap()` the segments into a process running with 16KB pages, the page offsets overlap or violate alignment constraints.
- The OS immediately aborts execution with a fatal linker error:
  ```text
  dlopen failed: "/data/app/.../lib/arm64/libmain.so" has bad ELF alignment
  ```

### The Solution: Two-Pronged 16KB Compliance
1. **Linker Page Boundary**:
   Pass `-z,max-page-size=16384` to the compiler/linker so all ELF segments (`LOAD`) are spaced by at least 16384 bytes.
2. **ZIP Entry Alignment**:
   Pass `zipalign -f 16384` so the start offset of `libmain.so` inside the `.apk` file is an exact multiple of 16384 bytes, allowing zero-copy memory mapping directly from the APK file (`android:extractNativeLibs="false"` or uncompressed storage).

---

## 2. Deep Dive into Compiler & Linker Flags

```bash
CGO_ENABLED=1 GOOS=android GOARCH=arm64 \
CC=/data/data/com.termux/files/usr/bin/clang \
go build -v -buildmode=c-shared \
-ldflags="-s -w -extldflags=-Wl,-soname,libmain.so,-wrap,fopen,-z,max-page-size=16384" \
-o libmain.so .
```

Flag | Rationale
:--- | :---
`CGO_ENABLED=1` | Required because Raylib-Go binds to native C Raylib headers and OpenGL ES implementations via CGO.
`GOOS=android GOARCH=arm64` | Targets the Android OS on 64-bit ARM architecture.
`CC=.../clang` | Points to the Android/Termux Clang compiler toolchain containing Bionic libc headers and NDK sysroots.
`-buildmode=c-shared` | Generates a C-compatible shared object (`.so`) with an `init()` and exported symbols rather than a standalone Linux executable.
`-s -w` | Strips symbol tables and DWARF debugging information, reducing binary size significantly (from ~15MB to ~3.5MB).
`-soname,libmain.so` | Sets `DT_SONAME` in the ELF header to `libmain.so`, allowing `NativeActivity` to locate and bind the library via `<meta-data android:name="android.app.lib_name" android:value="main" />`.
`-wrap,fopen` | Wraps C stdio `fopen()` calls. Raylib uses standard C file operations, and on Android, file requests inside APK assets must redirect to the Android asset manager or APK filesystem.
`-z,max-page-size=16384` | Sets maximum ELF segment page size to 16KB for Android 15 & 16 compatibility.

---

## 3. ELF Sanitization with `patchelf`

Modern Android dynamic linkers enforce strict security policies regarding search paths:
- Modern Android linker (`/linker64`) rejects libraries containing `DT_RUNPATH` or `DT_RPATH` under certain target API levels.
- By running:
  ```bash
  patchelf --remove-rpath libmain.so
  ```
  any residual build-machine paths injected by Clang/Go are cleanly stripped.

---

## 4. AAPT Packaging & Library Ingestion

1. **Packaging Resources**:
   ```bash
   aapt package -f -M AndroidManifest.xml -S res -I /root/android-sdk/android-34/android.jar -F iris3d.unaligned.apk
   ```
   Uses Android SDK 34 (Android 14) platform definitions to compile strings, icons, and AndroidManifest into an initial unaligned zip.

2. **Embedding the Shared Library**:
   ```bash
   mkdir -p lib/arm64-v8a
   cp -f libmain.so lib/arm64-v8a/libmain.so
   zip -0 -u iris3d.unaligned.apk lib/arm64-v8a/libmain.so
   ```
   **CRITICAL**: Note the `-0` flag. Compression must be disabled (`store` mode) so `zipalign` can guarantee byte-exact page boundaries in uncompressed storage.

3. **Zipaligning**:
   ```bash
   zipalign -f 16384 iris3d.unaligned.apk iris3d.aligned.apk
   ```

---

## 5. APK Signing Schemes (v1, v2, v3)

Modern Android installs fail if only legacy v1 (JAR signing) is used.
Using `apksigner` with `--v1-signing-enabled true --v2-signing-enabled true --v3-signing-enabled true`:
- **v1**: JAR signing (backwards compatibility for older tools).
- **v2**: Whole-file binary signature (fast verification, tamper-proof).
- **v3**: Key rotation support and modern Android security requirements.
