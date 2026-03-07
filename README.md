# IMGE - Minimal Game Engine

**IMGE** is a lightweight, platform-agnostic 2D game engine written in modern C++ (C++20). Build your game once, run anywhere!

## 🎯 Philosophy

**SEPARATION OF CONCERNS:**
- **This Repository:** IMGE Engine library only
- **Your Project:** Your game + build configuration

**Key Point:** Your game code is **100% platform-agnostic**. Platform selection happens at **build time**, not in code!

## ✨ Features

- **Platform-Agnostic Core:** Write game logic without platform-specific code
- **Multiple Platforms:**
  - SDL2 (Desktop: Linux, Windows, macOS)
  - WebAssembly (Browser: Chrome, Firefox, Safari)
- **Object-Component System:** Flat hierarchy, O(1) lookups
- **Abstract Interfaces:** No implementation dependencies in your game code
- **Modern C++:** C++20 with smart pointers and RAII

## 📦 Repository Structure

```
imge/  (this repository - ENGINE ONLY)
├── include/imge/          # Abstract interfaces
├── src/                   # Core implementation
├── src/impl/sdl2/         # SDL2 implementation
├── src/impl/emscripten/   # WebAssembly implementation
└── templates/             # Example templates (copy to your project)

your_game/  (YOUR project)
├── CMakeLists.txt         # Your build config
├── main.cpp                # Your game entry
└── MyComponents.hpp       # Your game logic
```

## 🚀 Quick Start

### Step 1: Build IMGE Engine

```bash
git clone <repo-url>
cd imge

# Build for Desktop
mkdir build && cd build
cmake ..
make -j$(nproc)

# (Optional) Build for Web
../build-web.sh
```

**Generated libraries:**
```
build/libimge_core.a      # Core engine (always needed)
build/libimge_sdl2.a      # SDL2 implementation

build-web/libimge_webgl.a # WebAssembly implementation
```

### Step 2: Create Your Game

```bash
# Copy template to a NEW LOCATION (important!)
cp -r imge/templates/minimal_game ~/my_game
cd ~/my_game

# Edit your game
vim main.cpp
vim MyPlayerController.hpp

# Build for Desktop
cmake -B build -DIMGE_DIR=~/imge -DBUILD_DESKTOP=ON
cmake --build build
./build/minimal_game

# Build for Web (optional)
emcmake cmake -B build-web -DIMGE_DIR=~/imge -DBUILD_WEB=ON
emmake cmake --build build-web
cd build-web && emrun minimal_game.html
```

**IMPORTANT:** Always copy template to a **new directory**, don't build inside `imge/`!

## 📖 How It Works

### Your Game Code (100% Abstract)

```cpp
// MyPlayerController.hpp
#include "imge/core/Component.hpp"

class MyPlayerController : public imge::Component {
public:
    void onUpdate(imge::Object* owner) override {
        // ONLY abstract interfaces - no SDL2, no Emscripten!
        auto* input = imge::Input::getInstance();

        if (input->isKeyPressed(imge::Key::W)) {
            owner->y -= speed * dt;  // Works on ALL platforms!
        }
    }
};
```

### Build Selection (CMakeLists.txt)

```cmake
# Desktop build
option(BUILD_DESKTOP "Build for desktop (SDL2)" ON)
if(BUILD_DESKTOP AND NOT EMSCRIPTEN)
    target_link_libraries(minimal_game
        ${IMGE_DIR}/build/libimge_core.a
        ${IMGE_DIR}/build/libimge_sdl2.a
        SDL2::SDL2
    )
    target_compile_definitions(minimal_game PRIVATE IMGE_USE_SDL2)
endif()

# Web build
option(BUILD_WEB "Build for web (WebAssembly)" OFF)
if(BUILD_WEB AND EMSCRIPTEN)
    target_link_libraries(minimal_game
        ${IMGE_DIR}/build-web/libimge_webgl.a
        nlohmann_json::nlohmann_json
    )
    target_compile_definitions(minimal_game PRIVATE IMGE_USE_WEBGL)
endif()
```

## 🎮 Complete Example

**main.cpp (Your Game):**
```cpp
#include "imge/config.hpp"  // Auto-selects platform
#include "MyPlayerController.hpp"

int main() {
    // EngineImpl = SDL2Engine OR WebGLEngine
    // Selected by CMAKE! Not by code!
    imge::EngineImpl engine;

    auto scene = std::make_shared<imge::Scene>();
    // ... create objects ...

    engine.run();
}
```

**Build & Run:**
```bash
# Desktop
cmake -DBUILD_DESKTOP=ON ..
./minimal_game  # SDL2

# Web
emcmake -DBUILD_WEB=ON ..
emrun minimal_game.html  # WebGL
```

## 🏗️ Architecture

```
┌─────────────────────────────────────┐
│     Your Game Code                  │
│  (100% Platform-Agnostic)           │
│  - Custom Components                │
│  - Game Logic                      │
│  - Asset References                │
└─────────────────────────────────────┘
                ↓
        CMake Build Selection
        ↓                   ↓
┌──────────────┐      ┌──────────────┐
│ SDL2 Build   │      │ WebGL Build  │
│              │      │              │
┌──────────────┐      ┌──────────────┐
│ SDL2Engine   │      │ WebGLEngine  │
│ + SDL2Render │      │ + Canvas2D   │
│ + SDL2Input  │      │ + Browser    │
└──────────────┘      └──────────────┘
```

