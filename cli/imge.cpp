#include <iostream>
#include <string>
#include <vector>
#include <filesystem>
#include <cstdlib>
#include <fstream>
#include <sstream>

namespace fs = std::filesystem;

// Colors for terminal output
const char* RESET = "\033[0m";
const char* GREEN = "\033[32m";
const char* BLUE = "\033[34m";
const char* YELLOW = "\033[33m";
const char* RED = "\033[31m";

void printUsage() {
    std::cout << BLUE << "IMGE Game Engine CLI" << RESET << std::endl;
    std::cout << "Usage: imge <command> [options]" << std::endl;
    std::cout << std::endl;
    std::cout << "Commands:" << std::endl;
    std::cout << "  " << GREEN << "build" << RESET << " <platform>    Build game for platform (web|desktop)" << std::endl;
    std::cout << "  " << GREEN << "init" << RESET << " <name>         Create new game project" << std::endl;
    std::cout << "  " << GREEN << "help" << RESET << "                Show this help message" << std::endl;
    std::cout << std::endl;
    std::cout << "Examples:" << std::endl;
    std::cout << "  imge build web" << std::endl;
    std::cout << "  imge build desktop" << std::endl;
    std::cout << "  imge init my_game" << std::endl;
}

std::vector<std::string> findFiles(const std::string& extension) {
    std::vector<std::string> files;
    fs::current_path(fs::path("."));

    for (const auto& entry : fs::recursive_directory_iterator(".")) {
        if (entry.is_regular_file()) {
            if (entry.path().extension() == extension) {
                files.push_back(entry.path().string());
            }
        }
    }
    return files;
}

bool hasDirectory(const std::string& name) {
    return fs::exists(name) && fs::is_directory(name);
}

std::vector<std::string> findScriptFiles() {
    std::vector<std::string> scripts;
    if (hasDirectory("scripts")) {
        for (const auto& entry : fs::directory_iterator("scripts")) {
            if (entry.is_regular_file()) {
                auto path = entry.path();
                if (path.extension() == ".hpp" || path.extension() == ".cpp") {
                    scripts.push_back(path.string());
                }
            }
        }
    }
    return scripts;
}

