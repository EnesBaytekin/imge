#include "imge/core/Scene.hpp"

#include <cassert>
#include <iostream>

using namespace imge;

void testSceneObjectManagement() {
    std::cout << "Testing Scene object management..." << std::endl;

    auto scene = std::shared_ptr<Scene>(new Scene());

    // Test adding objects
    auto obj1 = std::shared_ptr<Object>(new Object(100.0f, 200.0f, "player"));
    auto obj2 = std::shared_ptr<Object>(new Object(150.0f, 250.0f, "enemy"));

    scene->addObject(obj1);
    scene->addObject(obj2);

    // Objects are added in next update
    assert(scene->objects.empty());

    scene->update();

    // Now objects should be added
    assert(scene->objects.size() == 2);
    assert(scene->getObject("player") != nullptr);
    assert(scene->getObject("enemy") != nullptr);

    // Test getObject
    auto* player = scene->getObject("player");
    assert(player != nullptr);
    assert(player->x == 100.0f);

    // Test removeObject (marks dead)
    scene->removeObject(obj2);
    assert(obj2->dead);

    scene->update();
    assert(scene->objects.size() == 1);
    assert(scene->getObject("enemy") == nullptr);

    std::cout << "  Scene object management tests passed!" << std::endl;
}

void testSceneTags() {
    std::cout << "Testing Scene tag system..." << std::endl;

    auto scene = std::shared_ptr<Scene>(new Scene());

    auto obj1 = std::shared_ptr<Object>(new Object(0.0f, 0.0f, "player", {"hero", "controllable"}));
    auto obj2 = std::shared_ptr<Object>(new Object(100.0f, 100.0f, "enemy1", {"enemy"}));
    auto obj3 = std::shared_ptr<Object>(new Object(200.0f, 200.0f, "enemy2", {"enemy"}));

    scene->addObject(obj1);
    scene->addObject(obj2);
    scene->addObject(obj3);
    scene->update();

    // Test getObjectsByTag
    auto enemies = scene->getObjectsByTag("enemy");
    assert(enemies.size() == 2);

    auto heroes = scene->getObjectsByTag("hero");
    assert(heroes.size() == 1);

    auto nonExistent = scene->getObjectsByTag("nonexistent");
    assert(nonExistent.empty());

    // Test tag mapping
    assert(scene->_tags.find("enemy") != scene->_tags.end());
    assert(scene->_tags.at("enemy").size() == 2);

    std::cout << "  Scene tag system tests passed!" << std::endl;
}

void testSceneDepthSorting() {
    std::cout << "Testing Scene depth sorting..." << std::endl;

    auto scene = std::shared_ptr<Scene>(new Scene());

    auto obj1 = std::shared_ptr<Object>(new Object(0.0f, 0.0f, "background"));
    obj1->depth = 10.0f;

    auto obj2 = std::shared_ptr<Object>(new Object(100.0f, 100.0f, "foreground"));
    obj2->depth = 50.0f;

    auto obj3 = std::shared_ptr<Object>(new Object(50.0f, 50.0f, "ui"));
    obj3->depth = 100.0f;

    scene->addObject(obj1);
    scene->addObject(obj2);
    scene->addObject(obj3);
    scene->update();

    // Get all objects
    auto allObjects = scene->getAllObjects();
    assert(allObjects.size() == 3);

    // Check depth order (higher depth should be drawn first = appear earlier in list)
    assert(allObjects[0]->depth == 100.0f);  // ui
    assert(allObjects[1]->depth == 50.0f);   // foreground
    assert(allObjects[2]->depth == 10.0f);   // background

    std::cout << "  Scene depth sorting tests passed!" << std::endl;
}

void testScenePendingUpdates() {
    std::cout << "Testing Scene pending updates..." << std::endl;

    auto scene = std::shared_ptr<Scene>(new Scene());

    auto obj = std::shared_ptr<Object>(new Object(0.0f, 0.0f, "player"));
    scene->addObject(obj);
    scene->update();

    // Add tag (pending)
    obj->addTag("hero");
    assert(!scene->_tags["hero"].empty()); // Should be added after update

    scene->update(); // Apply pending tag changes

    auto heroes = scene->getObjectsByTag("hero");
    assert(heroes.size() == 1);

    // Remove tag (pending)
    obj->removeTag("hero");
    scene->update();

    auto heroesAfter = scene->getObjectsByTag("hero");
    assert(heroesAfter.empty());

    std::cout << "  Scene pending updates tests passed!" << std::endl;
}

int main() {
    std::cout << "=== IMGE Scene Tests ===" << std::endl;

    testSceneObjectManagement();
    testSceneTags();
    testSceneDepthSorting();
    testScenePendingUpdates();

    std::cout << "\n=== All Scene tests passed! ===" << std::endl;

    return 0;
}