## 📚 Documentation

- **[DESIGN.md](DESIGN.md)** - Detailed engine design
- **[WEBASSEMBLY_BUILD.md](WEBASSEMBLY_BUILD.md)** - WebAssembly guide

## 🔧 Building IMGE

### Prerequisites

```bash
# Fedora
sudo dnf install cmake gcc-c++ ninja-build \
    SDL2-devel SDL2_image-devel SDL2_mixer-devel \
    nlohmann-json-devel

# Ubuntu
sudo apt install cmake build-essential ninja-build \
    libsdl2-dev libsdl2-image-dev libsdl2-mixer-dev \
    nlohmann-json3-dev

# macOS
brew install cmake ninja sdl2 sdl2_image sdl2_mixer nlohmann-json
```

### Build Commands

```bash
# Desktop (default)
mkdir build && cd build
cmake ..
make -j$(nproc)

# WebAssembly (requires emscripten)
./build-web.sh
```

## 🎯 Platform Comparison

| Aspect | Desktop (SDL2) | Web (WebAssembly) |
|--------|-----------------|---------------------|
| **Platforms** | Linux, Windows, macOS | Any modern browser |
| **Renderer** | SDL2 Renderer | HTML5 Canvas 2D |
| **Input** | SDL2 Events | Browser Events |
| **Performance** | Native | Near-native (WASM) |
| **Output** | Native binary | .html + .wasm |
| **Build Tool** | cmake | emcmake |
| **File Size** | ~500 KB | ~1 MB |

## 📝 Templates

**Available Templates:**

- **`templates/minimal_game/`** - Minimal working example
  - Single component (WASD movement)
  - Shows platform-agnostic code
  - Both desktop and web builds

**Reference Examples (not built, just for learning):**
- **`templates/simple_game/`** - Rectangle rendering
- **`templates/full_game/`** - Collision, enemies, HUD

## 🚀 Usage Workflow

```bash
# 1. Clone and build IMGE engine (once)
git clone <repo-url>
cd imge && mkdir build && cd build
cmake .. && make -j$(nproc)
../build-web.sh  # Optional

# 2. Create your game (repeat for each game)
cp -r imge/templates/minimal_game ~/my_first_game
cd ~/my_first_game

# 3. Edit game logic
# (Your game code here - platform agnostic!)

# 4. Build for your target platform
cmake -B build -DIMGE_DIR=~/imge -DBUILD_DESKTOP=ON
cmake --build build
./build/minimal_game

# OR build for web
emcmake cmake -B build-web -DIMGE_DIR=~/imge -DBUILD_WEB=ON
emmake cmake --build build-web
cd build-web && emrun minimal_game.html
```

## 💡 Key Concepts

### Platform Selection

**NOT in code:**
```cpp
// ❌ WRONG - Don't do this!
#ifdef SDL2
    SDL2Engine engine;
#elif WEBGL
    WebGLEngine engine;
#endif
```

**NOT in includes:**
```cpp
// ❌ WRONG - Don't do this!
#include "SDL2Engine.hpp"  // Platform specific!
```

**YES - Use config.hpp:**
```cpp
// ✅ CORRECT - Abstract!
#include "imge/config.hpp"

imge::EngineImpl engine;  // Selected by CMake!
```

### Build-Time Selection

Platform is selected **at build time** by CMake options:
- `-DBUILD_DESKTOP=ON` → Links `SDL2Engine`
- `-DBUILD_WEB=ON` → Links `WebGLEngine`

Your game code stays **exactly the same**!

## 📄 File Sizes

```
IMGE Engine Libraries:
├── libimge_core.a      ~5 MB  (Core engine)
├── libimge_sdl2.a      ~0.6 MB (SDL2 implementation)
└── libimge_webgl.a     ~0.5 MB (WebAssembly impl)

Your Game (example):
├── Desktop binary        ~2 MB
└── Web (.html + .wasm)  ~1.5 MB
```

## 🎯 Best Practices

1. **Always copy template to a NEW directory**
   - Don't build inside `imge/`
   - Keeps engine and game separate

2. **Use abstract interfaces only**
   - `Input::getInstance()`, not `SDL2Input::getInstance()`
   - `Screen::getInstance()`, not `SDL2Renderer::getInstance()`

3. **Platform selection in CMake**
   - Use CMake options, not `#ifdef`
   - Keep game code clean

4. **Build once, target many**
   - Same game code, different build commands
   - Desktop or web, your choice!

## 🤝 Contributing

IMGE is designed to be:
- **Minimal:** Small codebase
- **Agnostic:** Platform-independent core
- **Clear:** Separation of engine and game

## 📄 License

[Your License Here]

---

**Remember:**
1. IMGE = Engine library (this repo)
2. Your Game = Separate project
3. Same code → Multiple platforms via build selection

**One codebase, infinite platforms!** 🚀
