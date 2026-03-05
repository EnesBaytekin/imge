#pragma once

#include "imge/services/Input.hpp"

#include <SDL2/SDL.h>
#include <map>
#include <unordered_set>

namespace imge {

/**
 * SDL2 implementation of Input service
 */
class SDL2Input : public Input {
public:
    SDL2Input() = default;
    ~SDL2Input() override = default;

    void update() override;
    [[nodiscard]] bool isKeyPressed(Key key) const override;
    [[nodiscard]] bool isKeyJustPressed(Key key) const override;
    [[nodiscard]] bool isKeyJustReleased(Key key) const override;
    [[nodiscard]] std::pair<int, int> getMousePosition() const override;
    [[nodiscard]] bool isMouseButtonPressed(MouseButton button) const override;
    [[nodiscard]] bool isMouseButtonJustPressed(MouseButton button) const override;
    [[nodiscard]] bool isMouseButtonJustReleased(MouseButton button) const override;
    [[nodiscard]] std::pair<float, float> getMouseWheel() const override;

    /**
     * Convert SDL key code to our Key enum
     */
    static Key SDLKeyToKey(SDL_Keycode sdlKey);

private:
    std::unordered_set<Key> currentKeys;
    std::unordered_set<Key> previousKeys;
    std::unordered_set<MouseButton> currentMouseButtons;
    std::unordered_set<MouseButton> previousMouseButtons;
    int mouseX = 0;
    int mouseY = 0;
    float mouseWheelX = 0.0f;
    float mouseWheelY = 0.0f;
};

} // namespace imge
