/* fastzlib.c
 *
 * This module is an implementation of a non-streaming "compress" and
 * "decompress", Similar to and based on Python 2.7's "zlibmodule.c".
 *
 * Hopefully this will perform very similarly to Python's zlib module,
 * as both vitess' cgzip and compress/zlib perform between very and
 * quite poorly in comparisson on small payloads.
 */
#include "_cgo_export.h"
#include <stdio.h>
#include <stdlib.h>
#include <zlib.h>
#include "fastzlib.h"

// 16k chunk size used by Python zlibmodule.c and http://www.zlib.net/zpipe.c
#define DEFAULTALLOC (16*1024)
#define ERRMSG_MAX (1024*2)

// Create an error message for various types of errors.
char *zlib_error(z_stream zst, int err, char *msg) {
    char *errmsg = NULL;

    if (err == Z_VERSION_ERROR) {
        errmsg = "library version mismatch";
    }
    if (errmsg == Z_NULL) {
        errmsg = zst.msg;
    }
    if (errmsg == Z_NULL) {
        switch(err) {
        case Z_BUF_ERROR:
            errmsg = "incomplete or truncated stream";
            break;
        case Z_STREAM_ERROR:
            errmsg = "inconsistent stream state";
            break;
        case Z_DATA_ERROR:
            errmsg = "invalid input data";
            break;
        }
    }
    if (errmsg == Z_NULL) {
        errmsg = (char *)malloc(ERRMSG_MAX);
        snprintf(errmsg, ERRMSG_MAX, "Error: %d %s", err, msg);
    } else {
        char *orig = errmsg;
        errmsg = (char *)malloc(ERRMSG_MAX);
        snprintf(errmsg, ERRMSG_MAX, "Error %d %s: %s", err, msg, orig);
    }

    return errmsg;
}

// Entirely decompress input, returning a ByteArray
// return value is a ByteArray whose err/str, if set, must be freed.
ByteArray c_decompress(char *input, uint length) {
    ByteArray ret;
    int err, wsize=MAX_WBITS;
    z_stream zst;

    ret.len = DEFAULTALLOC;
    ret.str = (char *)malloc(ret.len);

    zst.avail_in = length;
    zst.avail_out = DEFAULTALLOC;
    zst.zalloc = (alloc_func)Z_NULL;
    zst.zfree = (free_func)Z_NULL;
    zst.next_out = (Byte *)ret.str;
    zst.next_in = (Byte *)input;

    err = inflateInit2(&zst, wsize);

    switch(err) {
    case(Z_OK):
        break;
    case(Z_MEM_ERROR):
        // something here might not even be possible..
        goto error;
    default:
        inflateEnd(&zst);
        goto error;
    }

    do {
        err = inflate(&zst, Z_FINISH);
        switch(err) {
        case(Z_STREAM_END):
            break;
        case(Z_BUF_ERROR):
            /*
             * If there is at least 1 byte of room according to zst.avail_out
             * and we get this error, assume that it means zlib cannot
             * process the inflate call() due to an error in the data.
             */
            if (zst.avail_out > 0) {
                inflateEnd(&zst);
                goto error;
            }
            /* fall through */
        case(Z_OK):
            /* need more memory, double size of return string each time. */
            ret.str = (char *)realloc(ret.str, ret.len <<1);
            if (ret.str == NULL) {
                inflateEnd(&zst);
                goto error;
            }

            zst.next_out = (unsigned char *)(ret.str + ret.len);
            zst.avail_out = ret.len;
            ret.len = ret.len << 1;
            break;
        default:
            inflateEnd(&zst);
            goto error;
        }
    } while (err != Z_STREAM_END);

    err = inflateEnd(&zst);
    if (err != Z_OK) {
        goto error;
    }

    /* success!  return everything */
    ret.len = zst.total_out;
    ret.err = NULL;
    return ret;

error:
    if (ret.str != NULL) free(ret.str);
    ret.err = zlib_error(zst, err, "while decompressing data");
    return ret;
}

