#!/bin/bash
time patchlib/testdata/patch32lsb -i patchlib/testdata/nickel -o ne -p patchlib/testdata/nickel.patch.all
time go run patch32lsb/patch32lsb.go -i patchlib/testdata/nickel -o n -p patchlib/testdata/nickel.patch.all || {
    rm ne
    exit 1
}
sha256sum ne n
icdiff <(xxd ne) <(xxd n)
rm n ne