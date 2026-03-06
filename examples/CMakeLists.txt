# Examples CMakeLists.txt

# Simple example game
add_executable(simple_game
    simple_game/main.cpp
)

target_link_libraries(simple_game
    PRIVATE
        imge_sdl2
)

# Full example game - demonstrates all features
add_executable(full_game
    full_game/main.cpp
)

target_link_libraries(full_game
    PRIVATE
        imge_sdl2
)
