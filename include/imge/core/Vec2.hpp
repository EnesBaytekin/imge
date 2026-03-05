#pragma once

#include <cmath>

namespace imge {

struct Vec2 {
    float x = 0.0f;
    float y = 0.0f;

    constexpr Vec2() = default;
    constexpr Vec2(float x_, float y_) : x(x_), y(y_) {}

    // Arithmetic operators
    constexpr Vec2 operator+(const Vec2& other) const {
        return Vec2(x + other.x, y + other.y);
    }

    constexpr Vec2 operator-(const Vec2& other) const {
        return Vec2(x - other.x, y - other.y);
    }

    constexpr Vec2 operator*(float scalar) const {
        return Vec2(x * scalar, y * scalar);
    }

    constexpr Vec2 operator/(float scalar) const {
        return Vec2(x / scalar, y / scalar);
    }

    // Compound assignment operators
    Vec2& operator+=(const Vec2& other) {
        x += other.x;
        y += other.y;
        return *this;
    }

    Vec2& operator-=(const Vec2& other) {
        x -= other.x;
        y -= other.y;
        return *this;
    }

    Vec2& operator*=(float scalar) {
        x *= scalar;
        y *= scalar;
        return *this;
    }

    Vec2& operator/=(float scalar) {
        x /= scalar;
        y /= scalar;
        return *this;
    }

    // Comparison
    constexpr bool operator==(const Vec2& other) const {
        return x == other.x && y == other.y;
    }

    constexpr bool operator!=(const Vec2& other) const {
        return !(*this == other);
    }

    // Vector operations
    [[nodiscard]] float length() const {
        return std::sqrt(x * x + y * y);
    }

    [[nodiscard]] float lengthSquared() const {
        return x * x + y * y;
    }

    [[nodiscard]] Vec2 normalized() const {
        float len = length();
        if (len > 0.0f) {
            return Vec2(x / len, y / len);
        }
        return Vec2(0.0f, 0.0f);
    }

    [[nodiscard]] float dot(const Vec2& other) const {
        return x * other.x + y * other.y;
    }

    [[nodiscard]] float cross(const Vec2& other) const {
        return x * other.y - y * other.x;
    }

    [[nodiscard]] float distanceTo(const Vec2& other) const {
        return (*this - other).length();
    }

    [[nodiscard]] float distanceSquaredTo(const Vec2& other) const {
        return (*this - other).lengthSquared();
    }
};

} // namespace imge
