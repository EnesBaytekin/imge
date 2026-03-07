#!/bin/bash
# WebAssembly Build Script for IMGE
# Usage: ./build-web.sh

set -e  # Exit on error

echo "=== IMGE WebAssembly Build Script ==="

# Check if emscripten is installed
if ! command -v emcmake &> /dev/null; then
    echo "Error: Emscripten not found!"
    echo "Please install Emscripten:"
    echo "  git clone https://github.com/emscripten-core/emsdk.git"
    echo "  cd emsdk"
    echo "  ./emsdk install latest"
    echo "  ./emsdk activate latest"
    echo "  source ./emsdk_env.sh"
    exit 1
fi

# Create build directory
BUILD_DIR="build-web"
mkdir -p "$BUILD_DIR"
cd "$BUILD_DIR"

echo "Configuring CMake for Emscripten..."
emcmake cmake \
    -DIMGE_WITH_WEBGL=ON \
    -DIMGE_WITH_SDL2=OFF \
    -DIMGE_BUILD_TESTS=OFF \
    ..

echo "Building WebAssembly version..."
emmake make -j$(nproc)

echo ""
echo "=== Build Complete! ==="
echo ""
echo "WebAssembly files are in: $BUILD_DIR/examples/"
echo ""
echo "To run the game in a browser:"
echo "  1. Start a local server in the build directory:"
echo "     cd $BUILD_DIR/examples"
echo "     python3 -m http.server 8000"
echo ""
echo "  2. Open in browser:"
echo "     http://localhost:8000/full_game_web.html"
echo "     http://localhost:8000/simple_game_web.html"
echo ""
echo "Or use emscripten's serve command:"
echo "  cd $BUILD_DIR/examples"
echo "  emrun full_game_web.html"
