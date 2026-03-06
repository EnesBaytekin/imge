#pragma once

#include "Object.hpp"

#include <memory>
#include <optional>
#include <string>
#include <unordered_map>
#include <vector>

namespace imge {

/**
 * Scene class - manages objects and tags
 * Inspired by pygaminal framework with deferred update pattern
 *
 * Key features:
 * - O(1) object lookup by name (dictionary storage)
 * - O(1) tag-based queries (tag mapping)
 * - Deferred updates for consistency (applied at end of frame)
 * - Depth sorting: higher depth = rendered first (per user request)
 */
class Scene {
public:
    // Dictionary for O(1) object lookup by name
    std::unordered_map<std::string, std::shared_ptr<Object>> objects;

    // Tag mapping for O(1) tag queries: tag -> vector of object names
    std::unordered_map<std::string, std::vector<std::string>> _tags;

    // Pending objects to add at end of frame
    std::vector<std::shared_ptr<Object>> _pending_objects;

    // Scene properties
    int width = 800;
    int height = 600;
    std::optional<std::string> backgroundColor;
    std::optional<std::string> backgroundImage;

    Scene() = default;
    ~Scene() = default;

    /**
     * Add an object to the scene (deferred - added at end of frame)
     * @param obj Object to add
     */
    void addObject(std::shared_ptr<Object> obj);

    /**
     * Remove an object from the scene (marks dead, removed at end of frame)
     * @param obj Object to remove
     */
    void removeObject(std::shared_ptr<Object> obj);

    /**
     * Get an object by its unique name (O(1) lookup)
     * @param name Object name
     * @return Object pointer or nullptr if not found
     */
    [[nodiscard]] Object* getObject(const std::string& name);

    /**
     * Get all objects that have a specific tag (O(1) lookup via tag mapping)
     * @param tag Tag to query
     * @return Vector of object pointers
     */
    [[nodiscard]] std::vector<Object*> getObjectsByTag(const std::string& tag);

    /**
     * Get all objects that have a specific tag (const version)
     * @param tag Tag to query
     * @return Vector of object pointers
     */
    [[nodiscard]] std::vector<Object*> getObjectsByTag(const std::string& tag) const;

    /**
     * Get all objects in the scene
     * @return Vector of all object pointers
     */
    [[nodiscard]] std::vector<Object*> getAllObjects();

    /**
     * Update all objects, then apply pending updates
     */
    void update();

    /**
     * Draw all objects sorted by depth (higher depth = drawn first)
     */
    void draw();

    /**
     * Collision detection - check if two objects' hitboxes overlap
     * @param obj1 First object
     * @param obj2 Second object
     * @return true if hitboxes overlap
     */
    [[nodiscard]] bool checkCollision(Object* obj1, Object* obj2) const;

    /**
     * Get all objects colliding with a given object
     * @param obj Object to check collisions against
     * @return Vector of colliding objects
     */
    [[nodiscard]] std::vector<Object*> getCollisions(Object* obj) const;

    /**
     * Get objects with a specific tag that are colliding with a given object
     * @param obj Object to check collisions against
     * @param tag Tag to filter by
     * @return Vector of colliding objects with the specified tag
     */
    [[nodiscard]] std::vector<Object*> getCollisionsWithTag(Object* obj, const std::string& tag) const;

    /**
     * Load scene from JSON file
     * @param filename Path to scene JSON file
     * @return Shared pointer to loaded scene
     */
    [[nodiscard]] static std::shared_ptr<Scene> fromFile(const std::string& filename);

    /**
     * Create scene from JSON data
     * @param sceneData JSON object containing scene definition
     * @return Shared pointer to created scene
     */
    [[nodiscard]] static std::shared_ptr<Scene> fromData(const nlohmann::json& sceneData);

private:
    /**
     * Apply all pending updates at end of frame
     * 1. Remove dead objects
     * 2. Add pending objects
     * 3. Apply pending tag changes
     */
    void _applyPendingUpdates();

    /**
     * Immediately add an object to the scene (internal use)
     * @param obj Object to add
     */
    void _addObjectNow(std::shared_ptr<Object> obj);

    /**
     * Immediately remove an object from the scene (internal use)
     * @param obj Object to remove
     */
    void _removeObjectNow(std::shared_ptr<Object> obj);

    /**
     * Immediately add a tag to the index (internal use)
     * @param obj Object that owns the tag
     * @param tag Tag to add
     */
    void _addTagNow(std::shared_ptr<Object> obj, const std::string& tag);

    /**
     * Immediately remove a tag from the index (internal use)
     * @param obj Object that owns the tag
     * @param tag Tag to remove
     */
    void _removeTagNow(std::shared_ptr<Object> obj, const std::string& tag);

    /**
     * Load an object from JSON data
     * Handles both direct definition and .obj file references
     * @param objectData JSON object containing object definition
     * @return Shared pointer to loaded object
     */
    [[nodiscard]] static std::shared_ptr<Object> _loadObject(const nlohmann::json& objectData);
};

} // namespace imge
