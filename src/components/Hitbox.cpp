#include "imge/components/Hitbox.hpp"

#include <stdexcept>

namespace imge {

Hitbox::Hitbox(const nlohmann::json& hitboxesData) {
    if (hitboxesData.is_array()) {
        // Single hitbox or multiple hitboxes
        if (hitboxesData.size() == 4 && hitboxesData[0].is_number()) {
            // Single hitbox: [offsetX, offsetY, width, height]
            hitboxes.push_back(_parseHitbox(hitboxesData));
        } else {
            // Multiple hitboxes: [[...], [...]]
            for (const auto& hb : hitboxesData) {
                hitboxes.push_back(_parseHitbox(hb));
            }
        }
    } else if (hitboxesData.is_object()) {
        // Named hitboxes: {"body": [...], "attack": [...]}
        for (auto& [name, value] : hitboxesData.items()) {
            if (value.is_array() && value.size() == 4) {
                auto hb = _parseHitbox(value);
                hb.name = name;
                hitboxes.push_back(hb);
            }
        }
    }
}

std::vector<Rect> Hitbox::getWorldHitboxes(Object* owner) const {
    std::vector<Rect> worldHitboxes;
    worldHitboxes.reserve(hitboxes.size());

    for (const auto& hb : hitboxes) {
        Rect rect;
        rect.x = owner->x + hb.offsetX;
        rect.y = owner->y + hb.offsetY;
        rect.width = hb.width;
        rect.height = hb.height;
        worldHitboxes.push_back(rect);
    }

    return worldHitboxes;
}

std::optional<Rect> Hitbox::getNamedHitbox(const std::string& name,
                                             Object* owner) const {
    for (const auto& hb : hitboxes) {
        if (hb.name == name) {
            Rect rect;
            rect.x = owner->x + hb.offsetX;
            rect.y = owner->y + hb.offsetY;
            rect.width = hb.width;
            rect.height = hb.height;
            return rect;
        }
    }
    return std::nullopt;
}

void Hitbox::fromJSON(const nlohmann::json& j) {
    hitboxes.clear();

    if (j.contains("hitboxes")) {
        const auto& hitboxesData = j["hitboxes"];

        if (hitboxesData.is_array()) {
            if (hitboxesData.size() == 4 && hitboxesData[0].is_number()) {
                // Single hitbox
                hitboxes.push_back(_parseHitbox(hitboxesData));
            } else {
                // Multiple hitboxes
                for (const auto& hb : hitboxesData) {
                    hitboxes.push_back(_parseHitbox(hb));
                }
            }
        } else if (hitboxesData.is_object()) {
            // Named hitboxes
            for (auto& [name, value] : hitboxesData.items()) {
                if (value.is_array() && value.size() == 4) {
                    auto hb = _parseHitbox(value);
                    hb.name = name;
                    hitboxes.push_back(hb);
                }
            }
        }
    }

    // Alternative: args array format
    if (j.contains("args")) {
        const auto& args = j["args"];
        if (args.is_array() && !args.empty()) {
            const auto& arg0 = args[0];
            if (arg0.is_array() && arg0.size() == 4) {
                hitboxes.push_back(_parseHitbox(arg0));
            }
        }
    }
}

Hitbox::HitboxDef Hitbox::_parseHitbox(const nlohmann::json& j) const {
    HitboxDef hb;

    if (j.is_array() && j.size() == 4) {
        hb.offsetX = j[0];
        hb.offsetY = j[1];
        hb.width = j[2];
        hb.height = j[3];
    }

    return hb;
}

} // namespace imge
