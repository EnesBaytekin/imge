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
├── docs/
│   └── DESIGN.md             // Detailed design document
└── CMakeLists.txt
```

## Building

### Dependencies

- CMake 3.20+
- C++20 compatible compiler
- nlohmann_json 3.10.0+
- SDL2 (for SDL2 implementation)
- SDL2_image
- SDL2_mixer

### Build Instructions

```bash
# Clone the repository
cd imge

# Create build directory
mkdir build && cd build

# Configure with CMake
cmake ..

# Build
cmake --build .

# Run tests
ctest --output-on-failure

# Run example game
cd examples
./simple_game
```

## Usage

### Creating a Custom Component

```cpp
#include "imge/core/Component.hpp"
#include "imge/core/Object.hpp"
#include "imge/services/Input.hpp"
#include "imge/services/Time.hpp"

using namespace imge;

class PlayerController : public Component {
public:
    float speed = 200.0f;

    void fromJSON(const nlohmann::json& j) override {
        speed = j.value("speed", 200.0f);
    }

    void onUpdate(Object* owner) override {
        auto* input = static_cast<SDL2Input*>(Input::getInstance());
        auto dt = Time::getInstance()->deltaTime;

        float dx = input->isKeyPressed(Key::D) - input->isKeyPressed(Key::A);
        float dy = input->isKeyPressed(Key::S) - input->isKeyPressed(Key::W);

        owner->x += dx * speed * dt;
        owner->y += dy * speed * dt;
    }
};
```

### Creating a Scene

```cpp
auto scene = std::make_shared<Scene>();
scene->width = 800;
scene->height = 600;
scene->backgroundColor = "#222222";

auto player = std::make_shared<Object>(400.0f, 300.0f, "player");
player->addTag("player");
player->depth = 10.0f;
scene->addObject(player);

auto* engine = Engine::getInstance();
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
