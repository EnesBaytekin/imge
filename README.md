# IMGE - Minimal Game Engine

**IMGE** is a lightweight, performant 2D game engine written in modern C++ (C++20/23). The design is inspired by the pygaminal framework, adapted for C++ best practices.

## Features

- **Platform-Agnostic Core**: Core engine logic is completely independent of platform-specific code
- **Object-Component System**: Flat hierarchy, NO parent-child relationships
- **O(1) Lookups**: Dictionary storage for objects and tags
- **Deferred Updates**: All mutations applied at end of frame for consistency
- **Template System**: .obj prefab files for reusable objects
- **Component Naming**: No "Component" suffix (Image, not ImageComponent)
- **Data-Driven**: Games defined by JSON + C++ components
- **Modern C++**: C++20/23 with smart pointers and RAII

## Key Design Decisions

| Feature | Decision |
|---------|----------|
| Coordinate System | Top-Left (0,0), Y+ down |
| Depth System | Higher depth = rendered in front |
| Component Lifecycle | onCreate, onUpdate, onDraw (with Object* param) |
| Update Order | All updates → All draws |
| Object Storage | std::unordered_map (O(1) lookup) |
| Tag System | Tag mapping for O(1) filtered queries |

## Project Structure

```
imge/
├── include/imge/
│   ├── core/
│   │   ├── Vec2.hpp           // 2D vector class
│   │   ├── Rect.hpp           // Rectangle class
│   │   ├── Singleton.hpp      // Singleton template
│   │   ├── Component.hpp      // Base component class
│   │   ├── Object.hpp         // Entity with components
│   │   ├── Scene.hpp          // Scene management
│   │   └── Engine.hpp         // Main engine singleton
│   ├── services/
│   │   ├── Time.hpp          // Delta time service
│   │   ├── Screen.hpp        // Abstract render interface
│   │   ├── Input.hpp         // Abstract input interface
│   │   └── Audio.hpp         // Abstract audio interface
│   ├── components/
│   │   ├── Image.hpp         // Builtin: Sprite rendering
│   │   ├── Animation.hpp     // Builtin: Sprite sheet animation
│   │   └── Hitbox.hpp        // Builtin: Collision boxes
│   └── impl/
│       ├── SDL2Renderer.hpp  // SDL2 implementation
│       ├── SDL2Input.hpp     // SDL2 input
│       └── SDL2Audio.hpp     // SDL2 audio
├── src/
│   ├── core/                 // Core implementations
│   ├── components/           // Builtin component implementations
│   └── impl/sdl2/            // SDL2 implementations
├── tests/                    // Unit tests
├── examples/                 // Example games
├── DESIGN.md                 // Detailed design document
└── CMakeLists.txt
```

## Building

### Dependencies

**Required:**
- CMake 3.20+
- C++20 compatible compiler (GCC 11+, Clang 13+, MSVC 2019+)
- nlohmann_json 3.10.0+

**For SDL2 Implementation (recommended):**
- SDL2
- SDL2_image
- SDL2_mixer

### Installing Dependencies

**Fedora/RHEL/CentOS:**
```bash
sudo dnf install cmake gcc-c++ ninja-build
sudo dnf install SDL2-devel SDL2_image-devel SDL2_mixer-devel
sudo dnf install json-devel libpng-devel
```

**Ubuntu/Debian:**
```bash
sudo apt install cmake build-essential ninja-build
sudo apt install libsdl2-dev libsdl2-image-dev libsdl2-mixer-dev
sudo apt install nlohmann-json3-dev
```

**macOS (with Homebrew):**
```bash
brew install cmake ninja
brew install sdl2 sdl2_image sdl2_mixer
brew install nlohmann-json
```

**Windows (with vcpkg):**
```cmd
vcpkg install sdl2 sdl2-image[libpng] sdl2-mixer nlohmann-json
```

### Build Instructions

```bash
# Clone the repository
cd imge

# Create build directory
mkdir build && cd build

# Configure with CMake (Release build)
cmake .. -DCMAKE_BUILD_TYPE=Release

# Or configure with Ninja (faster for large projects)
cmake .. -GNinja -DCMAKE_BUILD_TYPE=Release

# Build
cmake --build . -j$(nproc)  # Linux/macOS
cmake --build . -j%NUMBER_OF_PROCESSORS%  # Windows

# Run tests
ctest --output-on-failure

# Or run tests individually
./tests/test_object
./tests/test_scene

# Run example game
cd examples
./simple_game
# Controls: WASD to move, ESC to quit
```

### Build Options

| Option | Default | Description |
|--------|---------|-------------|
| `IMGE_WITH_SDL2` | `ON` | Build SDL2 implementation |
| `IMGE_BUILD_TESTS` | `ON` | Build unit tests |
| `IMGE_BUILD_EXAMPLES` | `ON` | Build example games |

**Example:**
```bash
# Build without SDL2 implementation
cmake .. -DIMGE_WITH_SDL2=OFF

# Build without tests and examples
cmake .. -DIMGE_BUILD_TESTS=OFF -DIMGE_BUILD_EXAMPLES=OFF
```

