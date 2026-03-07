cmake_minimum_required(VERSION 3.20)
project(MinimalGame)

# C++20 required
set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# Find IMGE engine
# User can set IMGE_DIR environment variable, or we assume it's in parent directory
if(DEFINED ENV{IMGE_DIR})
    set(IMGE_DIR "$ENV{IMGE_DIR}")
elseif(NOT IMGE_DIR)
    set(IMGE_DIR "${CMAKE_CURRENT_SOURCE_DIR}/../..")
endif()

# Convert to absolute path
get_filename_component(IMGE_DIR "${IMGE_DIR}" ABSOLUTE)

message(STATUS "IMGE_DIR: ${IMGE_DIR}")
message(STATUS "You can override with: IMGE_DIR=/path/to/imge cmake ..")

# Include IMGE headers
include_directories(${IMGE_DIR}/include)

# Add IMGE library directory for linking
link_directories(${IMGE_DIR}/build)

# Find dependencies
find_package(nlohmann_json 3.10.0 REQUIRED)

# Platform selection
option(BUILD_DESKTOP "Build for desktop (SDL2)" ON)
option(BUILD_WEB "Build for web (WebAssembly)" OFF)

# ============================================
# Desktop Build (SDL2)
# ============================================
if(BUILD_DESKTOP AND NOT EMSCRIPTEN)
    find_package(SDL2 REQUIRED)
    find_package(SDL2_image REQUIRED)
    find_package(SDL2_mixer REQUIRED)

    add_executable(minimal_game
        main.cpp
    )

    target_link_libraries(minimal_game
        PRIVATE
            ${IMGE_DIR}/build/libimge_core.a
            ${IMGE_DIR}/build/libimge_sdl2.a
            SDL2::SDL2
            SDL2_image::SDL2_image
            SDL2_mixer::SDL2_mixer
            nlohmann_json::nlohmann_json
    )

    target_compile_definitions(minimal_game
        PRIVATE
            IMGE_USE_SDL2
    )

    message(STATUS "Building for Desktop (SDL2)")
endif()

# ============================================
# Web Build (WebAssembly)
# ============================================
if(BUILD_WEB AND EMSCRIPTEN)
    add_executable(minimal_game
        main.cpp
    )

    target_link_libraries(minimal_game
        PRIVATE
            ${IMGE_DIR}/build-web/libimge_webgl.a
            nlohmann_json::nlohmann_json
    )

    target_compile_definitions(minimal_game
        PRIVATE
            IMGE_USE_WEBGL
    )

    set_target_properties(minimal_game PROPERTIES
        SUFFIX ".html"
    )

    # Emscripten linker flags
    target_link_options(minimal_game
        PRIVATE
            "SHELL:-s WASM=1"
            "SHELL:-s ALLOW_MEMORY_GROWTH=1"
    )

    message(STATUS "Building for Web (WebAssembly)")
endif()

# ============================================
# Build Instructions
# ============================================
if(NOT EMSCRIPTEN)
    message(STATUS "")
    message(STATUS "=== Build Instructions ===")
    message(STATUS "Desktop (SDL2):")
    message(STATUS "  cmake -B build -DBUILD_DESKTOP=ON")
    message(STATUS "  cmake --build build")
    message(STATUS "  ./build/minimal_game")
    message(STATUS "")
    message(STATUS "Web (WebAssembly):")
    message(STATUS "  emcmake cmake -B build-web -DBUILD_WEB=ON")
    message(STATUS "  emmake cmake --build build-web")
    message(STATUS "  cd build-web && emrun minimal_game.html")
    message(STATUS "")
endif()
