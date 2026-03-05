#include "imge/impl/SDL2Audio.hpp"

#include <algorithm>
#include <map>
#include <stdexcept>

namespace imge {

SDL2Audio::SDL2Audio() {
    setInstance(this);
}

SDL2Audio::~SDL2Audio() {
    if (currentMusic) {
        Mix_FreeMusic(currentMusic);
        currentMusic = nullptr;
    }
}

void SDL2Audio::init() {
    // Initialize SDL2_mixer
    if (Mix_OpenAudio(44100, MIX_DEFAULT_FORMAT, 2, 2048) < 0) {
        throw std::runtime_error("Failed to initialize SDL2_mixer: " + std::string(Mix_GetError()));
    }

    // Allocate channels for sound effects
    Mix_AllocateChannels(16);
}

void SDL2Audio::playMusic(const std::string& filename,
                          bool loop,
                          float fadeIn,
                          float volume) {
    // Load music
    currentMusic = Mix_LoadMUS(filename.c_str());
    if (!currentMusic) {
        return; // Failed to load
    }

    // Set volume
    Mix_VolumeMusic(static_cast<int>(musicVolume * MIX_MAX_VOLUME));

    // Play music
    int loops = loop ? -1 : 0; // -1 = infinite loop
    if (fadeIn > 0.0f) {
        Mix_FadeInMusic(currentMusic, loops, static_cast<int>(fadeIn * 1000));
    } else {
        Mix_PlayMusic(currentMusic, loops);
    }
}

void SDL2Audio::stopMusic(float fadeOut) {
    if (fadeOut > 0.0f) {
        Mix_FadeOutMusic(static_cast<int>(fadeOut * 1000));
    } else {
        Mix_HaltMusic();
    }
}

void SDL2Audio::pauseMusic() {
    Mix_PauseMusic();
}

void SDL2Audio::resumeMusic() {
    Mix_ResumeMusic();
}

bool SDL2Audio::isMusicPlaying() const {
    return Mix_PlayingMusic();
}

void SDL2Audio::setMusicVolume(float volume) {
    musicVolume = std::clamp(volume, 0.0f, 1.0f);
    Mix_VolumeMusic(static_cast<int>(musicVolume * MIX_MAX_VOLUME));
}

void SDL2Audio::playSound(const std::string& filename,
                          float volume,
                          bool loop) {
    // Load sound
    Mix_Chunk* chunk = Mix_LoadWAV(filename.c_str());
    if (!chunk) {
        return; // Failed to load
    }

    // Set volume
    Mix_VolumeChunk(chunk, static_cast<int>(soundVolume * MIX_MAX_VOLUME));

    // Play sound
    int loops = loop ? -1 : 0;
    Mix_PlayChannel(-1, chunk, loops);

    // Note: This is a simplified version that doesn't track sound instances
    // A full implementation would track channels and clean up chunks
}

void SDL2Audio::stopSound(const std::string& filename) {
    // TODO: Implement sound stopping by tracking channels
    (void)filename;
}

void SDL2Audio::setSoundVolume(float volume) {
    soundVolume = std::clamp(volume, 0.0f, 1.0f);
    // Note: This affects newly played sounds, not currently playing ones
    // A full implementation would update all currently playing sounds
}

} // namespace imge
