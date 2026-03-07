cmake_minimum_required(VERSION 3.20)
project(IMGE VERSION 0.1.0 LANGUAGES CXX)

# C++20/23 required
set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

# Compiler flags
if(MSVC)
    add_compile_options(/W4)
else()
    add_compile_options(-Wall -Wextra -Wpedantic)
endif()

# Find dependencies
find_package(nlohmann_json 3.10.0 REQUIRED)

# Include directories
include_directories(${CMAKE_SOURCE_DIR}/include)

# Core library
add_library(imge_core
    src/core/Component.cpp
    src/core/ComponentFactory.cpp
    src/core/Object.cpp
    src/core/Scene.cpp
    src/core/Engine.cpp
    src/services/Screen.cpp
    src/services/Input.cpp
    src/services/Audio.cpp
    src/components/Hitbox.cpp
    src/components/Image.cpp
    src/components/Animation.cpp
)

target_link_libraries(imge_core
    PUBLIC nlohmann_json::nlohmann_json
)

target_include_directories(imge_core
    PUBLIC
        $<BUILD_INTERFACE:${CMAKE_SOURCE_DIR}/include>
        $<INSTALL_INTERFACE:include>
)

# SDL2 implementation library (optional)
option(IMGE_WITH_SDL2 "Build SDL2 implementation" ON)

if(IMGE_WITH_SDL2)
    find_package(SDL2 REQUIRED)

    add_library(imge_sdl2
        src/impl/sdl2/SDL2Renderer.cpp
        src/impl/sdl2/SDL2Input.cpp
        src/impl/sdl2/SDL2Audio.cpp
        src/impl/sdl2/SDL2Engine.cpp
    )

    target_link_libraries(imge_sdl2
        PUBLIC
            imge_core
            SDL2
            SDL2_image
            SDL2_mixer
    )

    target_compile_definitions(imge_sdl2
        PUBLIC
            IMGE_USE_SDL2
    )

    target_include_directories(imge_sdl2
        PRIVATE
            ${SDL2_INCLUDE_DIRS}
    )
endif()

# WebAssembly implementation library (optional)
option(IMGE_WITH_WEBGL "Build WebAssembly implementation" OFF)

if(IMGE_WITH_WEBGL)
    # Check if we're using Emscripten toolchain
    if(NOT EMSCRIPTEN)
        message(WARNING "WebAssembly implementation requires Emscripten toolchain!")
        message(WARNING "Use: emcmake cmake ..")
    endif()

    add_library(imge_webgl
        src/impl/emscripten/EmscriptenRenderer.cpp
        src/impl/emscripten/EmscriptenInput.cpp
        src/impl/emscripten/WebGLEngine.cpp
    )

    target_link_libraries(imge_webgl
        PUBLIC
            imge_core
    )

    target_compile_definitions(imge_webgl
        PUBLIC
            IMGE_USE_WEBGL
    )

    # Emscripten-specific linker flags
    if(EMSCRIPTEN)
        set_target_properties(imge_webgl PROPERTIES
            SUFFIX ".html"
        )

        # Linker flags for Emscripten
        target_link_options(imge_webgl
            PUBLIC
                "SHELL:-s USE_SDL=2"
                "SHELL:-s WASM=1"
                "SHELL:-s ALLOW_MEMORY_GROWTH=1"
                "SHELL:-s ASYNCIFY"
                "SHELL:-s EXPORTED_RUNTIME_METHODS=['ccall','cwrap']"
        )
    endif()
endif()

# Tests
option(IMGE_BUILD_TESTS "Build tests" ON)
if(IMGE_BUILD_TESTS)
    enable_testing()
    add_subdirectory(tests)
endif()

# Templates (built only when explicitly requested)
option(IMGE_BUILD_TEMPLATES "Build template projects" OFF)
if(IMGE_BUILD_TEMPLATES)
    add_subdirectory(templates)
endif()

# Installation
include(GNUInstallDirs)
install(TARGETS imge_core
    ARCHIVE DESTINATION ${CMAKE_INSTALL_LIBDIR}
    LIBRARY DESTINATION ${CMAKE_INSTALL_LIBDIR}
    RUNTIME DESTINATION ${CMAKE_INSTALL_BINDIR}
)

install(DIRECTORY include/
    DESTINATION ${CMAKE_INSTALL_INCLUDEDIR}
)
