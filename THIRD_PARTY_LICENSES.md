# Third-party licenses

IMGE depends on and bundles third-party software. The engine source is embedded
into every game build, so the licenses below apply to distributed builds as well
as to this repository.

## Ebitengine

- **Project:** [Ebitengine](https://ebitengine.org/) — a pure-Go 2D game library
- **Module:** `github.com/hajimehoshi/ebiten/v2`
- **Version:** `v2.9.9`
- **License:** [Apache License 2.0](https://www.apache.org/licenses/LICENSE-2.0)
- **Copyright:** Hajime Hoshi and the Ebitengine Authors

Ebitengine is licensed under the permissive Apache License 2.0 — it is **not**
copyleft, so games built with it may be licensed however their author chooses,
including closed-source. IMGE compiles Ebitengine into every game executable, so
`imge build` copies Ebitengine's license text into the output as
`THIRD_PARTY_LICENSES.txt` (beside the executable, and inside the web bundle).
