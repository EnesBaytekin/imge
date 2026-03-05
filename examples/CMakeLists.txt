# Examples CMakeLists.txt

# Simple example game
add_executable(simple_game
    simple_game/main.cpp
)

target_link_libraries(simple_game
    PRIVATE
        imge_sdl2
)
