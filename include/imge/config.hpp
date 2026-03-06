#pragma once

/**
 * IMGE Configuration
 * This file selects the platform implementation based on build configuration
 *
 * Usage:
 *   #include "imge/config.hpp"
 *   imge::EngineImpl engine;  // Automatically uses selected implementation
 */

// Platform implementation selection
// Defined by CMake based on which implementation library is linked
#if defined(IMGE_USE_SDL2)
    #include "imge/impl/SDL2Engine.hpp"
// Add other implementations here in the future:
// #elif defined(IMGE_USE_RAYLIB)
//     #include "imge/impl/RaylibEngine.hpp"
// #elif defined(IMGE_USE_SFML)
//     #include "imge/impl/SFMLEngine.hpp"
// #elif defined(IMGE_USE_WEBGL)
//     #include "imge/impl/WebGLEngine.hpp"
#else
    #error "No IMGE implementation selected! Please link against an implementation library (e.g., imge_sdl2)."
#endif

namespace imge {

// Type alias for the selected implementation
#if defined(IMGE_USE_SDL2)
    using EngineImpl = SDL2Engine;
// #elif defined(IMGE_USE_RAYLIB)
//     using EngineImpl = RaylibEngine;
// #elif defined(IMGE_USE_SFML)
//     using EngineImpl = SFMLEngine;
#endif

} // namespace imge
