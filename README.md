# Iris 3D & 4D Data Visualizer (Raylib-Go on Android)

An interactive, high-performance 3D & 4D scientific data visualizer written in **Go** using **raylib-go**, targeting **Android (Nothing Phone 3a / Android 16 / MediaTek ARM64)**.

---

## Key Features

1. **Iris Dataset 3D & 4D Plotting**:
   - Visualizes Fisher's classic 150-sample Iris dataset in full 3D space with high-precision sphere rendering.
   - 4-feature multidimensional representation:
     - Sepal Length, Sepal Width, Petal Length mapped to 3D X, Y, Z coordinates.
     - Petal Width / Species mapped to the 4th Dimension (Color, Sphere Size, Dual, or Time Pulse animation).

2. **Custom & Multi-Dataset Importer**:
   - Built-in embedded datasets:
     - **Fisher's Iris** (150 samples, 4 features, 3 species)
     - **Wine Recognition** (178 samples, 13 chemical features, 3 cultivars)
     - **Palmer Penguins** (344 samples, multiple physical features, 3 species)
     - **Synthetic 4D Hypersurface** (300 samples, Clifford Torus & clustered manifolds)
   - **External CSV Importer**:
     - Scans device storage (`/sdcard/Download`, `/sdcard/Documents`, etc.).
     - Parses arbitrary CSV/TSV data with automatic header and numeric column detection.
     - In-app demo generator for custom multi-dimensional data.

3. **Flexible 4D Visual Encodings**:
   - **Mode 1: Categorical Palette**: High-contrast, vibrant distinct colors for class labels (Setosa = Cyan, Versicolor = Orange, Virginica = Neon Violet).
   - **Mode 2: Continuous Colormap**: Smooth gradient colormaps (**Viridis**, **Plasma**, **Cool-Warm**, **Turbo**) mapped to the 4th dimension value.
   - **Mode 3: Sphere Size**: Point radius dynamically scales with 4th dimension magnitude.
   - **Mode 4: Dual (Color + Size)**: Simultaneous color and size modulation.
   - **Mode 5: Time Pulse**: Spheres oscillate in scale and brightness based on 4D phase/frequency.

4. **Interactive Touch & Camera Controls (Android-Optimized)**:
   - **Single-finger drag**: Smooth 3D orbit around data center with inertia.
   - **Pinch-to-zoom**: Two-finger pinch in/out to zoom in and out.
   - **Two-finger drag**: Pan camera center in screen plane.
   - **Turntable / Auto-Rotate**: Smooth continuous turntable rotation toggle.
   - **On-screen Zoom & Reset Buttons**: Direct touch controls for quick adjustments.
   - **3D Ray Picking & Inspection**: Tap any data point sphere in 3D to inspect exact numeric values and class labels in a floating HUD card.

5. **Android 16 (Baklava) & Nothing Phone 3a Compatibility**:
   - Native ARM64 (`arm64-v8a`) architecture.
   - Compliant with **16KB page size alignment** required for Android 15 & 16 (`zipalign -f 16384` verified).
   - Clean ELF binary with no external dependency leaks (`patchelf --remove-rpath`).
   - Signed with Android APK Signature Schemes v2 and v3.
   - Fullscreen immersive `NativeActivity` rendering via OpenGL ES 2.0 / 3.0.

---

## File Structure

- [`main.go`](file:///root/3dandroid/main.go): Application entry point, Raylib lifecycle, touch gesture dispatcher, main loop.
- [`dataset.go`](file:///root/3dandroid/dataset.go): CSV parsing, dataset structures, embedded datasets, normalization logic.
- [`camera3d.go`](file:///root/3dandroid/camera3d.go): Spherical orbit camera with touch gestures, pinch-to-zoom, and inertia.
- [`renderer.go`](file:///root/3dandroid/renderer.go): 3D bounding box, coordinate grid, colored axes, sphere rendering, drop lines, ray picking.
- [`ui.go`](file:///root/3dandroid/ui.go): Touch-friendly UI controls, dataset modal, axes mapping modal, view options modal, inspection card.
- [`colormap.go`](file:///root/3dandroid/colormap.go): Color gradients (Viridis, Plasma, CoolWarm, Turbo) and categorical palettes.
- [`build_apk.sh`](file:///root/3dandroid/build_apk.sh): Automated build script that compiles CGO to ARM64 `.so`, aligns to 16KB, packages, and signs the APK.
- [`iris3d.apk`](file:///root/3dandroid/iris3d.apk): Ready-to-install Android package (3.4 MB).

---

## Rebuilding the APK

To rebuild the APK at any time:
```bash
cd /root/3dandroid
./build_apk.sh
```

The output will be verified and written to `/root/3dandroid/iris3d.apk`.