void createWebCMakeLists(const std::string& gameName) {
    std::ofstream file("CMakeLists.txt");
    if (!file.is_open()) {
        std::cerr << RED << "Error: Cannot create CMakeLists.txt" << RESET << std::endl;
        return;
    }

    file << "cmake_minimum_required(VERSION 3.20)\n";
    file << "project(" << gameName << " VERSION 1.0.0)\n\n";

    file << "# C++20 required\n";
    file << "set(CMAKE_CXX_STANDARD 20)\n";
    file << "set(CMAKE_CXX_STANDARD_REQUIRED ON)\n\n";

    // Find IMGE engine
    file << "# Find IMGE engine\n";
    file << "if(DEFINED ENV{IMGE_DIR})\n";
    file << "    set(IMGE_DIR \"$ENV{IMGE_DIR}\")\n";
    file << "elseif(NOT IMGE_DIR)\n";
    file << "    set(IMGE_DIR \"${CMAKE_CURRENT_SOURCE_DIR}/../imge\")\n";
    file << "endif()\n\n";

    file << "get_filename_component(IMGE_DIR \"${IMGE_DIR}\" ABSOLUTE)\n\n";

    file << "include_directories(${IMGE_DIR}/include)\n";
    file << "include_directories(${IMGE_DIR}/cli/templates)\n";
    file << "link_directories(${IMGE_DIR}/build-web)\n\n";

    // nlohmann_json for Emscripten
    file << "include(FetchContent)\n";
    file << "FetchContent_Declare(\n";
    file << "    json\n";
    file << "    URL https://github.com/nlohmann/json/releases/download/v3.11.3/json.tar.xz\n";
    file << ")\n";
    file << "FetchContent_MakeAvailable(json)\n";
    file << "include_directories(${json_SOURCE_DIR}/include)\n\n";

    // Collect script files
    auto scripts = findScriptFiles();
    file << "# Custom scripts\n";
    file << "set(SCRIPT_SOURCES\n";
    for (const auto& script : scripts) {
        file << "    " << script << "\n";
    }
    file << ")\n\n";

    // Create game executable
    file << "add_executable(" << gameName << "\n";
    file << "    ${SCRIPT_SOURCES}\n";
    file << "    ${IMGE_DIR}/cli/templates/game_main_web.cpp\n";
    file << ")\n\n";

    // Embed data files
    file << "# Embed data files\n";
    file << "if(EXISTS \"${CMAKE_CURRENT_SOURCE_DIR}/scenes\")\n";
    file << "    file(GLOB SCENE_FILES \"${CMAKE_CURRENT_SOURCE_DIR}/scenes/*.json\")\n";
    file << "    foreach(scene ${SCENE_FILES})\n";
    file << "        get_filename_component(scene_name \"${scene}\" NAME)\n";
    file << "        target_link_options(" << gameName << " PRIVATE \"SHELL:--embed-file ${scene}@scenes/${scene_name}\")\n";
    file << "    endforeach()\n";
    file << "endif()\n\n";

    file << "if(EXISTS \"${CMAKE_CURRENT_SOURCE_DIR}/objects\")\n";
    file << "    file(GLOB OBJECT_FILES \"${CMAKE_CURRENT_SOURCE_DIR}/objects/*.obj\")\n";
    file << "    foreach(obj ${OBJECT_FILES})\n";
    file << "        get_filename_component(obj_name \"${obj}\" NAME)\n";
    file << "        target_link_options(" << gameName << " PRIVATE \"SHELL:--embed-file ${obj}@objects/${obj_name}\")\n";
    file << "    endforeach()\n";
    file << "endif()\n\n";

    file << "if(EXISTS \"${CMAKE_CURRENT_SOURCE_DIR}/images\")\n";
    file << "    file(GLOB IMAGE_FILES \"${CMAKE_CURRENT_SOURCE_DIR}/images/*\")\n";
    file << "    foreach(img ${IMAGE_FILES})\n";
    file << "        get_filename_component(img_name \"${img}\" NAME)\n";
    file << "        target_link_options(" << gameName << " PRIVATE \"SHELL:--embed-file ${img}@images/${img_name}\")\n";
    file << "    endforeach()\n";
    file << "endif()\n\n";

    file << "if(EXISTS \"${CMAKE_CURRENT_SOURCE_DIR}/sounds\")\n";
    file << "    file(GLOB SOUND_FILES \"${CMAKE_CURRENT_SOURCE_DIR}/sounds/*\")\n";
    file << "    foreach(snd ${SOUND_FILES})\n";
    file << "        get_filename_component(snd_name \"${snd}\" NAME)\n";
    file << "        target_link_options(" << gameName << " PRIVATE \"SHELL:--embed-file ${snd}@sounds/${snd_name}\")\n";
    file << "    endforeach()\n";
    file << "endif()\n\n";

    // Link libraries
    file << "# Link IMGE libraries\n";
    file << "target_link_libraries(" << gameName << "\n";
    file << "    PRIVATE\n";
    file << "        -Wl,--whole-archive ${IMGE_DIR}/build-web/libimge_webgl.a -Wl,--no-whole-archive\n";
    file << "        -Wl,--whole-archive ${IMGE_DIR}/build-web/libimge_core.a -Wl,--no-whole-archive\n";
    file << ")\n\n";

    file << "target_compile_definitions(" << gameName << " PRIVATE IMGE_USE_WEBGL)\n\n";

    // Output settings
    file << "set_target_properties(" << gameName << " PROPERTIES\n";
    file << "    SUFFIX \".html\"\n";
    file << "    RUNTIME_OUTPUT_DIRECTORY \"${CMAKE_CURRENT_SOURCE_DIR}/build/web\"\n";
    file << ")\n\n";

    file << "# Use minimal HTML shell\n";
    file << "target_link_options(" << gameName << " PRIVATE \"SHELL:--shell-file ${IMGE_DIR}/cli/templates/shell_minimal.html\")\n\n";

    file << "# Emscripten flags\n";
    file << "target_link_options(" << gameName << "\n";
    file << "    PRIVATE\n";
    file << "        \"SHELL:-s WASM=1\"\n";
    file << "        \"SHELL:-s ALLOW_MEMORY_GROWTH=1\"\n";
    file << ")\n";

    file.close();
    std::cout << GREEN << "✓" << RESET << " Generated CMakeLists.txt for web" << std::endl;
}