## Usage

### Quick Start

The simplest way to create a game with IMGE:

```cpp
#include "imge/core/Engine.hpp"
#include "imge/impl/SDL2Renderer.hpp"
#include "imge/impl/SDL2Input.hpp"
#include "imge/impl/SDL2Audio.hpp"

// 1. Create a custom component
class PlayerController : public imge::Component {
public:
    float speed = 200.0f;

    void onUpdate(imge::Object* owner) override {
        auto* input = static_cast<imge::SDL2Input*>(imge::Input::getInstance());
        auto dt = imge::Time::getInstance()->deltaTime;

        float dx = 0.0f, dy = 0.0f;
        if (input->isKeyPressed(imge::Key::W)) dy -= 1.0f;
        if (input->isKeyPressed(imge::Key::S)) dy += 1.0f;
        if (input->isKeyPressed(imge::Key::A)) dx -= 1.0f;
        if (input->isKeyPressed(imge::Key::D)) dx += 1.0f;

        // Normalize diagonal movement
        if (dx != 0.0f && dy != 0.0f) {
            float length = std::sqrt(dx * dx + dy * dy);
            dx /= length;
            dy /= length;
        }

        owner->x += dx * speed * dt;
        owner->y += dy * speed * dt;
    }
};

// 2. Create a renderer component
class RectangleRenderer : public imge::Component {
public:
    uint8_t r = 255, g = 255, b = 255, a = 255;
    float width = 50.0f, height = 50.0f;

    RectangleRenderer(uint8_t r_, uint8_t g_, uint8_t b_, float w, float h)
        : r(r_), g(g_), b(b_), width(w), height(h) {}

    void onDraw(imge::Object* owner) override {
        auto* screen = static_cast<imge::SDL2Renderer*>(imge::Screen::getInstance());
        auto* renderer = screen->getRenderer();

        SDL_Rect rect{
            static_cast<int>(owner->x - width / 2),
            static_cast<int>(owner->y - height / 2),
            static_cast<int>(width),
            static_cast<int>(height)
        };

        SDL_SetRenderDrawColor(renderer, r, g, b, a);
        SDL_RenderFillRect(renderer, &rect);
    }
};

// 3. Setup and run
int main() {
    SDL_Init(SDL_INIT_VIDEO | SDL_INIT_AUDIO);

    // Create service instances
    imge::SDL2Renderer renderer;
    imge::SDL2Input input;
    imge::SDL2Audio audio;

    // Create engine
    imge::SDL2Engine engine;

    // Create scene
    auto scene = std::make_shared<imge::Scene>();
    scene->width = 800;
    scene->height = 600;
    scene->backgroundColor = "#222222";

    // Create player object
    auto player = std::make_shared<imge::Object>(400.0f, 300.0f, "player");

    // Add components
    player->addComponent(
        std::shared_ptr<RectangleRenderer>(new RectangleRenderer(50, 150, 255, 50, 50)),
        "renderer"
    );
    player->addComponent(
        std::shared_ptr<PlayerController>(new PlayerController()),
        "controller"
    );

    scene->addObject(player);
    engine.addScene("main", scene);

    // Initialize and run
    engine.init(800, 600, "My Game");
    engine.run();

    SDL_Quit();
    return 0;
}
```

### Creating a Custom Component

Components are the building blocks of game logic:

```cpp
class MyComponent : public imge::Component {
public:
    // Properties
    float value = 100.0f;

    // Called when component is created
    void onCreate(imge::Object* owner) override {
        // Initialization code
    }

    // Called every frame
    void onUpdate(imge::Object* owner) override {
        // Game logic
        auto* input = static_cast<imge::SDL2Input*>(imge::Input::getInstance());
        auto dt = imge::Time::getInstance()->deltaTime;
    }

    // Called every frame (after all updates)
    void onDraw(imge::Object* owner) override {
        // Rendering code
    }

    // JSON serialization for data-driven design
    void fromJSON(const nlohmann::json& j) override {
        value = j.value("value", 100.0f);
    }
};
```

### Creating a Scene

```cpp
// Create scene
auto scene = std::make_shared<imge::Scene>();
scene->width = 800;
scene->height = 600;
scene->backgroundColor = "#222222";

// Create object
auto player = std::make_shared<imge::Object>(400.0f, 300.0f, "player");
player->addTag("player");
player->depth = 10.0f;

// Add to scene
scene->addObject(player);

// Add to engine
auto* engine = imge::Engine::getInstance();
engine->addScene("main", scene);
```

## License

MIT License - see LICENSE file for details.

## Roadmap

- [ ] Complete SDL2 implementation
- [ ] Add more builtin components (@Movability, @YSort, @BackgroundMusic, @SoundEffect)
- [ ] Implement component loader (shared libraries)
- [ ] Add collision detection system
- [ ] Create visual editor
- [ ] WebAssembly support
- [ ] Mobile platform support
