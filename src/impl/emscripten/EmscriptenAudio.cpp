#include "imge/impl/EmscriptenAudio.hpp"
#include <emscripten.h>
#include <iostream>

namespace imge {

EmscriptenAudio::EmscriptenAudio() {
    setInstance(this);
}

EmscriptenAudio::~EmscriptenAudio() = default;

void EmscriptenAudio::init() {
    if (initialized) return;

    // Initialize AudioContext via JavaScript
    EM_ASM({
        if (!window.imgeAudioContext) {
            window.imgeAudioContext = new (window.AudioContext || window.webkitAudioContext)();
            console.log('[Audio] AudioContext created');
        }

        // Resume audio context if suspended (browser autoplay policy)
        if (window.imgeAudioContext.state === 'suspended') {
            window.imgeAudioContext.resume();
        }

        // Store loaded sounds
        if (!window.imgeSounds) {
            window.imgeSounds = {};
        }

        // Store active sound sources for stopping
        if (!window.imgeSoundSources) {
            window.imgeSoundSources = {};
        }
    });

    initialized = true;
    std::cout << "[Audio] Emscripten audio initialized" << std::endl;
}

void EmscriptenAudio::playMusic(const std::string& filename,
                                bool loop,
                                float fadeIn,
                                float volume) {
    (void)fadeIn; // TODO: Implement fade in
    if (!initialized) init();

    EM_ASM_({
        var filename = UTF8ToString($0);
        var loop = $1;
        var volume = $2;

        var ctx = window.imgeAudioContext;
        if (!ctx) return;

        // Stop existing music if playing
        if (window.imgeMusicSource) {
            try {
                window.imgeMusicSource.stop();
            } catch(e) {}
        }

        // Check if already loaded
        if (window.imgeSounds[filename]) {
            _playMusicBuffer(ctx, window.imgeSounds[filename], loop, volume);
        } else {
            // Read file from Emscripten FS and decode
            try {
                var data = FS.readFile(filename);
                var blob = new Blob([data], { type: 'audio/mpeg' });

                var reader = new FileReader();
                reader.onload = function(e) {
                    ctx.decodeAudioData(e.target.result, function(buffer) {
                        window.imgeSounds[filename] = buffer;
                        _playMusicBuffer(ctx, buffer, loop, volume);
                        console.log('[Audio] Music loaded and playing:', filename);
                    }, function(err) {
                        console.error('[Audio] Failed to decode audio:', err);
                    });
                };
                reader.readAsArrayBuffer(blob);
            } catch(e) {
                console.error('[Audio] Failed to read music file:', filename, e);
            }
        }
    }, filename.c_str(), loop, volume);

    musicPlaying = true;
}

void EmscriptenAudio::stopMusic(float fadeOut) {
    (void)fadeOut; // TODO: Implement fade out

    EM_ASM({
        if (window.imgeMusicSource) {
            try {
                window.imgeMusicSource.stop();
                window.imgeMusicSource = null;
            } catch(e) {}
        }
    });

    musicPlaying = false;
}

void EmscriptenAudio::pauseMusic() {
    EM_ASM({
        if (window.imgeMusicSource) {
            try {
                window.imgeMusicSource.stop();
            } catch(e) {}
        }
    });
    musicPlaying = false;
}

void EmscriptenAudio::resumeMusic() {
    // For simplicity, just restart the music
    // In a full implementation, we'd track pause position
    musicPlaying = true;
}

bool EmscriptenAudio::isMusicPlaying() const {
    return musicPlaying;
}

void EmscriptenAudio::setMusicVolume(float volume) {
    musicVolume = volume;

    EM_ASM_({
        var volume = $0;
        if (window.imgeMusicGain) {
            window.imgeMusicGain.gain.value = volume;
        }
    }, volume);
}

void EmscriptenAudio::playSound(const std::string& filename,
                                float volume,
                                bool loop) {
    if (!initialized) init();

    EM_ASM_({
        var filename = UTF8ToString($0);
        var volume = $1;
        var loop = $2;

        var ctx = window.imgeAudioContext;
        if (!ctx) return;

        // Check if already loaded
        if (window.imgeSounds[filename]) {
            _playSoundBuffer(ctx, window.imgeSounds[filename], filename, volume, loop);
        } else {
            // Read file from Emscripten FS and decode
            try {
                var data = FS.readFile(filename);
                var blob = new Blob([data], { type: 'audio/mpeg' });

                var reader = new FileReader();
                reader.onload = function(e) {
                    ctx.decodeAudioData(e.target.result, function(buffer) {
                        window.imgeSounds[filename] = buffer;
                        _playSoundBuffer(ctx, buffer, filename, volume, loop);
                        console.log('[Audio] Sound loaded and playing:', filename);
                    }, function(err) {
                        console.error('[Audio] Failed to decode audio:', err);
                    });
                };
                reader.readAsArrayBuffer(blob);
            } catch(e) {
                console.error('[Audio] Failed to read sound file:', filename, e);
            }
        }
    }, filename.c_str(), volume, loop);
}

void EmscriptenAudio::stopSound(const std::string& filename) {
    EM_ASM_({
        var filename = UTF8ToString($0);

        if (window.imgeSoundSources && window.imgeSoundSources[filename]) {
            var sources = window.imgeSoundSources[filename];
            for (var i = 0; i < sources.length; i++) {
                try {
                    sources[i].stop();
                } catch(e) {}
            }
            window.imgeSoundSources[filename] = [];
        }
    }, filename.c_str());
}

void EmscriptenAudio::setSoundVolume(float volume) {
    soundVolume = volume;
}

} // namespace imge
