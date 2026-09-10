# Fast Edit-Build-Debug Inner Loop Guide for Raylib-Go Android

This document details the day-to-day developer inner loop when writing, testing, and debugging Raylib-Go applications on Android.

---

## 1. Fast Incremental Rebuilds

During active coding, rebuilding the full APK does not require re-running AAPT if your `AndroidManifest.xml` and `res/` assets have not changed:

### Fast Path (Code-Only Change):
```bash
# 1. Recompile shared library (~3-5 seconds)
CGO_ENABLED=1 GOOS=android GOARCH=arm64 \
CC=/data/data/com.termux/files/usr/bin/clang \
go build -v -buildmode=c-shared \
-ldflags="-s -w -extldflags=-Wl,-soname,libmain.so,-wrap,fopen,-z,max-page-size=16384" \
-o libmain.so .

# 2. Sanitize ELF headers
patchelf --remove-rpath libmain.so

# 3. Update library inside unaligned APK
cp -f libmain.so lib/arm64-v8a/libmain.so
zip -0 -u app.unaligned.apk lib/arm64-v8a/libmain.so

# 4. Re-align and sign
zipalign -f 16384 app.unaligned.apk app.aligned.apk
apksigner sign --ks /root/debug.keystore --ks-pass pass:android \
  --ks-key-alias androiddebugkey --key-pass pass:android \
  --v1-signing-enabled true --v2-signing-enabled true --v3-signing-enabled true \
  --out app.apk app.aligned.apk
```
This reduces cycle turnaround time to ~5 seconds.

---

## 2. Touch and Gesture Input Patterns

On Android, touch screens interact with Raylib's mouse and touch APIs:

### Single Touch Detection
- Primary touch down: `rl.IsMouseButtonDown(rl.MouseLeftButton)`
- Primary touch pressed: `rl.IsMouseButtonPressed(rl.MouseLeftButton)`
- Primary touch released: `rl.IsMouseButtonReleased(rl.MouseLeftButton)`
- Touch position: `pos := rl.GetMousePosition()`

### Multi-Touch Gestures (Pinch / Pan)
Raylib provides native touch point index queries:
```go
touchCount := rl.GetTouchPointCount()
if touchCount >= 2 {
    t0 := rl.GetTouchPosition(0)
    t1 := rl.GetTouchPosition(1)
    currentDist := Vector2Distance(t0, t1)
    // Compare against previous frame distance for pinch-to-zoom
}
```

### Tap vs Drag Disambiguation
To distinguish between an interactive tap (e.g. clicking a button or 3D object) and a drag/orbit gesture:
```go
var touchStartPos rl.Vector2
var touchStartTime float64

if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
    touchStartPos = rl.GetMousePosition()
    touchStartTime = rl.GetTime()
}

if rl.IsMouseButtonReleased(rl.MouseLeftButton) {
    tapDuration := rl.GetTime() - touchStartTime
    dist := rl.Vector2Distance(touchStartPos, rl.GetMousePosition())
    if tapDuration < 0.35 && dist < 20.0 {
        // Trigger tap action (e.g. Raycast pick or button action)
    }
}
```

---

## 3. Android Back Button & App Lifecycle

### Android Back Button (`rl.KeyBack`)
Android devices deliver back-button presses as key code `rl.KeyBack`:
```go
if runtime.GOOS == "android" && rl.IsKeyPressed(rl.KeyBack) {
    // 1. Close open modal or dialog
    if isModalOpen {
        isModalOpen = false
    // 2. Clear current selection
    } else if selectedItem >= 0 {
        selectedItem = -1
    // 3. Exit loop to close application
    } else {
        break
    }
}
```

### Screen Resolution & Orientation Changes
Always query screen dimensions dynamically in each frame rather than caching static values:
```go
screenWidth := float32(rl.GetScreenWidth())
screenHeight := float32(rl.GetScreenHeight())
```
This ensures responsive layout if the screen rotates or on-screen keyboards appear.

---

## 4. Debugging & Device Logcat

When testing on an Android device or inside Termux, observe Raylib and runtime logs via `logcat`:

### Filter for Raylib & Go Output
```bash
# In adb shell or Termux:
logcat -c # Clear log buffer
logcat -s "Raylib:*" "GoLog:*" "DEBUG:*" "AndroidRuntime:*"
```

### Logging from Go
Use standard `fmt.Println()` or Raylib's logger:
```go
rl.TraceLog(rl.LogInfo, "Scene initialized: %d items loaded", count)
```
Raylib routes `TraceLog` directly to Android's `__android_log_print` with tag `Raylib`.

---

## 5. Troubleshooting Common Traps

Trap | Cause | Fix
:--- | :--- | :---
**`dlopen failed: ... has bad ELF alignment`** | Library compiled with default 4KB alignment on Android 15/16. | Add `-z,max-page-size=16384` to linker flags and verify with `zipalign -c -v 4`.
**Black screen / Immediate exit** | Go `main` was not registered with Raylib's C backend. | Ensure `func init() { rl.SetMain(main) }` is present in `main.go`.
**Dynamic Linker rejects RUNPATH** | Modern Android linker disallows `DT_RUNPATH` / `DT_RPATH`. | Run `patchelf --remove-rpath libmain.so` before packaging.
**`ClassNotFoundException: android.app.NativeActivity`** | Missing `android:hasCode="false"` in manifest. | Ensure `<application android:hasCode="false" ...>` is in `AndroidManifest.xml`.
**`NativeActivity: Library 'main' not found`** | SONAME mismatch or missing metadata. | Check `<meta-data android:name="android.app.lib_name" android:value="main" />` and `-Wl,-soname,libmain.so`.
**Asset loading returns null / empty** | File accessed using standard Linux file paths instead of asset manager. | Link with `-wrap,fopen` and package assets via `aapt package -A assets ...`.