void createDesktopCMakeLists(const std::string& gameName) {
    std::ofstream file("CMakeLists.txt");
    if (!file.is_open()) {
        std::cerr << RED << "Error: Cannot create CMakeLists.txt" << RESET << std::endl;
        return;
    }

    file << "cmake_minimum_required(VERSION 3.20)\n";
    file << "project(" << gameName << " VERSION 1.0.0)\n\n";

    file << "# C++20 required\n";
    file << "set(CMAKE_CXX_STANDARD 20)\n";
    file << "set(CMAKE_CXX_STANDARD_REQUIRED ON)\n\n";

    // Find IMGE engine
    file << "# Find IMGE engine\n";
    file << "if(DEFINED ENV{IMGE_DIR})\n";
    file << "    set(IMGE_DIR \"$ENV{IMGE_DIR}\")\n";
    file << "elseif(NOT IMGE_DIR)\n";
    file << "    set(IMGE_DIR \"${CMAKE_CURRENT_SOURCE_DIR}/../imge\")\n";
    file << "endif()\n\n";

    file << "get_filename_component(IMGE_DIR \"${IMGE_DIR}\" ABSOLUTE)\n\n";

    file << "include_directories(${IMGE_DIR}/include)\n";
    file << "include_directories(${IMGE_DIR}/cli/templates)\n";
    file << "link_directories(${IMGE_DIR}/build)\n\n";

    // Find SDL2
    file << "find_package(SDL2 REQUIRED)\n";
    file << "find_package(SDL2_image REQUIRED)\n";
    file << "find_package(SDL2_mixer REQUIRED)\n";
    file << "find_package(nlohmann_json 3.10.0 REQUIRED)\n\n";

    // Collect script files
    auto scripts = findScriptFiles();
    file << "# Custom scripts\n";
    file << "set(SCRIPT_SOURCES\n";
    for (const auto& script : scripts) {
        file << "    " << script << "\n";
    }
    file << ")\n\n";

    // Create game executable
    file << "add_executable(" << gameName << "\n";
    file << "    ${SCRIPT_SOURCES}\n";
    file << "    ${IMGE_DIR}/cli/templates/game_main_desktop.cpp\n";
    file << ")\n\n";

    // Link libraries
    file << "target_link_libraries(" << gameName << "\n";
    file << "    PRIVATE\n";
    file << "        ${IMGE_DIR}/build/libimge_core.a\n";
    file << "        ${IMGE_DIR}/build/libimge_sdl2.a\n";
    file << "        SDL2::SDL2\n";
    file << "        SDL2_image::SDL2_image\n";
    file << "        SDL2_mixer::SDL2_mixer\n";
    file << "        nlohmann_json::nlohmann_json\n";
    file << ")\n\n";

    file << "target_compile_definitions(" << gameName << " PRIVATE IMGE_USE_SDL2)\n\n";

    // Output settings
    file << "set_target_properties(" << gameName << " PROPERTIES\n";
    file << "    RUNTIME_OUTPUT_DIRECTORY \"${CMAKE_CURRENT_SOURCE_DIR}/build/desktop\"\n";
    file << ")\n";

    file.close();
    std::cout << GREEN << "✓" << RESET << " Generated CMakeLists.txt for desktop" << std::endl;
}

