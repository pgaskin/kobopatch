# kobopatch
An improved alternative to patch32lsb. WIP.

**Progress:**
- [X] Core patching functionality (./patchlib)
- [X] Drop-in replacement for patch32lsb with exactly the same output and features (./patch32lsb)
- [X] All-in-one patcher (base features) (./kobopatch)
- [ ] All-in-one patcher (./kobopatch)
- [ ] Zlib support
- [ ] Automatic builds
- [X] Alternative patch format (which has more reliable and has less complex parsing)

**Improvements/Goals:**
- More readable code
- Unit tests
- Built-in support for generating kobo update files (coming soon)
- More modular
- Embeddable into another application
- Single file
- Easier cross-platform build support without external deps
- Optional improved patch format (coming soon)
- Save and restore lists of enabled patches (coming soon)
- more (coming soon)
- Zlib support (coming soon)

**Testing:**
- Requires Linux and icdiff
- Unit tests (automatically run by Travis CI): `go test -v ./...`
- Real tests (compares original and new): `./testalllibnickel.sh` and `./testallnickel.sh`