// entirely compress input, returning a compressed  ByteArray
// return value is a ByteArray whose err/str, if set, must be freed.
ByteArray c_compress(char *input, uint length) {
    ByteArray ret;
    // FIXME: allow this to be tunable?
    int err, level=Z_DEFAULT_COMPRESSION;
    z_stream zst;

    // allocate the maximum possible length of the output up front
    // this calculation comes from python and is almost certainly safe
    zst.avail_out = length + length/1000 + 12 + 1;
    ret.str = (char *)malloc(zst.avail_out);

    if (ret.str == NULL) {
        ret.err = zlib_error(zst, Z_MEM_ERROR, "Out of memory while compressing data.");
        goto error;
    }

    zst.zalloc = (alloc_func)Z_NULL;
    zst.zfree = (free_func)Z_NULL;
    zst.opaque = Z_NULL;
    zst.next_out = (Byte *)ret.str;
    zst.next_in = (Byte *)input;
    zst.avail_in = length;

    err = deflateInit(&zst, level);

    switch(err) {
    case(Z_OK):
        break;
    case(Z_MEM_ERROR):
        ret.err = zlib_error(zst, err, "Out of memory while compressing data");
        goto error;
    // XXX: this isn't currently possible but will be in future
    case(Z_STREAM_ERROR):
        ret.err = zlib_error(zst, err, "Bad compression level");
        goto error;
    default:
        deflateEnd(&zst);
        ret.err = zlib_error(zst, err, "while compressing data");
        goto error;
    }

    err = deflate(&zst, Z_FINISH);

    if (err != Z_STREAM_END) {
        ret.err = zlib_error(zst, err, "while compressing data");
        deflateEnd(&zst);
        goto error;
    }

    err = deflateEnd(&zst);

    if (err != Z_OK) {
        ret.err = zlib_error(zst, err, "while finishing compression");
        goto error;
    } else {
        // XXX: we've allocated the maximum possible size (worst case) for zlib
        // compression up top, but it's likely that zst.total_out (the end size
        // of the buffer) is much smaller;  Python actually performs a copy here
        // so that the old, larger buffer can be freed immediately.
        //
        // This is a good strategy for a lot of reasons.  We could use realloc
        // to shrink the amount of space allocated to our buffer pointer, but
        // the excess memory is not actually freed;  instead it is made available
        // to future calls to malloc/calloc.
        //
        // Since the common case, Compress, will do this in Go and then free
        // immediately, it doesn't make sense to actually do this copy here, but
        // this might open up UnsafeCompress to weird allocation slowdowns if it's
        // doing lots of compression and decompression.
        ret.str = (char *)realloc(ret.str, zst.total_out);
        ret.len = zst.total_out;
        ret.err = NULL;
    }
    
    return ret;

error:
    if (ret.str != NULL) free(ret.str);
    return ret;
}


// entirely compress input, returning a compressed ByteArray
// return value is a ByteArray whose err/str, if set, must be freed.
// This version of compress uses the defaultbuffer + doubling growth
// like the decompress version does.
ByteArray c_compress2(char *input, uint length) {
    ByteArray ret;
    // FIXME: allow this to be tunable?
    int err, grow, level=Z_DEFAULT_COMPRESSION;
    z_stream zst;

    // allocation strategy:
    // we want to use a single worst-case allocation for very small inputs,
    // DEFAULTALLOC for medium sized inputs, and length/8 for large inputs.

    if (length < DEFAULTALLOC) {
        //printf("Using worst-case alloc of %d\n", length + length/1000 + 13);
        ret.len = zst.avail_out = length + length/1000 + 12 + 1;
    } else if (length/8 > DEFAULTALLOC) {
        //printf("Using quarter buffer alloc of %d\n", length/4);
        ret.len = zst.avail_out = length/8;
    } else {
        //printf("Using DEFAULTALLOC of %d\n", DEFAULTALLOC);
        ret.len = zst.avail_out = DEFAULTALLOC;
    }
    ret.str = (char *)malloc(zst.avail_out);

    if (ret.str == NULL) {
        ret.err = zlib_error(zst, Z_MEM_ERROR, "Out of memory while compressing data.");
        goto error;
    }

    zst.zalloc = (alloc_func)Z_NULL;
    zst.zfree = (free_func)Z_NULL;
    zst.opaque = Z_NULL;
    zst.next_out = (Byte *)ret.str;
    zst.next_in = (Byte *)input;
    zst.avail_in = length;

    err = deflateInit2(&zst, level, Z_DEFLATED,
                        15, // 16 makes it a gzip file, 15 is default
                        8, Z_DEFAULT_STRATEGY); // default values

    switch(err) {
    case(Z_OK):
        break;
    case(Z_MEM_ERROR):
        ret.err = zlib_error(zst, err, "Out of memory while compressing data");
        goto error;
    // XXX: this isn't currently possible but will be in future
    case(Z_STREAM_ERROR):
        ret.err = zlib_error(zst, err, "Bad compression level");
        goto error;
    default:
        deflateEnd(&zst);
        ret.err = zlib_error(zst, err, "while compressing data");
        goto error;
    }

    do {
        err = deflate(&zst, Z_FINISH);

        switch(err) {
        case(Z_STREAM_END):
            break;
        case(Z_BUF_ERROR):
            /*
             * If there is at least 1 byte of room according to zst.avail_out
             * and we get this error, assume that it means zlib cannot
             * process the inflate call() due to an error in the data.
             */
            if (zst.avail_out > 0) {
                deflateEnd(&zst);
                goto error;
            }
            /* fall through */
        case(Z_OK):
            /* we need more memory;  increase by 1/8th the original buffer each time */
            grow = ret.len + length/8;
            ret.str = (char *)realloc(ret.str, grow);

            if (ret.str == NULL) {
                deflateEnd(&zst);
                goto error;
            }

            zst.next_out = (unsigned char *)(ret.str + ret.len);
            zst.avail_out = grow - ret.len;
            ret.len = grow;
            break;
        default:
            deflateEnd(&zst);
            goto error;
        }
    } while (err != Z_STREAM_END);

    err = deflateEnd(&zst);

    if (err != Z_OK) {
        goto error;
    }

    /* success!  return everything */
    ret.str = (char *)realloc(ret.str, zst.total_out);
    ret.len = zst.total_out;
    ret.err = NULL;
    return ret;

error:
    if (ret.str != NULL) free(ret.str);
    if (ret.err == NULL) ret.err = zlib_error(zst, err, "while compressing data");
    return ret;
}
