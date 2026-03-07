#include "imge/impl/EmscriptenAudio.hpp"
#include <iostream>

namespace imge {

EmscriptenAudio::EmscriptenAudio() {
    setInstance(this);
}

EmscriptenAudio::~EmscriptenAudio() = default;

void EmscriptenAudio::init() {
    initialized = true;
    std::cout << "[Audio] Emscripten audio (stub) initialized" << std::endl;
}

void EmscriptenAudio::playMusic(const std::string& filename,
                                bool loop,
                                float fadeIn,
                                float volume) {
    (void)filename;
    (void)loop;
    (void)fadeIn;
    (void)volume;
    if (!initialized) init();
    musicPlaying = true;
}

void EmscriptenAudio::stopMusic(float fadeOut) {
    (void)fadeOut;
    musicPlaying = false;
}

void EmscriptenAudio::pauseMusic() {}

void EmscriptenAudio::resumeMusic() {}

bool EmscriptenAudio::isMusicPlaying() const {
    return musicPlaying;
}

void EmscriptenAudio::setMusicVolume(float volume) {
    musicVolume = volume;
}

void EmscriptenAudio::playSound(const std::string& filename,
                                float volume,
                                bool loop) {
    (void)filename;
    (void)volume;
    (void)loop;
    if (!initialized) init();
}

void EmscriptenAudio::stopSound(const std::string& filename) {
    (void)filename;
}

void EmscriptenAudio::setSoundVolume(float volume) {
    soundVolume = volume;
}

} // namespace imge
