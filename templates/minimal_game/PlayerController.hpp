#pragma once

#include "imge/core/Component.hpp"
#include "imge/core/Object.hpp"
#include "imge/services/Input.hpp"
#include "imge/services/Time.hpp"

#include <cmath>

/**
 * Minimal Player Controller
 * WASD movement - Platform agnostic!
 */
class PlayerController : public imge::Component {
public:
    float speed = 200.0f;

    void onUpdate(imge::Object* owner) override {
        auto* input = imge::Input::getInstance();
        auto dt = imge::Time::getInstance()->deltaTime;

        float dx = 0.0f;
        float dy = 0.0f;

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

        // ESC to quit
        if (input->isKeyPressed(imge::Key::Escape)) {
            imge::Engine::getInstance()->stop();
        }
    }
};
