# kobopatch
An improved patching system for Kobo eReaders. See https://www.mobileread.com/forums/showthread.php?t=297338. Download patches for v4.9.11311+ [here](https://github.com/pgaskin/kobopatch-patches/releases/latest).

## Features
- Zlib replacement.
- Add additional files.
- Translation file support.
- Simplified BLX instruction replacement.
- Multi-version configuration file.
- Extensible patch file.
- Built-in generation of Kobo update files.
- Additional instructions.
- Single executable.
- Automated testing of patches.
- Comprehensive log file and error messages.
- Modular and embeddable.
- Structured patch file format.
- Backwards-compatible with old patch format.

## Usage
```
Usage: kobopatch [OPTIONS] [CONFIG_FILE]

Options:
  -f, --firmware string   firmware file to be used (can also use a testdata tarball from kobopatch-patches)
  -h, --help              show this help text
  -t, --run-tests         test all patches (instead of running kobopatch)

If CONFIG_FILE is not specified, kobopatch will use ./kobopatch.yaml.
```

```
cssextract extracts zlib-compressed from a binary file
Usage: cssextract BINARY_FILE
```

```
symdump dumps symbol addresses from an ARMv6+ 32-bit ELF executable
Usage: symdump BINARY_FILE
```

```
Usage: kobopatch-apply [OPTIONS]

Options:
  -h, --help                  show this help text
  -i, --input string          the file to patch (required)
  -o, --output string         the file to write the patched output to (will be overwritten if exists) (required)
  -p, --patch-file string     the file containing the patches (required)
  -f, --patch-format string   the patch format (one of: kobopatch,patch32lsb) (default "kobopatch")
  -v, --verbose               show verbose output from patchlib
```