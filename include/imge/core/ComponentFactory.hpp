#pragma once

#include "imge/core/Component.hpp"
#include "imge/core/Object.hpp"

#include <memory>
#include <string>
#include <unordered_map>
#include <functional>

namespace imge {

/**
 * ComponentFactory - creates components from JSON data
 *
 * Handles both builtin components (prefixed with @) and custom components
 * Custom components can be registered at runtime
 */
class ComponentFactory {
public:
    /**
     * Component creator function signature
     * @param args JSON array with constructor arguments
     * @return Shared pointer to created component
     */
    using CreatorFunc = std::function<std::shared_ptr<Component>(const nlohmann::json& args)>;

    /**
     * Register a custom component type
     * @param componentName Name of the component (without @ prefix)
     * @param creator Function that creates the component from JSON args
     */
    static void registerComponent(const std::string& componentName, CreatorFunc creator);

    /**
     * Create a component from JSON data
     * @param componentData JSON object with "file", "name", "args" fields
     * @return Shared pointer to created component, or nullptr on failure
     */
    [[nodiscard]] static std::shared_ptr<Component> createComponent(const nlohmann::json& componentData);

    /**
     * Create a builtin component
     * @param componentName Component name (with @ prefix, e.g., "@Hitbox")
     * @param args JSON array with constructor arguments
     * @return Shared pointer to created component, or nullptr on failure
     */
    [[nodiscard]] static std::shared_ptr<Component> createBuiltin(const std::string& componentName,
                                                                  const nlohmann::json& args);

private:
    static inline std::unordered_map<std::string, CreatorFunc> customComponents;
};

} // namespace imge