void createWebReadme() {
    fs::create_directories("build/web");
    std::ofstream file("build/web/README.md");
    file << "# How to Run Your Game\n\n";
    file << "## Start HTTP Server\n\n";
    file << "The game needs to be served via HTTP. From the project root:\n\n";
    file << "```bash\n";
    file << "cd build/web\n";
    file << "python3 -m http.server 8080\n";
    file << "```\n\n";
    file << "## Open in Browser\n\n";
    file << "Navigate to: http://localhost:8080/" << fs::current_path().filename().string() << ".html\n\n";
    file << "## Controls\n\n";
    file << "- Arrow Keys or WASD: Move\n";
    file.close();
    std::cout << GREEN << "✓" << RESET << " Created build/web/README.md" << std::endl;
}

void createDesktopReadme() {
    fs::create_directories("build/desktop");
    std::ofstream file("build/desktop/README.md");
    file << "# How to Run Your Game\n\n";
    file << "Simply run:\n\n";
    file << "```bash\n";
    file << "./" << fs::current_path().filename().string() << "\n";
    file << "```\n\n";
    file << "## Controls\n\n";
    file << "- Arrow Keys or WASD: Move\n";
    file.close();
    std::cout << GREEN << "✓" << RESET << " Created build/desktop/README.md" << std::endl;
}

void buildWeb(const std::string& gameName) {
    std::cout << BLUE << "Building for WebAssembly..." << RESET << std::endl;

    // Check if emscripten is available
    if (system("which emcc > /dev/null 2>&1") != 0) {
        std::cerr << RED << "Error: Emscripten not found. Please install and activate emscripten:" << RESET << std::endl;
        std::cerr << "  https://emscripten.org/docs/getting_started/downloads.html" << std::endl;
        return;
    }

    // Check if engine is built for web
    const char* imgeDir = std::getenv("IMGE_DIR");
    std::string enginePath = imgeDir ? imgeDir : "../imge";

    if (!fs::exists(enginePath + "/build-web/libimge_core.a")) {
        std::cerr << YELLOW << "Warning: Engine web libraries not found. Building engine first..." << RESET << std::endl;
        std::string buildCmd = "cd " + enginePath + " && source ~/emsdk/emsdk_env.sh && emmake cmake -B build-web && emmake cmake --build build-web";
        system(buildCmd.c_str());
    }

    // Generate CMakeLists.txt
    createWebCMakeLists(gameName);

    // Create build directory and configure
    std::cout << YELLOW << "Configuring..." << RESET << std::endl;
    std::string configCmd = "source ~/emsdk/emsdk_env.sh && emcmake cmake -B build-web";
    if (system(configCmd.c_str()) != 0) {
        std::cerr << RED << "Error: CMake configuration failed" << RESET << std::endl;
        return;
    }

    // Build
    std::cout << YELLOW << "Building..." << RESET << std::endl;
    std::string buildCmd = "source ~/emsdk/emsdk_env.sh && emmake cmake --build build-web";
    if (system(buildCmd.c_str()) != 0) {
        std::cerr << RED << "Error: Build failed" << RESET << std::endl;
        return;
    }

    // Create README
    createWebReadme();

    std::cout << GREEN << "✓" << RESET << " Build complete! Output in " << GREEN << "build/web/" << RESET << std::endl;
}

