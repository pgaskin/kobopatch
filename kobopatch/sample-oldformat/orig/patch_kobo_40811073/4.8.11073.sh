#!/bin/bash

FIRMWARE_VERSION=4.8.11073

SOURCE_DIR=${FIRMWARE_VERSION}_source
TARGET_DIR=${FIRMWARE_VERSION}_target

KOBO_FIRMWARE=$SOURCE_DIR/kobo-update-$FIRMWARE_VERSION.zip

PATCH32LSB_SRC=tools/patch32lsb.c
PATCH32LSB_BIN=tools/patch32lsb

set -e

# This bit only works in Bash
cd "$(dirname "${BASH_SOURCE[0]}")"
exec >  >(tee -ia logfile.txt)
exec 2> >(tee -ia logfile.txt >&2)

FILES_TO_PATCH=""
for F in $SOURCE_DIR/*.patch; do
    FILES_TO_PATCH="$FILES_TO_PATCH ./usr/local/Kobo/`basename $F .patch`";
done

case `uname -s` in
    Darwin)
	PATCH32LSB_BIN=tools/patch32lsb-Darwin
	;;
    Linux)
	case `uname -m` in
	    i?86)
		PATCH32LSB_BIN=tools/patch32lsb-i386-Linux
		;;
	    x86_64)
		PATCH32LSB_BIN=tools/patch32lsb-x86_64-Linux
		;;
	    armv7l)
		PATCH32LSB_BIN=tools/patch32lsb-ARM-linux
		;;
	esac
	;;
esac

SCRATCH=`mktemp -d -t patch32lsb_XXXXXXXX`
echo "Created scratch directory $SCRATCH"
trap 'echo "Cleaning up scratch directory $SCRATCH"; rm -r $SCRATCH' EXIT

OLD=$SCRATCH/original; mkdir $OLD
NEW=$SCRATCH/patched; mkdir $NEW

echo "Checking $KOBO_FIRMWARE ..."
unzip -t $KOBO_FIRMWARE KoboRoot.tgz

echo "Extracting files to patch from $KOBO_FIRMWARE ..."
unzip -p $KOBO_FIRMWARE KoboRoot.tgz | tar xvz --directory=$OLD $FILES_TO_PATCH

for F in $FILES_TO_PATCH; do
    echo "Patching $F ..."
    mkdir -p `dirname $NEW/$F`;
    $PATCH32LSB_BIN -p $SOURCE_DIR/`basename $F`.patch -i $OLD/$F -o $NEW/$F;
    chmod 0755 $NEW/$F;
done

echo "Creating KoboRoot.tgz ..."
rm -rf $TARGET_DIR
mkdir -p $TARGET_DIR
tar cvzf $TARGET_DIR/KoboRoot.tgz --directory=$NEW $FILES_TO_PATCH
