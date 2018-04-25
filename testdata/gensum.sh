#!/bin/bash

cd "$(dirname "$0")"
./patch32lsb -p libnickel.so.1.0.0.patch -i libnickel.so.1.0.0 -o libnickel.so.tmp
sha256sum libnickel.so.tmp
rm libnickel.so.tmp
