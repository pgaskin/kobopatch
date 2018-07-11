# kobopatch
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fgeek1011%2Fkobopatch.svg?type=shield)](https://app.fossa.io/projects/git%2Bgithub.com%2Fgeek1011%2Fkobopatch?ref=badge_shield)

An improved patching system for Kobo eReaders. See https://www.mobileread.com/forums/showthread.php?t=297338 . Download patches for v4.9.11311+ [here](https://github.com/geek1011/kobopatch-patches/releases/latest).

**Progress:**
- [X] Core patching functionality (./patchlib)
- [X] Drop-in replacement for patch32lsb with exactly the same output and features (./patch32lsb)
- [X] All-in-one patcher (base features) (./kobopatch)
- [X] All-in-one patcher (./kobopatch)
- [X] kobopatch: support new format
- [X] kobopatch: support old format
- [X] kobopatch: pluggable format system
- [X] Automatic builds
- [X] Alternative patch format (which has more reliable and has less complex parsing)
- [X] Manually check hashes of output from both batch formats with each other and the old patcher with all patches enabled to ensure reliability
- [X] Patch overrides to force enable/disable patches in a portable way.
- [X] Patch consistency check
- [X] Convert old patch zips
- [X] Translation file support
- [X] Zlib support
- [X] CSS extraction tool (./cssextract)
- [X] Support for adding additional files

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
- Save and restore lists of enabled patches
- Translation file support
- more (coming soon)
- Zlib support (coming soon)

**Testing:**
- Unit tests (automatically run by Travis CI): `go test -v ./...`
- Real tests (compares original and new): rewrite coming soon, already done manually

## License
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fgeek1011%2Fkobopatch.svg?type=large)](https://app.fossa.io/projects/git%2Bgithub.com%2Fgeek1011%2Fkobopatch?ref=badge_large)