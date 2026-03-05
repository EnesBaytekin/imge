#pragma once

#include "imge/services/Audio.hpp"

#include <SDL2/SDL_mixer.h>
#include <map>
#include <string>

namespace imge {

/**
 * SDL2_mixer implementation of Audio service
 */
class SDL2Audio : public Audio {
public:
    SDL2Audio();
    ~SDL2Audio() override;

    void init() override;
    void playMusic(const std::string& filename,
                   bool loop = true,
                   float fadeIn = 0.0f,
                   float volume = 1.0f) override;
    void stopMusic(float fadeOut = 0.0f) override;
    void pauseMusic() override;
    void resumeMusic() override;
    [[nodiscard]] bool isMusicPlaying() const override;
    void setMusicVolume(float volume) override;
    void playSound(const std::string& filename,
                   float volume = 1.0f,
                   bool loop = false) override;
    void stopSound(const std::string& filename) override;
    void setSoundVolume(float volume) override;

private:
    Mix_Music* currentMusic = nullptr;
    float musicVolume = 1.0f;
    float soundVolume = 1.0f;
};

} // namespace imge