void buildDesktop(const std::string& gameName) {
    std::cout << BLUE << "Building for Desktop..." << RESET << std::endl;

    // Check if engine is built
    const char* imgeDir = std::getenv("IMGE_DIR");
    std::string enginePath = imgeDir ? imgeDir : "../imge";

    if (!fs::exists(enginePath + "/build/libimge_core.a")) {
        std::cerr << YELLOW << "Warning: Engine desktop libraries not found. Building engine first..." << RESET << std::endl;
        std::string buildCmd = "cd " + enginePath + " && cmake -B build && cmake --build build";
        system(buildCmd.c_str());
    }

    // Generate CMakeLists.txt
    createDesktopCMakeLists(gameName);

    // Create build directory and configure
    std::cout << YELLOW << "Configuring..." << RESET << std::endl;
    std::string configCmd = "cmake -B build";
    if (system(configCmd.c_str()) != 0) {
        std::cerr << RED << "Error: CMake configuration failed" << RESET << std::endl;
        return;
    }

    // Build
    std::cout << YELLOW << "Building..." << RESET << std::endl;
    std::string buildCmd = "cmake --build build";
    if (system(buildCmd.c_str()) != 0) {
        std::cerr << RED << "Error: Build failed" << RESET << std::endl;
        return;
    }

    // Create README
    createDesktopReadme();

    std::cout << GREEN << "✓" << RESET << " Build complete! Output in " << GREEN << "build/desktop/" << RESET << std::endl;
}

void initProject(const std::string& gameName) {
    std::cout << BLUE << "Creating new game project: " << gameName << RESET << std::endl;

    // Create project directory
    fs::create_directories(gameName);
    fs::current_path(gameName);

    // Create directory structure
    fs::create_directories("scripts");
    fs::create_directories("scenes");
    fs::create_directories("objects");
    fs::create_directories("images");
    fs::create_directories("sounds");

    // Create example scene
    std::ofstream scene("scenes/main_scene.json");
    scene << "{\n";
    scene << "    \"width\": 800,\n";
    scene << "    \"height\": 600,\n";
    scene << "    \"background_color\": \"#222222\",\n";
    scene << "    \"objects\": []\n";
    scene << "}\n";
    scene.close();

    // Create example script
    std::ofstream script("scripts/ExampleComponent.hpp");
    script << "#pragma once\n\n";
    script << "#include \"imge/core/Component.hpp\"\n";
    script << "#include <iostream>\n\n";
    script << "class ExampleComponent : public imge::Component {\n";
    script << "public:\n";
    script << "    void onUpdate(imge::Object* owner) override {\n";
    script << "        std::cout << \"Hello from \" << owner->name << std::endl;\n";
    script << "    }\n";
    script << "};\n";
    script.close();

    std::cout << GREEN << "✓" << RESET << " Project structure created" << std::endl;
    std::cout << "  - scenes/" << std::endl;
    std::cout << "  - objects/" << std::endl;
    std::cout << "  - scripts/" << std::endl;
    std::cout << "  - images/" << std::endl;
    std::cout << "  - sounds/" << std::endl;
    std::cout << std::endl;
    std::cout << "Next steps:" << std::endl;
    std::cout << "  1. cd " << gameName << std::endl;
    std::cout << "  2. Add your game assets and scripts" << std::endl;
    std::cout << "  3. imge build web     # or 'imge build desktop'" << std::endl;
}

int main(int argc, char* argv[]) {
    if (argc < 2) {
        printUsage();
        return 1;
    }

    std::string command = argv[1];

    if (command == "build") {
        if (argc < 3) {
            std::cerr << RED << "Error: Please specify platform (web|desktop)" << RESET << std::endl;
            return 1;
        }

        std::string platform = argv[2];
        std::string gameName = fs::current_path().filename().string();

        if (platform == "web") {
            buildWeb(gameName);
        } else if (platform == "desktop") {
            buildDesktop(gameName);
        } else {
            std::cerr << RED << "Error: Unknown platform '" << platform << "'. Use 'web' or 'desktop'" << RESET << std::endl;
            return 1;
        }

    } else if (command == "init") {
        if (argc < 3) {
            std::cerr << RED << "Error: Please specify project name" << RESET << std::endl;
            return 1;
        }
        initProject(argv[2]);

    } else if (command == "help" || command == "--help" || command == "-h") {
        printUsage();

    } else {
        std::cerr << RED << "Error: Unknown command '" << command << "'" << RESET << std::endl;
        printUsage();
        return 1;
    }

    return 0;
}
