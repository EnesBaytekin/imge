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
    src/core/Object.cpp
    src/core/Scene.cpp
    src/core/Engine.cpp
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
    find_package(SDL2_image REQUIRED)
    find_package(SDL2_mixer REQUIRED)

    add_library(imge_sdl2
        src/impl/sdl2/SDL2Renderer.cpp
        src/impl/sdl2/SDL2Input.cpp
        src/impl/sdl2/SDL2Audio.cpp
        src/impl/sdl2/main.cpp
    )

    target_link_libraries(imge_sdl2
        PUBLIC
            imge_core
            SDL2::SDL2
            SDL2::SDL2_image
            SDL2::SDL2_mixer
    )

    target_include_directories(imge_sdl2
        PRIVATE
            ${SDL2_INCLUDE_DIRS}
    )
endif()

# Tests
option(IMGE_BUILD_TESTS "Build tests" ON)
if(IMGE_BUILD_TESTS)
    enable_testing()
    add_subdirectory(tests)
endif()

# Examples
option(IMGE_BUILD_EXAMPLES "Build examples" ON)
if(IMGE_BUILD_EXAMPLES)
    add_subdirectory(examples)
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
