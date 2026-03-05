#include "imge/impl/SDL2Input.hpp"

#include <algorithm>

namespace imge {

SDL2Input::SDL2Input() {
    setInstance(this);
}

void SDL2Input::update() {
    // Store previous state
    previousKeys = currentKeys;
    previousMouseButtons = currentMouseButtons;

    // Clear current state
    currentKeys.clear();
    currentMouseButtons.clear();

    // Reset mouse wheel
    mouseWheelX = 0.0f;
    mouseWheelY = 0.0f;

    // Get keyboard state
    const Uint8* state = SDL_GetKeyboardState(nullptr);

    // Map SDL key codes to our Key enum
    // This is a simplified version - full implementation would map all keys
    if (state[SDL_SCANCODE_A]) currentKeys.insert(Key::A);
    if (state[SDL_SCANCODE_B]) currentKeys.insert(Key::B);
    if (state[SDL_SCANCODE_C]) currentKeys.insert(Key::C);
    if (state[SDL_SCANCODE_D]) currentKeys.insert(Key::D);
    if (state[SDL_SCANCODE_E]) currentKeys.insert(Key::E);
    if (state[SDL_SCANCODE_F]) currentKeys.insert(Key::F);
    if (state[SDL_SCANCODE_G]) currentKeys.insert(Key::G);
    if (state[SDL_SCANCODE_H]) currentKeys.insert(Key::H);
    if (state[SDL_SCANCODE_I]) currentKeys.insert(Key::I);
    if (state[SDL_SCANCODE_J]) currentKeys.insert(Key::J);
    if (state[SDL_SCANCODE_K]) currentKeys.insert(Key::K);
    if (state[SDL_SCANCODE_L]) currentKeys.insert(Key::L);
    if (state[SDL_SCANCODE_M]) currentKeys.insert(Key::M);
    if (state[SDL_SCANCODE_N]) currentKeys.insert(Key::N);
    if (state[SDL_SCANCODE_O]) currentKeys.insert(Key::O);
    if (state[SDL_SCANCODE_P]) currentKeys.insert(Key::P);
    if (state[SDL_SCANCODE_Q]) currentKeys.insert(Key::Q);
    if (state[SDL_SCANCODE_R]) currentKeys.insert(Key::R);
    if (state[SDL_SCANCODE_S]) currentKeys.insert(Key::S);
    if (state[SDL_SCANCODE_T]) currentKeys.insert(Key::T);
    if (state[SDL_SCANCODE_U]) currentKeys.insert(Key::U);
    if (state[SDL_SCANCODE_V]) currentKeys.insert(Key::V);
    if (state[SDL_SCANCODE_W]) currentKeys.insert(Key::W);
    if (state[SDL_SCANCODE_X]) currentKeys.insert(Key::X);
    if (state[SDL_SCANCODE_Y]) currentKeys.insert(Key::Y);
    if (state[SDL_SCANCODE_Z]) currentKeys.insert(Key::Z);

    if (state[SDL_SCANCODE_0]) currentKeys.insert(Key::Num0);
    if (state[SDL_SCANCODE_1]) currentKeys.insert(Key::Num1);
    if (state[SDL_SCANCODE_2]) currentKeys.insert(Key::Num2);
    if (state[SDL_SCANCODE_3]) currentKeys.insert(Key::Num3);
    if (state[SDL_SCANCODE_4]) currentKeys.insert(Key::Num4);
    if (state[SDL_SCANCODE_5]) currentKeys.insert(Key::Num5);
    if (state[SDL_SCANCODE_6]) currentKeys.insert(Key::Num6);
    if (state[SDL_SCANCODE_7]) currentKeys.insert(Key::Num7);
    if (state[SDL_SCANCODE_8]) currentKeys.insert(Key::Num8);
    if (state[SDL_SCANCODE_9]) currentKeys.insert(Key::Num9);

    if (state[SDL_SCANCODE_SPACE]) currentKeys.insert(Key::Space);
    if (state[SDL_SCANCODE_RETURN]) currentKeys.insert(Key::Enter);
    if (state[SDL_SCANCODE_ESCAPE]) currentKeys.insert(Key::Escape);
    if (state[SDL_SCANCODE_BACKSPACE]) currentKeys.insert(Key::Backspace);
    if (state[SDL_SCANCODE_TAB]) currentKeys.insert(Key::Tab);

    if (state[SDL_SCANCODE_LEFT]) currentKeys.insert(Key::Left);
    if (state[SDL_SCANCODE_RIGHT]) currentKeys.insert(Key::Right);
    if (state[SDL_SCANCODE_UP]) currentKeys.insert(Key::Up);
    if (state[SDL_SCANCODE_DOWN]) currentKeys.insert(Key::Down);

    if (state[SDL_SCANCODE_LSHIFT]) currentKeys.insert(Key::LeftShift);
    if (state[SDL_SCANCODE_RSHIFT]) currentKeys.insert(Key::RightShift);
    if (state[SDL_SCANCODE_LCTRL]) currentKeys.insert(Key::LeftCtrl);
    if (state[SDL_SCANCODE_RCTRL]) currentKeys.insert(Key::RightCtrl);

    // Get mouse state
    Uint32 mouseState = SDL_GetMouseState(&mouseX, &mouseY);

    if (mouseState & SDL_BUTTON_LMASK) currentMouseButtons.insert(MouseButton::Left);
    if (mouseState & SDL_BUTTON_MMASK) currentMouseButtons.insert(MouseButton::Middle);
    if (mouseState & SDL_BUTTON_RMASK) currentMouseButtons.insert(MouseButton::Right);
}

bool SDL2Input::isKeyPressed(Key key) const {
    return currentKeys.find(key) != currentKeys.end();
}

bool SDL2Input::isKeyJustPressed(Key key) const {
    return currentKeys.find(key) != currentKeys.end() &&
           previousKeys.find(key) == previousKeys.end();
}

bool SDL2Input::isKeyJustReleased(Key key) const {
    return currentKeys.find(key) == currentKeys.end() &&
           previousKeys.find(key) != previousKeys.end();
}

std::pair<int, int> SDL2Input::getMousePosition() const {
    return {mouseX, mouseY};
}

bool SDL2Input::isMouseButtonPressed(MouseButton button) const {
    return currentMouseButtons.find(button) != currentMouseButtons.end();
}

bool SDL2Input::isMouseButtonJustPressed(MouseButton button) const {
    return currentMouseButtons.find(button) != currentMouseButtons.end() &&
           previousMouseButtons.find(button) == previousMouseButtons.end();
}

bool SDL2Input::isMouseButtonJustReleased(MouseButton button) const {
    return currentMouseButtons.find(button) == currentMouseButtons.end() &&
           previousMouseButtons.find(button) != previousMouseButtons.end();
}

std::pair<float, float> SDL2Input::getMouseWheel() const {
    return {mouseWheelX, mouseWheelY};
}

Key SDL2Input::SDLKeyToKey(SDL_Keycode sdlKey) {
    // Map SDL key codes to our Key enum
    switch (sdlKey) {
        case SDLK_a: return Key::A;
        case SDLK_b: return Key::B;
        case SDLK_c: return Key::C;
        case SDLK_d: return Key::D;
        case SDLK_e: return Key::E;
        case SDLK_f: return Key::F;
        case SDLK_g: return Key::G;
        case SDLK_h: return Key::H;
        case SDLK_i: return Key::I;
        case SDLK_j: return Key::J;
        case SDLK_k: return Key::K;
        case SDLK_l: return Key::L;
        case SDLK_m: return Key::M;
        case SDLK_n: return Key::N;
        case SDLK_o: return Key::O;
        case SDLK_p: return Key::P;
        case SDLK_q: return Key::Q;
        case SDLK_r: return Key::R;
        case SDLK_s: return Key::S;
        case SDLK_t: return Key::T;
        case SDLK_u: return Key::U;
        case SDLK_v: return Key::V;
        case SDLK_w: return Key::W;
        case SDLK_x: return Key::X;
        case SDLK_y: return Key::Y;
        case SDLK_z: return Key::Z;

        case SDLK_0: return Key::Num0;
        case SDLK_1: return Key::Num1;
        case SDLK_2: return Key::Num2;
        case SDLK_3: return Key::Num3;
        case SDLK_4: return Key::Num4;
        case SDLK_5: return Key::Num5;
        case SDLK_6: return Key::Num6;
        case SDLK_7: return Key::Num7;
        case SDLK_8: return Key::Num8;
        case SDLK_9: return Key::Num9;

        case SDLK_SPACE: return Key::Space;
        case SDLK_RETURN: return Key::Enter;
        case SDLK_ESCAPE: return Key::Escape;
        case SDLK_BACKSPACE: return Key::Backspace;
        case SDLK_TAB: return Key::Tab;

        case SDLK_LEFT: return Key::Left;
        case SDLK_RIGHT: return Key::Right;
        case SDLK_UP: return Key::Up;
        case SDLK_DOWN: return Key::Down;

        default: return Key::A; // Fallback
    }
}

} // namespace imge
