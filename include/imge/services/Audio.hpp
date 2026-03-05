#pragma once

#include <cstdint>
#include <string>

namespace imge {

/**
 * Audio service - abstract interface for audio playback
 * Platform-specific implementations (SDL2_mixer, OpenAL, etc.) inherit from this
 *
 * Uses pointer-based singleton pattern to allow abstract base class
 */
class Audio {
public:
    virtual ~Audio() = default;

    /**
     * Get the singleton instance
     * @return Pointer to the audio service instance or nullptr if not set
     */
    static Audio* getInstance() {
        return instance;
    }

    /**
     * Set the singleton instance (called by concrete implementation)
     * @param inst Pointer to the concrete implementation
     */
    static void setInstance(Audio* inst) {
        instance = inst;
    }

    /**
     * Initialize the audio system
     */
    virtual void init() = 0;

    /**
     * Play background music
     * @param filename Path to music file
     * @param loop Should the music loop?
     * @param fadeIn Fade-in duration (seconds)
     * @param volume Volume (0.0 to 1.0)
     */
    virtual void playMusic(const std::string& filename,
                          bool loop = true,
                          float fadeIn = 0.0f,
                          float volume = 1.0f) = 0;

    /**
     * Stop the background music
     * @param fadeOut Fade-out duration (seconds)
     */
    virtual void stopMusic(float fadeOut = 0.0f) = 0;

    /**
     * Pause the background music
     */
    virtual void pauseMusic() = 0;

    /**
     * Resume the background music
     */
    virtual void resumeMusic() = 0;

    /**
     * Check if music is currently playing
     */
    [[nodiscard]] virtual bool isMusicPlaying() const = 0;

    /**
     * Set music volume
     * @param volume Volume (0.0 to 1.0)
     */
    virtual void setMusicVolume(float volume) = 0;

    /**
     * Play a sound effect
     * @param filename Path to sound file
     * @param volume Volume (0.0 to 1.0)
     * @param loop Should the sound loop?
     */
    virtual void playSound(const std::string& filename,
                          float volume = 1.0f,
                          bool loop = false) = 0;

    /**
     * Stop a specific sound
     * @param filename Path to sound file
     */
    virtual void stopSound(const std::string& filename) = 0;

    /**
     * Set sound effects volume
     * @param volume Volume (0.0 to 1.0)
     */
    virtual void setSoundVolume(float volume) = 0;

protected:
    static Audio* instance;
};

} // namespace imge
