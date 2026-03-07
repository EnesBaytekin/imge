cmake_minimum_required(VERSION 3.20)
project(imge_cli VERSION 1.0.0)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# Create imge CLI executable
add_executable(imge
    imge.cpp
)

# Link filesystem library (needed for older compilers)
if(CMAKE_CXX_COMPILER_ID STREQUAL "GNU" AND CMAKE_CXX_COMPILER_VERSION VERSION_LESS 9.0)
    target_link_libraries(imge stdc++fs)
endif()

# Install to /usr/local/bin
install(TARGETS imge DESTINATION bin)

# Print installation message
message(STATUS "")
message(STATUS "=== IMGE CLI Build Complete ===")
message(STATUS "Install with: sudo cmake --install .")
message(STATUS "Or run directly: ./imge")
message(STATUS "")
