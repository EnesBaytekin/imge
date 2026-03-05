#include "imge/core/Object.hpp"

#include <cassert>
#include <iostream>

using namespace imge;

void testObjectCreation() {
    std::cout << "Testing Object creation..." << std::endl;

    // Test auto-generated name
    auto obj1 = std::shared_ptr<Object>(new Object(100.0f, 200.0f));
    assert(obj1->name == "object_0");

    auto obj2 = std::shared_ptr<Object>(new Object(150.0f, 250.0f));
    assert(obj2->name == "object_1");

    // Test explicit name
    auto obj3 = std::shared_ptr<Object>(new Object(300.0f, 400.0f, "player"));
    assert(obj3->name == "player");

    // Test position
    assert(obj1->x == 100.0f);
    assert(obj1->y == 200.0f);

    // Test depth
    assert(obj1->depth == 0.0f);
    std::unordered_set<std::string> emptyTags;
    auto obj4 = std::shared_ptr<Object>(new Object(0.0f, 0.0f, "", emptyTags, 10.0f));
    assert(obj4->depth == 10.0f);

    std::cout << "  Object creation tests passed!" << std::endl;
}

void testObjectTags() {
    std::cout << "Testing Object tags..." << std::endl;

    auto obj = std::shared_ptr<Object>(new Object(0.0f, 0.0f, "", {"enemy", "flying"}));

    // Test hasTag
    assert(obj->hasTag("enemy"));
    assert(obj->hasTag("flying"));
    assert(!obj->hasTag("player"));

    // Test addTag
    obj->addTag("boss");
    assert(obj->hasTag("boss"));

    // Test removeTag
    obj->removeTag("flying");
    assert(!obj->hasTag("flying"));
    assert(obj->hasTag("enemy"));

    // Test pending tag updates
    assert(!obj->_pending_tag_adds.empty());
    assert(!obj->_pending_tag_removes.empty());

    obj->_clearPendingUpdates();
    assert(obj->_pending_tag_adds.empty());
    assert(obj->_pending_tag_removes.empty());

    std::cout << "  Object tag tests passed!" << std::endl;
}

void testObjectKill() {
    std::cout << "Testing Object kill..." << std::endl;

    auto obj = std::shared_ptr<Object>(new Object(0.0f, 0.0f));
    assert(!obj->dead);

    obj->kill();
    assert(obj->dead);

    std::cout << "  Object kill tests passed!" << std::endl;
}

void testObjectComponents() {
    std::cout << "Testing Object components..." << std::endl;

    auto obj = std::shared_ptr<Object>(new Object(0.0f, 0.0f));

    // Create a simple test component
    class TestComponent : public Component {
    public:
        int value = 42;
    };

    auto comp = std::make_shared<TestComponent>();
    obj->addComponent(comp, "test");

    // Test getComponent
    auto* retrieved = obj->getComponent("test");
    assert(retrieved != nullptr);

    auto* testComp = static_cast<TestComponent*>(retrieved);
    assert(testComp->value == 42);

    std::cout << "  Object component tests passed!" << std::endl;
}

int main() {
    std::cout << "=== IMGE Object Tests ===" << std::endl;

    testObjectCreation();
    testObjectTags();
    testObjectKill();
    testObjectComponents();

    std::cout << "\n=== All Object tests passed! ===" << std::endl;

    return 0;
}
