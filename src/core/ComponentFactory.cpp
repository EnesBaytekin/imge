#include "imge/core/ComponentFactory.hpp"
#include "imge/components/Hitbox.hpp"

namespace imge {

void ComponentFactory::registerComponent(const std::string& componentName, CreatorFunc creator) {
    customComponents[componentName] = std::move(creator);
}

std::shared_ptr<Component> ComponentFactory::createComponent(const nlohmann::json& componentData) {
    if (!componentData.contains("file")) {
        return nullptr;
    }

    std::string file = componentData["file"].get<std::string>();
    nlohmann::json args = componentData.value("args", nlohmann::json::array());

    std::shared_ptr<Component> comp = nullptr;

    // Check if it's a builtin component (starts with @)
    if (file.starts_with("@")) {
        comp = createBuiltin(file, args);
    } else {
        // Check if it's a registered custom component
        auto it = customComponents.find(file);
        if (it != customComponents.end()) {
            comp = it->second(args);
        }
    }

    return comp;
}

std::shared_ptr<Component> ComponentFactory::createBuiltin(const std::string& componentName,
                                                           const nlohmann::json& args) {
    // Handle builtin components
    if (componentName == "@Hitbox") {
        // Hitbox expects JSON array: [[x, y, w, h]] or just [[x, y, w, h]]
        if (args.is_array() && args.size() > 0) {
            return std::make_shared<Hitbox>(args);
        }
    }

    // Add more builtin components here (@Image, @Animation, etc.)

    return nullptr;
}

} // namespace imge
