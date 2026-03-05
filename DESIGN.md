# IMGE - Minimal Game Engine Design Document

## Executive Summary

**IMGE** is a lightweight, performant 2D game engine written in modern C++ (C++20/23). The design is inspired by the pygaminal framework, adapted for C++ best practices.

1. **Core Engine** - Platform-agnostic engine logic (this phase)
2. **Implementations** - Platform-specific code (SDL2, WebAssembly, etc.)

### Key Design Decisions

| Aspect | Decision | Notes |
|--------|----------|-------|
| Coordinate System | Top-Left (0,0), Y+ down | Screen space standard |
| Depth System | Float-based, **higher depth = front** | Reverse of pygaminal (user request) |
| Component Lifecycle | onCreate, onUpdate, onDraw (with Object* param) | Same as pygaminal |
| Update Order | All updates → All draws | Deferred pattern |
| Scripting | C++ components | Shared libraries (.so/.dll) |
| JSON Serialization | Manual fromJSON/toJSON | Explicit control |
| Object Structure | Flat, NO parent-child hierarchy | Simplified |
| Object Storage | std::unordered_map (O(1) lookup) | Same as pygaminal |
| Tag System | Tag mapping for O(1) filtered queries | Same as pygaminal |
| Template System | .obj file format | Prefab support |
| Test Implementation | SDL2 | Cross-platform |
| Component Naming | No "Component" suffix | Image, not ImageComponent |

### Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Game Data                             │
│         (JSON Files + C++ Shared Libraries)              │
└─────────────────────────────────────────────────────────┘
                            │
┌─────────────────────────────────────────────────────────┐
│              IMGE Core (Platform-Agnostic)               │
│  • Object/Component System                               │
│  • Scene Management (O(1) lookups)                       │
│  • Tag System (deferred updates)                         │
│  • Template/Prefab System (.obj files)                   │
│  • Singletons (Engine, Time, Screen, Input, Audio)      │
└─────────────────────────────────────────────────────────┘
                            │
┌─────────────────────────────────────────────────────────┐
│                Implementation (SDL2)                     │
│  • Rendering, Input, Main Loop                          │
└─────────────────────────────────────────────────────────┘
```

---

## Core Components

### 1. Object System (inspired by pygaminal)

Lightweight, flat structure - **NO parent-child hierarchy**.

```cpp
class Object {
public:
    std::string name;
    float x, y;        // Position
    float depth;       // Rendering depth (higher = front)
    std::unordered_set<std::string> tags;
    std::unordered_map<std::string, std::shared_ptr<Component>> components;
    bool dead = false;

    // Deferred tag updates (applied at end of frame)
    std::unordered_set<std::string> _pending_tag_adds;
    std::unordered_set<std::string> _pending_tag_removes;

    // Static counter for auto-generated names
    static inline size_t _id_counter = 0;
};
```

### 2. Component System

```cpp
class Component {
public:
    virtual ~Component() = default;

    // Lifecycle methods (receive Object* as parameter)
    virtual void onCreate(Object* owner) {}
    virtual void onUpdate(Object* owner) {}
    virtual void onDraw(Object* owner) {}

    // JSON serialization (manual implementation per component)
    virtual void fromJSON(const nlohmann::json& j) {}
    virtual void toJSON(nlohmann::json& j) const {}
};
```

**Component Naming Convention (from pygaminal):**
- **No "Component" suffix**
- `Image` not `ImageComponent`
- Builtin: `@Image`, `@Animation` prefix
- Custom: Just the class name

### 3. Scene Management (inspired by pygaminal)

```cpp
class Scene {
public:
    // Dictionary for O(1) object lookup by name
    std::unordered_map<std::string, std::shared_ptr<Object>> objects;

    // Tag mapping for O(1) tag queries
    std::unordered_map<std::string, std::vector<std::string>> _tags;

    // Pending objects to add at end of frame
    std::vector<std::shared_ptr<Object>> _pending_objects;

    // Scene properties
    int width = 800;
    int height = 600;
    std::optional<std::string> backgroundColor;
    std::optional<std::string> backgroundImage;
};
```

**Deferred Update Pattern (from pygaminal):**
All mutations (add/remove objects, add/remove tags) are deferred to end of frame:
1. Update all objects
2. Call `_applyPendingUpdates()`:
   - Remove dead objects
   - Add pending objects
   - Apply pending tag changes
3. Draw all objects

**Depth Sorting (modified from pygaminal):**
- Pygaminal: lower depth = drawn first (background → foreground)
- IMGE: **higher depth = drawn first (foreground → background)**

### 4. Tag System

Optimized tag-based object filtering:

```cpp
// Tag mapping structure
tagMapping = {
    "player":     {"obj_player", "obj_player_shadow"},
    "enemy":      {"obj_enemy_1", "obj_enemy_2", "obj_boss"},
    "collectible": {"obj_coin_1", "obj_gem_1", "obj_key"}
}
```

### 5. Template/Prefab System (.obj files from pygaminal)

Objects can be saved as `.obj` files and instantiated in scenes:

**enemy.obj:**
```json
{
    "name": "enemy_basic",
    "tags": ["enemy", "flying"],
    "depth": 10,
    "components": [
        {"file": "@Hitbox", "args": [[-16, -16, 32, 32]]},
        {"file": "@Image", "args": ["enemy.png", "center", "center"]},
        {"file": "EnemyAI", "args": [100]}
    ]
}
```

**scene_data.json:**
```json
{
    "width": 800,
    "height": 600,
    "background_color": "#222222",
    "objects": [
        {"file": "objects/enemy.obj", "x": 400, "y": 300},
        {"file": "objects/enemy.obj", "x": 600, "y": 200}
    ]
}
```

---

## Project Structure

```
imge/
├── include/imge/
│   ├── core/
│   │   ├── Component.hpp
│   │   ├── Object.hpp
│   │   ├── Scene.hpp
│   │   └── Engine.hpp
│   ├── services/
│   │   ├── Time.hpp
│   │   ├── Screen.hpp
│   │   ├── Input.hpp
│   │   └── Audio.hpp
│   ├── components/
│   │   ├── Image.hpp
│   │   ├── Animation.hpp
│   │   └── Hitbox.hpp
│   └── impl/
│       ├── SDL2Renderer.hpp
│       ├── SDL2Input.hpp
│       └── SDL2Audio.hpp
├── src/
│   ├── core/
│   ├── components/
│   └── impl/sdl2/
├── tests/
├── examples/
├── CMakeLists.txt
└── README.md
```

---

## Implementation Roadmap

### Priority 1: Core Foundation
- [ ] Setup CMake project structure
- [ ] Create Vec2, Rect basic types
- [ ] Implement Singleton template
- [ ] Component base class with lifecycle methods
- [ ] Object class with x, y, depth, tags, components
- [ ] Component-to-Object communication
- [ ] Tag system with mapping

### Priority 2: Scene System
- [ ] Scene class with object dictionary
- [ ] Tag mapping in Scene (auto-update)
- [ ] Scene JSON save/load
- [ ] Template system (prefabs)
- [ ] Scene transition support

### Priority 3: Engine Core
- [ ] Engine singleton (abstract)
- [ ] Time singleton
- [ ] Abstract Screen, Input, Audio interfaces

### Priority 4: SDL2 Implementation
- [ ] SDL2Renderer
- [ ] SDL2Input
- [ ] Main loop with SDL2
- [ ] SpriteRenderer component
- [ ] AABB Collider component

### Priority 5: Testing
- [ ] Unit tests for all core systems
- [ ] Integration tests
- [ ] Example game
