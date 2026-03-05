#pragma once

#include "Vec2.hpp"
#include <optional>

namespace imge {

struct Rect {
    float x = 0.0f;
    float y = 0.0f;
    float width = 0.0f;
    float height = 0.0f;

    constexpr Rect() = default;
    constexpr Rect(float x_, float y_, float w, float h)
        : x(x_), y(y_), width(w), height(h) {}

    // Position access
    [[nodiscard]] constexpr float left() const { return x; }
    [[nodiscard]] constexpr float right() const { return x + width; }
    [[nodiscard]] constexpr float top() const { return y; }
    [[nodiscard]] constexpr float bottom() const { return y + height; }

    // Center
    [[nodiscard]] constexpr Vec2 center() const {
        return Vec2(x + width / 2.0f, y + height / 2.0f);
    }

    // Size
    [[nodiscard]] constexpr Vec2 size() const {
        return Vec2(width, height);
    }

    // Containment
    [[nodiscard]] constexpr bool contains(float px, float py) const {
        return px >= x && px < x + width && py >= y && py < y + height;
    }

    [[nodiscard]] constexpr bool contains(const Vec2& point) const {
        return contains(point.x, point.y);
    }

    [[nodiscard]] constexpr bool contains(const Rect& other) const {
        return other.x >= x && other.right() <= right() &&
               other.y >= y && other.bottom() <= bottom();
    }

    // Intersection
    [[nodiscard]] constexpr bool intersects(const Rect& other) const {
        return x < other.right() && right() > other.x &&
               y < other.bottom() && bottom() > other.y;
    }

    [[nodiscard]] std::optional<Rect> intersection(const Rect& other) const {
        float x1 = std::max(x, other.x);
        float y1 = std::max(y, other.y);
        float x2 = std::min(right(), other.right());
        float y2 = std::min(bottom(), other.bottom());

        if (x1 < x2 && y1 < y2) {
            return Rect(x1, y1, x2 - x1, y2 - y1);
        }
        return std::nullopt;
    }

    // Comparison
    constexpr bool operator==(const Rect& other) const {
        return x == other.x && y == other.y &&
               width == other.width && height == other.height;
    }

    constexpr bool operator!=(const Rect& other) const {
        return !(*this == other);
    }
};

} // namespace imge
