#!/bin/bash
time patchlib/testdata/patch32lsb -i patchlib/testdata/libnickel.so.1.0.0 -o lnse -p patchlib/testdata/libnickel.so.1.0.0.patch.all
time go run patch32lsb/patch32lsb.go -i patchlib/testdata/libnickel.so.1.0.0 -o lns -p patchlib/testdata/libnickel.so.1.0.0.patch.all || {
    rm lnse
    exit 1
}
sha256sum lns lnse
icdiff <(xxd lnse) <(xxd lns)
rm lns lnse