#pragma once

#include "imge/services/Audio.hpp"
#include <string>

namespace imge {

/**
 * Emscripten stub implementation of Audio service
 * Currently a no-op placeholder for future Web Audio API implementation
 */
class EmscriptenAudio : public Audio {
public:
    EmscriptenAudio();
    ~EmscriptenAudio() override;

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
    bool initialized = false;
    bool musicPlaying = false;
    float musicVolume = 1.0f;
    float soundVolume = 1.0f;
};

} // namespace imge
