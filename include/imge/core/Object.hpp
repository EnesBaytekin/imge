#pragma once

#include "Component.hpp"

#include <memory>
#include <string>
#include <unordered_map>
#include <unordered_set>
#include <vector>

namespace imge {

// Forward declarations
class Scene;

/**
 * Object class - lightweight entity with components
 * Inspired by pygaminal framework
 *
 * Key design decisions:
 * - NO parent-child hierarchy (flat structure)
 * - Name is unique within scene (auto-generated or explicit)
 * - Higher depth = rendered in front (reverse of pygaminal per user request)
 * - Tags are deferred updates for scene consistency
 */
class Object {
public:
    std::string name;
    float x = 0.0f;
    float y = 0.0f;
    float depth = 0.0f;
    std::unordered_set<std::string> tags;
    std::unordered_map<std::string, std::shared_ptr<Component>> components;
    bool dead = false;

    // Pending tag updates (applied at end of frame by Scene)
    std::unordered_set<std::string> _pending_tag_adds;
    std::unordered_set<std::string> _pending_tag_removes;

    // Static counter for auto-generated names
    static inline size_t _id_counter = 0;

    /**
     * Constructor
     * @param x_ X position
     * @param y_ Y position
     * @param name_ Optional explicit name (auto-generated if empty)
     * @param tags_ Optional initial tags
     * @param depth_ Rendering depth (higher = front)
     */
    Object(float x_, float y_,
           const std::string& name_ = "",
           const std::unordered_set<std::string>& tags_ = {},
           float depth_ = 0.0f);

    /**
     * Constructor with initializer_list for tags
     * @param x_ X position
     * @param y_ Y position
     * @param name_ Optional explicit name (auto-generated if empty)
     * @param tags_ Optional initial tags (initializer list)
     * @param depth_ Rendering depth (higher = front)
     */
    Object(float x_, float y_,
           const std::string& name_,
           std::initializer_list<std::string> tags_,
           float depth_ = 0.0f);

    /**
     * Add a tag to this object (deferred update)
     * @param tag Tag to add
     */
    void addTag(const std::string& tag);

    /**
     * Remove a tag from this object (deferred update)
     * @param tag Tag to remove
     */
    void removeTag(const std::string& tag);

    /**
     * Check if object has a specific tag (immediate check)
     * @param tag Tag to check
     * @return true if object has the tag
     */
    [[nodiscard]] bool hasTag(const std::string& tag) const;

    /**
     * Mark object for removal at end of frame
     */
    void kill();

    /**
     * Clear pending tag updates (called by Scene after applying)
     */
    void _clearPendingUpdates();

    /**
     * Generate a unique name for the object
     * @param name_ Desired name (or empty for auto-generated)
     * @return Unique name
     */
    [[nodiscard]] static std::string _generateName(const std::string& name_);

    /**
     * Add a component to this object
     * @param comp Component to add
     * @param explicitName Optional explicit name (auto-generated if empty)
     */
    void addComponent(std::shared_ptr<Component> comp,
                     const std::string& explicitName = "");

    /**
     * Get a component by its unique name
     * @param name Component name
     * @return Component pointer or nullptr if not found
     */
    [[nodiscard]] Component* getComponent(const std::string& name);

    /**
     * Get all components of a specific type (by file_name)
     * @param type Component type/file name
     * @return Vector of component pointers
     */
    [[nodiscard]] std::vector<Component*> getComponents(const std::string& type);

    /**
     * Update all components
     */
    void update();

    /**
     * Draw all components
     */
    void draw();

    /**
     * Create object from JSON data
     * @param objectData JSON object containing object definition
     * @param x_ X position
     * @param y_ Y position
     * @return Shared pointer to created object
     */
    [[nodiscard]] static std::shared_ptr<Object> fromData(
        const nlohmann::json& objectData,
        float x_, float y_);

    /**
     * Create object from .obj file
     * @param filename Path to .obj file
     * @param x_ X position
     * @param y_ Y position
     * @return Shared pointer to created object
     */
    [[nodiscard]] static std::shared_ptr<Object> fromFile(
        const std::string& filename,
        float x_, float y_);
};

} // namespace imge
