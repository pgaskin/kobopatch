# kobopatch
An improved alternative to patch32lsb. WIP.

**Progress:**
- [X] Core patching functionality (./patchlib)
- [X] Drop-in replacement for patch32lsb with exactly the same output and features (./patch32lsb)
- [X] All-in-one patcher (base features) (./kobopatch)
- [X] All-in-one patcher (./kobopatch)
- [X] kobopatch: support new format
- [X] kobopatch: support old format
- [X] kobopatch: pluggable format system
- [ ] Zlib support
- [ ] Automatic builds
- [X] Alternative patch format (which has more reliable and has less complex parsing)
- [X] Manually check hashes of output from both batch formats with each other and the old patcher with all patches enabled to ensure reliability
- [X] Patch overrides to force enable/disable patches in a portable way.
- [X] Patch consistency check

**Improvements/Goals:**
- More readable code
- Unit tests
- Built-in support for generating kobo update files
- More modular
- Embeddable into another application
- Single file
- Better consistency checking of config and result
- Easier cross-platform build support without external deps
- Optional improved patch format
- Faster and lower memory consumption
- No need for temp files
- Save and restore lists of enabled patches (coming soon)
- more (coming soon)
- Zlib support (coming soon)

**Testing:**
- Unit tests (automatically run by Travis CI): `go test -v ./...`
- Real tests (compares original and new): rewrite coming soon, already done manually