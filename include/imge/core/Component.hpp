#pragma once

#include <nlohmann/json.hpp>
#include <memory>
#include <string>

namespace imge {

// Forward declaration
class Object;

/**
 * Base class for all components
 * Builtin components and custom components are identical in implementation
 *
 * Component naming convention (from pygaminal):
 * - No "Component" suffix (Image, not ImageComponent)
 * - Builtin: @Image, @Animation prefix
 * - Custom: Just the class name
 */
class Component {
public:
    virtual ~Component() = default;

    /**
     * Called once when component is created (first frame)
     * @param owner The object that owns this component
     */
    virtual void onCreate(Object* owner) {
        (void)owner;
    }

    /**
     * Called every frame
     * @param owner The object that owns this component
     */
    virtual void onUpdate(Object* owner) {
        (void)owner;
    }

    /**
     * Called every frame after all updates
     * @param owner The object that owns this component
     */
    virtual void onDraw(Object* owner) {
        (void)owner;
    }

    /**
     * Load component properties from JSON
     * Each component implements this manually for explicit control
     * @param j JSON object containing component data
     */
    virtual void fromJSON(const nlohmann::json& j) {
        (void)j;
    }

    /**
     * Save component properties to JSON
     * Each component implements this manually for explicit control
     * @param j JSON object to write component data to
     */
    virtual void toJSON(nlohmann::json& j) const {
        (void)j;
    }
};

} // namespace imge
