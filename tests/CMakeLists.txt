# Tests CMakeLists.txt

# Test framework (using simple assert-based tests for now)
# Can be extended to use Catch2 or Google Test later

# Core tests
add_executable(test_object
    test_object.cpp
)

target_link_libraries(test_object
    PRIVATE
        imge_core
)

add_test(NAME test_object COMMAND test_object)

add_executable(test_scene
    test_scene.cpp
)

target_link_libraries(test_scene
    PRIVATE
        imge_core
)

add_test(NAME test_scene COMMAND test_scene)
