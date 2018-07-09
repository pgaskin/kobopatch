
typedef unsigned int uint;

/* simulate the Go return type ([]byte, error) so that the Go function
 * can allocate the right amt of memory to copy the str if required or
 * report errors directly from the C lib.
 */
typedef struct {
    char *str;
    uint len;
    char *err;
} ByteArray;

ByteArray c_decompress(char *input, uint length);
ByteArray c_compress(char *input, uint length);
ByteArray c_compress2(char *input, uint length);

