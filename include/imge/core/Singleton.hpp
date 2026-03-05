#pragma once

namespace imge {

/**
 * Thread-safe Singleton template
 * Usage:
 * class MyClass : public Singleton<MyClass> {
 *     friend class Singleton<MyClass>;
 * private:
 *     MyClass() = default;
 * public:
 *     void doSomething();
 * };
 *
 * // Usage:
 * MyClass::getInstance()->doSomething();
 */
template<typename T>
class Singleton {
public:
    Singleton(const Singleton&) = delete;
    Singleton& operator=(const Singleton&) = delete;

    static T* getInstance() {
        static T instance;
        return &instance;
    }

protected:
    Singleton() = default;
    virtual ~Singleton() = default;
};

} // namespace imge
