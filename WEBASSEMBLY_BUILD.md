# WebAssembly Build Instructions

This document explains how to build IMGE games for WebAssembly using Emscripten.

## Prerequisites

### Install Emscripten

```bash
# Clone Emscripten SDK
git clone https://github.com/emscripten-core/emsdk.git
cd emsdk

# Install and activate latest SDK
./emsdk install latest
./emsdk activate latest

# Source environment variables
source ./emsdk_env.sh
```

### Verify Installation

```bash
emcc --version
emcmake --version
```

## Building for WebAssembly

### Quick Build

```bash
# From project root
./build-web.sh
```

### Manual Build

```bash
# Create build directory
mkdir -p build-web
cd build-web

# Configure with Emscripten
emcmake cmake \
    -DIMGE_WITH_WEBGL=ON \
    -DIMGE_WITH_SDL2=OFF \
    -DIMGE_BUILD_TESTS=OFF \
    ..

# Build
emmake make -j$(nproc)
```

## Running WebAssembly Games

### Option 1: Using emrun (Recommended)

```bash
cd build-web/examples
emrun full_game_web.html
```

### Option 2: Using Python HTTP Server

```bash
cd build-web/examples
python3 -m http.server 8000

# Then open in browser:
# http://localhost:8000/full_game_web.html
# http://localhost:8000/simple_game_web.html
```

### Option 3: Using Any HTTP Server

```bash
# Node.js http-server
npx http-server build-web/examples

# Or PHP built-in server
cd build-web/examples
php -S localhost:8000
```

## WebAssembly vs Desktop

### Desktop (SDL2)
```bash
cd build
cmake ..
make
./examples/full_game
```

### WebAssembly (Emscripten)
```bash
cd build-web
emcmake cmake -DIMGE_WITH_WEBGL=ON ..
emmake make
# Open full_game_web.html in browser
```

## Controls

Both desktop and web versions use the same controls:
- **W, A, S, D** - Move
- **ESC** - Quit/Stop

## Output Files

After building, you'll find:
```
build-web/examples/
├── full_game_web.html      # Main HTML file (opens in browser)
├── full_game_web.js        # Compiled JavaScript
├── full_game_web.wasm      # WebAssembly module
├── simple_game_web.html
├── simple_game_web.js
└── simple_game_web.wasm
```

## Troubleshooting

### "emcc: command not found"
Make sure you sourced Emscripten environment:
```bash
source /path/to/emsdk/emsdk_env.sh
```

### "Cannot find module 'nlohmann_json'"
Install nlohmann_json:
```bash
# Ubuntu/Debian
sudo apt install nlohmann-json3-dev

# Fedora
sudo dnf install json-devel

# Or from source
git clone https://github.com/nlohmann/json.git
cd json
mkdir build && cd build
cmake ..
make && sudo make install
```

### Game doesn't load in browser
1. Check browser console for errors
2. Make sure you're using a local HTTP server (not file://)
3. Verify all files (.html, .js, .wasm) are in the same directory

### Performance issues
- Game runs at 60 FPS using `requestAnimationFrame`
- If slow, check browser console for warnings
- Try closing other tabs

## Browser Compatibility

Tested on:
- Chrome/Edge 90+
- Firefox 88+
- Safari 14+

**Note:** WebAssembly requires modern browser support.

## Architecture

```
Desktop (SDL2)          WebAssembly (Emscripten)
    │                         │
    ├─ SDL2Engine            ├─ WebGLEngine
    ├─ SDL2Renderer          ├─ EmscriptenRenderer
    ├─ SDL2Input            ├─ EmscriptenInput
    └─ SDL2Audio            └─ (Browser Audio - TODO)
    │                         │
    └─ Same IMGE Core ────────┘
```

## Future Enhancements

- [ ] Audio support (Web Audio API)
- [ ] Texture loading (images)
- [ ] Touch controls for mobile
- [ ] PWA support (offline mode)
- [ ] Multiplayer support (WebSockets)
