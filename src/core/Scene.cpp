#include "imge/core/Scene.hpp"
#include "imge/components/Hitbox.hpp"
#include "imge/services/Screen.hpp"

#include <algorithm>
#include <fstream>
#include <sstream>

namespace imge {

void Scene::addObject(std::shared_ptr<Object> obj) {
    if (obj) {
        _pending_objects.push_back(obj);
    }
}

void Scene::removeObject(std::shared_ptr<Object> obj) {
    if (obj) {
        obj->kill();
    }
}

Object* Scene::getObject(const std::string& name) {
    auto it = objects.find(name);
    if (it != objects.end()) {
        return it->second.get();
    }
    return nullptr;
}

std::vector<Object*> Scene::getObjectsByTag(const std::string& tag) {
    std::vector<Object*> result;

    auto it = _tags.find(tag);
    if (it != _tags.end()) {
        for (const auto& objName : it->second) {
            if (auto* obj = getObject(objName)) {
                result.push_back(obj);
            }
        }
    }

    return result;
}

std::vector<Object*> Scene::getObjectsByTag(const std::string& tag) const {
    std::vector<Object*> result;

    auto it = _tags.find(tag);
    if (it != _tags.end()) {
        for (const auto& objName : it->second) {
            auto objIt = objects.find(objName);
            if (objIt != objects.end() && objIt->second) {
                result.push_back(objIt->second.get());
            }
        }
    }

    return result;
}

std::vector<Object*> Scene::getAllObjects() {
    std::vector<Object*> result;
    result.reserve(objects.size());

    for (auto& [name, obj] : objects) {
        if (obj) {
            result.push_back(obj.get());
        }
    }

    return result;
}

void Scene::update() {
    // Update all objects
    for (auto& [name, obj] : objects) {
        if (obj && !obj->dead) {
            obj->update();
        }
    }

    // Apply pending updates at end of frame
    _applyPendingUpdates();
}

void Scene::draw() {
    // Set background color if specified
    if (backgroundColor.has_value()) {
        Screen::getInstance()->setBackgroundColor(backgroundColor.value());
    }

    // Collect all non-dead objects
    std::vector<Object*> aliveObjects;
    for (auto& [name, obj] : objects) {
        if (obj && !obj->dead) {
            aliveObjects.push_back(obj.get());
        }
    }

    // Sort by depth (higher depth = drawn first, per user request)
    std::sort(aliveObjects.begin(), aliveObjects.end(),
              [](const Object* a, const Object* b) {
                  return a->depth > b->depth; // Higher depth = front
              });

    // Draw all objects
    for (auto* obj : aliveObjects) {
        obj->draw();
    }
}

void Scene::_applyPendingUpdates() {
    // 1. Remove dead objects
    std::vector<std::string> deadNames;
    for (auto& [name, obj] : objects) {
        if (obj && obj->dead) {
            deadNames.push_back(name);
        }
    }
    for (const auto& name : deadNames) {
        auto it = objects.find(name);
        if (it != objects.end()) {
            _removeObjectNow(it->second);
        }
    }

    // 2. Add pending objects
    for (auto& obj : _pending_objects) {
        _addObjectNow(obj);
    }
    _pending_objects.clear();

    // 3. Apply pending tag updates
    for (auto& [name, obj] : objects) {
        if (obj) {
            // Add pending tags
            for (const auto& tag : obj->_pending_tag_adds) {
                _addTagNow(obj, tag);
            }

            // Remove pending tags
            for (const auto& tag : obj->_pending_tag_removes) {
                _removeTagNow(obj, tag);
            }

            // Clear pending
            obj->_clearPendingUpdates();
        }
    }
}

void Scene::_addObjectNow(std::shared_ptr<Object> obj) {
    if (!obj) {
        return;
    }

    // Handle name conflicts
    std::string name = obj->name;
    size_t counter = 2;
    while (objects.find(name) != objects.end()) {
        name = obj->name + "_" + std::to_string(counter);
        counter++;
    }
    obj->name = name;

    // Add to objects dict
    objects[name] = obj;

    // Add to tag index
    for (const auto& tag : obj->tags) {
        _addTagNow(obj, tag);
    }
}

void Scene::_removeObjectNow(std::shared_ptr<Object> obj) {
    if (!obj) {
        return;
    }

    // Remove from objects dict
    objects.erase(obj->name);

    // Remove from tag index
    for (const auto& tag : obj->tags) {
        _removeTagNow(obj, tag);
    }
}

void Scene::_addTagNow(std::shared_ptr<Object> obj, const std::string& tag) {
    if (!obj) {
        return;
    }

    auto& objList = _tags[tag];
    if (std::find(objList.begin(), objList.end(), obj->name) == objList.end()) {
        objList.push_back(obj->name);
    }
}

void Scene::_removeTagNow(std::shared_ptr<Object> obj, const std::string& tag) {
    if (!obj) {
        return;
    }

    auto it = _tags.find(tag);
    if (it != _tags.end()) {
        auto& objList = it->second;
        objList.erase(std::remove(objList.begin(), objList.end(), obj->name),
                      objList.end());

        // Remove tag entry if empty
        if (objList.empty()) {
            _tags.erase(it);
        }
    }
}

std::shared_ptr<Scene> Scene::fromFile(const std::string& filename) {
    std::ifstream file(filename);
    if (!file.is_open()) {
        return nullptr;
    }

    std::stringstream buffer;
    buffer << file.rdbuf();
    file.close();

    nlohmann::json sceneData = nlohmann::json::parse(buffer.str());
    return fromData(sceneData);
}

std::shared_ptr<Scene> Scene::fromData(const nlohmann::json& sceneData) {
    auto scene = std::make_shared<Scene>();

    // Read scene properties
    scene->width = sceneData.value("width", 800);
    scene->height = sceneData.value("height", 600);

    if (sceneData.contains("background_color")) {
        scene->backgroundColor = sceneData["background_color"].get<std::string>();
    }

    if (sceneData.contains("background_image")) {
        scene->backgroundImage = sceneData["background_image"].get<std::string>();
    }

    // Read objects
    if (sceneData.contains("objects")) {
        for (const auto& objectData : sceneData["objects"]) {
            auto obj = _loadObject(objectData);
            if (obj) {
                scene->addObject(obj);
            }
        }
    }

    return scene;
}

std::shared_ptr<Object> Scene::_loadObject(const nlohmann::json& objectData) {
    // Check if it's an external .obj file reference
    if (objectData.contains("file")) {
        // Load from .obj file
        std::string objFile = objectData["file"].get<std::string>();
        float x = objectData["x"].get<float>();
        float y = objectData["y"].get<float>();

        return Object::fromFile(objFile, x, y);
    } else {
        // Direct definition in scene file
        float x = objectData["x"].get<float>();
        float y = objectData["y"].get<float>();

        return Object::fromData(objectData, x, y);
    }
}

bool Scene::checkCollision(Object* obj1, Object* obj2) const {
    // Get Hitbox components from both objects
    auto* hitbox1 = obj1->getComponent("Hitbox");
    auto* hitbox2 = obj2->getComponent("Hitbox");

    // Both objects need Hitbox components to collide
    if (!hitbox1 || !hitbox2) {
        return false;
    }

    // Get world-space hitboxes
    auto* hb1 = static_cast<Hitbox*>(hitbox1);
    auto* hb2 = static_cast<Hitbox*>(hitbox2);

    auto rects1 = hb1->getWorldHitboxes(obj1);
    auto rects2 = hb2->getWorldHitboxes(obj2);

    // Check if any hitbox from obj1 overlaps with any hitbox from obj2
    for (const auto& r1 : rects1) {
        for (const auto& r2 : rects2) {
            // AABB collision detection
            if (r1.x < r2.x + r2.width &&
                r1.x + r1.width > r2.x &&
                r1.y < r2.y + r2.height &&
                r1.y + r1.height > r2.y) {
                return true;
            }
        }
    }

    return false;
}

std::vector<Object*> Scene::getCollisions(Object* obj) const {
    std::vector<Object*> collisions;

    for (const auto& [name, otherObj] : objects) {
        if (otherObj.get() == obj) {
            continue; // Don't check against self
        }
        if (otherObj->dead) {
            continue; // Skip dead objects
        }
        if (checkCollision(obj, otherObj.get())) {
            collisions.push_back(otherObj.get());
        }
    }

    return collisions;
}

std::vector<Object*> Scene::getCollisionsWithTag(Object* obj, const std::string& tag) const {
    std::vector<Object*> collisions;

    // Get objects with the specified tag
    auto taggedObjects = getObjectsByTag(tag);

    for (auto* otherObj : taggedObjects) {
        if (otherObj == obj) {
            continue; // Don't check against self
        }
        if (otherObj->dead) {
            continue; // Skip dead objects
        }
        if (checkCollision(obj, otherObj)) {
            collisions.push_back(otherObj);
        }
    }

    return collisions;
}

} // namespace imge
