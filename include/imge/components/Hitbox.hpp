#pragma once

#include "imge/core/Component.hpp"
#include "imge/core/Rect.hpp"

#include <string>
#include <vector>

namespace imge {

/**
 * Hitbox component - defines collision boxes
 * Part of builtin components (prefixed with @)
 */
class Hitbox : public Component {
public:
    /**
     * Constructor
     * @param hitboxes Single hitbox [offsetX, offsetY, width, height]
     *                or multiple hitboxes [[...], [...]]
     *                or named hitboxes {"body": [...], "attack": [...]}
     */
    explicit Hitbox(const nlohmann::json& hitboxes);

    /**
     * Get all hitboxes in world space
     * @param owner Object that owns this component
     * @return Vector of hitbox rectangles
     */
    [[nodiscard]] std::vector<Rect> getWorldHitboxes(Object* owner) const;

    /**
     * Get a specific named hitbox
     * @param name Hitbox name
     * @param owner Object that owns this component
     * @return Hitbox rectangle or nullopt if not found
     */
    [[nodiscard]] std::optional<Rect> getNamedHitbox(const std::string& name,
                                                     Object* owner) const;

    void fromJSON(const nlohmann::json& j) override;

private:
    // Hitbox definition: [offsetX, offsetY, width, height]
    struct HitboxDef {
        float offsetX = 0.0f;
        float offsetY = 0.0f;
        float width = 0.0f;
        float height = 0.0f;
        std::string name;  // Optional name for the hitbox
    };

    std::vector<HitboxDef> hitboxes;

    /**
     * Parse hitbox from JSON array
     */
    [[nodiscard]] HitboxDef _parseHitbox(const nlohmann::json& j) const;
};

} // namespace imge
