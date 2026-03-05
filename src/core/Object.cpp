#include "imge/core/Object.hpp"

#include <fstream>
#include <sstream>

namespace imge {

Object::Object(float x_, float y_,
               const std::string& name_,
               const std::unordered_set<std::string>& tags_,
               float depth_)
    : name(_generateName(name_))
    , x(x_)
    , y(y_)
    , depth(depth_)
    , tags(tags_)
{
}

void Object::addTag(const std::string& tag) {
    if (tags.find(tag) != tags.end()) {
        return; // Already has this tag
    }
    tags.insert(tag);
    _pending_tag_adds.insert(tag);
    _pending_tag_removes.erase(tag);
}

void Object::removeTag(const std::string& tag) {
    if (tags.find(tag) == tags.end()) {
        return; // Doesn't have this tag
    }
    tags.erase(tag);
    _pending_tag_removes.insert(tag);
    _pending_tag_adds.erase(tag);
}

bool Object::hasTag(const std::string& tag) const {
    return tags.find(tag) != tags.end();
}

void Object::kill() {
    dead = true;
}

void Object::_clearPendingUpdates() {
    _pending_tag_adds.clear();
    _pending_tag_removes.clear();
}

std::string Object::_generateName(const std::string& name_) {
    if (!name_.empty()) {
        // Try to use the provided name
        // Note: Conflict resolution will be handled by Scene
        return name_;
    }

    // Auto-generate name
    std::string generated = "object_" + std::to_string(_id_counter);
    _id_counter++;
    return generated;
}

void Object::addComponent(std::shared_ptr<Component> comp,
                         const std::string& explicitName) {
    // For now, we'll use the explicit name if provided
    // Auto-generation will be added when we have the Scene context
    // This is a simplified version - the full version will need Scene access
    std::string componentName;
    if (!explicitName.empty()) {
        componentName = explicitName;
    } else {
        componentName = "component_" + std::to_string(components.size());
    }

    components[componentName] = comp;
}

Component* Object::getComponent(const std::string& name) {
    auto it = components.find(name);
    if (it != components.end()) {
        return it->second.get();
    }
    return nullptr;
}

std::vector<Component*> Object::getComponents(const std::string& type) {
    std::vector<Component*> result;
    // This will be implemented when we add type tracking to components
    // For now, return empty vector
    (void)type;
    return result;
}

void Object::update() {
    for (auto& [name, comp] : components) {
        if (comp) {
            comp->onUpdate(this);
        }
    }
}

void Object::draw() {
    for (auto& [name, comp] : components) {
        if (comp) {
            comp->onDraw(this);
        }
    }
}

std::shared_ptr<Object> Object::fromData(const nlohmann::json& objectData,
                                         float x_, float y_) {
    std::string name = objectData.value("name", "");
    float depth = objectData.value("depth", 0.0f);

    // Read tags if present
    std::unordered_set<std::string> tags_;
    if (objectData.contains("tags")) {
        for (const auto& tag : objectData["tags"]) {
            tags_.insert(tag.get<std::string>());
        }
    }

    auto obj = std::make_shared<Object>(x_, y_, name, tags_, depth);

    // Load components
    if (objectData.contains("components")) {
        // This will be implemented when we have the component loader
        // For now, just skip
    }

    return obj;
}

std::shared_ptr<Object> Object::fromFile(const std::string& filename,
                                         float x_, float y_) {
    std::ifstream file(filename);
    if (!file.is_open()) {
        return nullptr;
    }

    std::stringstream buffer;
    buffer << file.rdbuf();
    file.close();

    nlohmann::json objectData = nlohmann::json::parse(buffer.str());
    return fromData(objectData, x_, y_);
}

} // namespace imge
