/* patch32lsb.c -- Apply 'Patcher' style patches to a LSB executable.

   Version 0.10, created 5 Jan 2015 by Geoffrey Reynolds.
   This file is hereby placed into the public domain.

   Changes:
   0.1 -> 0.2  replace_int now only replaces 8-bit instead of 32-bit integers.
   0.2 -> 0.3  replace_string implemented.
   0.3 -> 0.4  encode_xor8 implemented.
   0.4 -> 0.5  -k switch leaves xor encoded strings unterminated, like kpg.exe.
               Don't use htole64() or getline(), so that mingw can compile it.
   0.5 -> 0.6  Accept \x in replace_string, e.g. \x00 encodes a null byte.
   0.6 -> 0.7  Ignore redundant utf-8 byte order mark inserted at the beginning
               of the patch file by some editors (e.g. Notepad, Wordpad).
   0.7 -> 0.8  Added base_address keyword.
               Recognise '\0' as a synonym for '\x00'.
   0.8 -> 0.9  Added find_base_address and find_xor8_mask functions.
   0.9 -> 0.10 Truncate addresses to 32 bits, to allow for negative offsets.
               Avoid using memmem() since mingw doesn't have it.
               Much faster one-pass implementation of find_xor8_mask.
               Include version number in --help.


   Limitations:
   replace_utf8chars, replace_zlib not implemented.
   Input file must be seekable.
   --revert option fails if patches modify the same data more than once.
   --revert option fails if find_base_address or find_xor8_mask are used.


   For patches of the following form:

     <Patch>
     ...
     replace_xor_D3, ADDRESS, `STRING1`, `STRING2`
     ...
     </Patch>

   use this form instead:

     <Patch>
     ...
     encode_xor8 = D3
     replace_string = ADDRESS, `STRING1`, `STRING2`
     ...
     </Patch>

   
   Compile: gcc -Wall -o patch32lsb patch32lsb.c
   Usage: patch32lsb [-k] [-r] [-p PATCH_FILE] -i INPUT_FILE [-o OUTPUT_FILE]
   Help: patch32lsb -h
*/

#include <ctype.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <getopt.h>

#define VERSION_MAJOR 0
#define VERSION_MINOR 10

/* Default patch file to use if not specified with -p on the command line.
 */
#define DEFAULT_PATCH_FILENAME "kpg.conf"

static const char *in_filename = NULL;
static const char *out_filename = NULL;
static const char *patch_filename = NULL;

static const char *program_name = "patch32lsb";

static const char *short_opts = "i:o:p:krh";
static const struct option long_opts[] =
  {
    {"input-file",     required_argument, 0, 'i'},
    {"output-file",    required_argument, 0, 'o'},
    {"patch-file",     required_argument, 0, 'p'},
    {"kpg",            no_argument,       0, 'k'},
    {"revert",         no_argument,       0, 'r'},
    {"help",           no_argument,       0, 'h'},
    {0, 0, 0, 0}
  };


static void help(void)
{
  printf("patch32lsb version %d.%d\n", VERSION_MAJOR, VERSION_MINOR);
  printf("Usage: %s [-r] [-p PATCH_FILE] -i INPUT_FILE [-o OUTPUT_FILE]\n",
	 program_name);
  printf("Apply/revert patches in PATCH_FILE against INPUT_FILE"
	 " to create OUTPUT_FILE.\n");
  printf("OPTIONS:\n");
  printf("-i --input-file FILE   Apply patches to input FILE\n");
  printf("-o --output-file FILE  Write patched output to FILE\n");
  printf("-p --patch-file FILE   Read patches from FILE (default '%s')\n",
	 DEFAULT_PATCH_FILENAME);
  printf("-k --kpg               Leave xor strings unterminated like kpg.exe\n");
  printf("-r --revert            Revert patch (some patches can't be reverted)\n");
  printf("-h --help              Display this helpful message\n");

  exit(EXIT_SUCCESS);
}

/* Reads a line from file, ignoring blank lines and comments. Returns a
   pointer to the first non-whitespace character in the line.
*/
#define MAX_LINE_LENGTH 4096
#define COMMENT_CHAR '#'
static int line_counter = 0;
static char line_buffer[MAX_LINE_LENGTH];
static char *read_line(FILE *file)
{
  char *ptr;

  while ((ptr = fgets(line_buffer,MAX_LINE_LENGTH,file)) != NULL)
  {
    line_counter++;
    while (isspace(*ptr))
      ptr++;
    if (*ptr != COMMENT_CHAR && *ptr != '\0')
      break;
  }

  return ptr;
}

static void line_error(const char *msg)
{
  fprintf(stderr, "%s: line %d in '%s': %s.\n",
	  program_name, line_counter, patch_filename, msg);
  exit(EXIT_FAILURE);
}

static void file_error(const char *file_name, const char *msg)
{
  fprintf(stderr, "%s: '%s': %s.\n", program_name, file_name, msg);
  exit(EXIT_FAILURE);
}

static void program_error(const char *msg)
{
  fprintf(stderr, "%s: %s.\n", program_name, msg);
  exit(EXIT_FAILURE);
}

static int parse_byte_list(const char *str, unsigned char *bytes)
{
  int len = 0;
  char *tail;
  long val;

  while (1) {
    val = strtol(str,&tail,16);
    if (tail == str)
      break;
    if (val < 0 || val > 255)
      return 0;
    bytes[len++] = val;
    str = tail;
  }

  return len;
}

static int unescape_line(char *line, char delim, char **tail)
{
  /* Replace \\, \n, \r, \t, \v \", \', \`, \0 with their unescaped byte values.
     Replace \xHH with the byte HH, where H is a hexdecimal digit.
     Stop processing at first unescaped occurance of delim (if any).
     Set tail to point to the first unprocessed character.
     Return the length of the new string (or the length of the substring up
     to the first occurance of delim) if successful, -1 if not. */

  char *p, *q;
  unsigned int x;

  p = q = line;
  do {
    if (*p == delim) {
      *tail = p + 1;
      return (int)(q - line);
    }
    else if (*p == '\\')
      switch (*++p) {
      case '\\':
	*q = '\\'; break;
      case 'n':
	*q = '\n'; break;
      case 'r':
	*q = '\r'; break;
      case 't':
	*q = '\t'; break;
      case 'v':
	*q = '\v'; break;
      case '0':
	*q = '\0'; break;
      case 'x':
	if (!isxdigit(p[1]) || !isxdigit(p[2]))
	  return -1;
	sscanf(p+1,"%2x",&x);
	*q = x;
	p += 2;
	break;
      case '\"':
      case '\'':
      case '`':
	*q  = *p;
	break;
      default:
	return -1;
      }
    else *q = *p;
    q++;
  } while (*p++ != '\0');

  *tail = p;
  return (int)(q - line - 1);
}

static
int memcmp_xor8(const void *e, const void *u, size_t len, unsigned char x)
{
  /* memcmp(e,u^x,len) */

  const unsigned char *p = e, *q = u;
  size_t i;

  for (i = 0; i < len && p[i] == (q[i]^x); i++)
    ;

  return (i < len)? (int)p[i]-(int)(q[i]^x) : 0;
}

static
void *memcpy_xor8(void *e, const void *u, size_t len, unsigned char x)
{
  /* memcpy(e,u^x,len) */

  unsigned char *p = e;
  const unsigned char *q = u;
  size_t i;

  for (i = 0; i < len; i++)
    p[i] = q[i]^x;

  return p;
}

static long find_unique_xor8(const void *haystack, size_t hlen,
			     const void *needle, size_t nlen, unsigned char x)
{
  /* Returns the position of the unique needle^x in haystack,
     or -1 if needle^x not found,
     or -2 if needle^x not unique. */

  const unsigned char *p, *q;
  const void *r;
  unsigned char n0;

  if (hlen == 0 || hlen < nlen)
    return -1;
  if (nlen == 0)
    return -2;

  n0 = *(const unsigned char *)needle;
  p = (const unsigned char *)haystack;
  q = p + hlen - nlen;

  while (p <= q && (*p != (n0^x) || memcmp_xor8(p,needle,nlen,x) != 0))
    p++;
  if (p > q)
    return -1;

  r = p++;
  while (p <= q && (*p != (n0^x) || memcmp_xor8(p,needle,nlen,x) != 0))
    p++;
  if (p <= q)
    return -2;

  return r - haystack;
}

static int find_xor8_mask(const void *haystack, size_t hlen,
			  const void *needle, size_t nlen)
{
  /* Returns the xor8 mask of the unique needle in haystack,
     or -1 if needle not found,
     or -2 if needle not unique. */

  const unsigned char *p, *q;
  int r;
  unsigned char n0, n1, x;

  if (hlen == 0 || hlen < nlen)
    return -1;
  if (nlen == 0)
    return -2;

  n0 = *(const unsigned char *)needle;
  p = (const unsigned char *)haystack;

  if (nlen == 1)
    return (hlen == 1)? *p^n0 : -2;

  n1 = *((const unsigned char *)needle+1);

  for (r = -1, q = p + hlen - nlen; p <= q; p++) {
    x = *p^n0;
    if (*(p+1) == (n1^x) && memcmp_xor8(p,needle,nlen,x) == 0) {
      if (r == -1)
	r = x;
      else
	return -2;
    }
  }

  return r;
}

/* Rearrange a double into little-endian byte order.
 */
static void set_htole_double(double *d)
{
  const double one = 1.0; /* 0x3FF0000000000000 */

  if (((const unsigned char *)&one)[0] == 0x3F) { /* big endian */
    unsigned char tmp, *b = (unsigned char *)d;
    tmp = b[0], b[0] = b[7], b[7] = tmp;
    tmp = b[1], b[1] = b[6], b[6] = tmp;
    tmp = b[2], b[2] = b[5], b[5] = tmp;
    tmp = b[3], b[3] = b[4], b[4] = tmp;
  }
  else if (((const unsigned char *)&one)[7] != 0x3F) /* not little endian */
    program_error("unknown floating point host format");
}


int main(int argc, char **argv)
{
  int opt_ind, opt_c;
  FILE *in_file = NULL;
  FILE *out_file = NULL;
  FILE *patch_file = NULL;
  unsigned char *data;
  long data_len;
  int kpg_compat = 0, revert = 0;

  program_name = argv[0];

  while ((opt_c = getopt_long(argc,argv,short_opts,long_opts,&opt_ind)) != -1)
    switch (opt_c)
      {
      case 'i':
        in_filename = optarg;
        break;
      case 'o':
        out_filename = optarg;
        break;
      case 'p':
        patch_filename = optarg;
        break;
      case 'k':
	kpg_compat = 1;
	break;
      case 'r':
	revert = 1;
	break;
      case 'h':
        help();
        break;
      default:
        return 1;
      }


  if (in_filename == NULL)
    program_error("no input filename supplied");
  else if ((in_file = fopen(in_filename,"rb")) == NULL)
    file_error(in_filename, "failed to open input file");

  if (fseek(in_file,0,SEEK_END))
    file_error(in_filename, "fseek failed");
  if ((data_len = ftell(in_file)) == -1)
    file_error(in_filename, "ftell failed");
  if ((data = malloc(data_len)) == NULL)
    file_error(in_filename,"failed to malloc enough space for input");
  if (fseek(in_file,0,SEEK_SET))
    file_error(in_filename, "fseek failed");
  if (fread(data,data_len,1,in_file) != 1)
    file_error(in_filename,"failed while reading input file");
  fclose(in_file);

  fprintf(stderr, "Read %ld bytes from input file '%s'\n",
	  data_len, in_filename);


  if (patch_filename == NULL)
    patch_filename = DEFAULT_PATCH_FILENAME;
  if ((patch_file = fopen(patch_filename,"r")) == NULL)
    file_error(patch_filename, "failed to open patch file");
  else {
    /* Check for possible unicode byte order mark in patch file */
    int ch;
    switch ((ch = fgetc(patch_file))) {
    case 0xEF: /* possible utf-8 mark, discard if found */
      if (fgetc(patch_file) != 0xBB || fgetc(patch_file) != 0xBF)
	file_error(patch_filename, "unknown character encoding");
      break;
    case 0xFE: /* possible utf-16 big-endian mark */
    case 0xFF: /* possible utf-16/32 little-endian mark */
    case 0x00: /* possible utf-32 big-endian mark */
      file_error(patch_filename, "unknown character encoding");
      break;
    default:
      ungetc(ch,patch_file);
    }
  }

  char *buf = NULL;
  char patch_name[80];
  int in_patch = 0, patch_enabled = 0, patch_started = 0;
  unsigned char xor8 = 0;
  unsigned long base_addr = 0;

  while ((buf = read_line(patch_file)) != NULL) {
    unsigned long addr;
    if (strncasecmp(buf,"<Patch>",strlen("<Patch>")) == 0) {
      if (in_patch++)
	line_error("missing </Patch>");
      patch_name[0] = '\0', patch_enabled = 0, patch_started = 0;
      xor8 = 0, base_addr = 0;
    }
    else if (strncasecmp(buf,"</Patch>",strlen("</Patch>")) == 0) {
      if (--in_patch)
	line_error("missing <Patch>");
      else if (patch_name[0] == '\0')
	line_error("missing patch_name");
      else if (patch_enabled == 0)
	fprintf(stderr,"Ignoring disabled patch `%s`\n",patch_name);
      else if (patch_started == 0)
	fprintf(stderr,"Ignoring empty patch `%s`\n",patch_name);
      else if (revert)
	fprintf(stderr,"Reverted patch `%s`\n",patch_name);
      else
	fprintf(stderr,"Applied patch `%s`\n",patch_name);
    }
    else if (strncasecmp(buf,"patch_name",strlen("patch_name")) == 0) {
      if (!in_patch)
	line_error("misplaced patch_name");
      else if (patch_name[0] != '\0')
	line_error("duplicate patch_name");
      else if (sscanf(buf,"%*[^=]=%*[^`]`%79[^`]`", patch_name) != 1)
	line_error("malformed patch_name");
    }
    else if (strncasecmp(buf,"patch_enable",strlen("patch_enable")) == 0) {
      char ch[8];
      if (!in_patch)
	line_error("misplaced patch_enable");
      else if (patch_enabled)
	line_error("duplicate patch_enable");
      else if (sscanf(buf, "%*[^=]=%*[^`]`%8[^`]`", ch) != 1)
	line_error("malformed patch_enable");
      else if (strcasecmp(ch,"yes") == 0)
	patch_enabled = 1;
      else if (strcasecmp(ch,"no") != 0)
	line_error("malformed argument to patch_enable");
    }
    else if (strncasecmp(buf,"encode_xor8",strlen("encode_xor8")) == 0) {
      if (!in_patch)
	line_error("misplaced encode_xor8");
      else if (patch_enabled) {
	unsigned int x;
	if (sscanf(buf, "%*[^=]=%x", &x) != 1)
	  line_error("malformed encode_xor8");
	else if (x > 255)
	  line_error("encode_xor8 value out of range 00-FF");
	xor8 = x;
      }
    }
    else if (strncasecmp(buf,"base_address",strlen("base_address")) == 0) {
      if (!in_patch)
	line_error("misplaced base_address");
      else if (patch_enabled) {
	if (sscanf(buf, "%*[^=]=%lx", &base_addr) != 1)
	  line_error("malformed base_address");
      }
    }
    else if (strncasecmp(buf,"find_base_address",strlen("find_base_address")) == 0) {
      char delim, *s, *t;
      int slen;
      long ret;
      if (!in_patch)
	line_error("misplaced find_base_address");
      else if (patch_enabled) {
	if (sscanf(buf, "%*[^=]=%*[^`\'\"]%c", &delim) != 1)
	  line_error("malformed find_base_address");
	if ((s = strchr(buf,delim)) == NULL)
	  line_error("malformed find_base_address string");
	s++;
	if ((slen = unescape_line(s,delim,&t)) < 0)
	  line_error("malformed find_base_address string");
	else if (slen == 0)
	  line_error("zero length find_base_address string");
	switch ((ret = find_unique_xor8(data,data_len,s,slen,xor8))) {
	  case -1: line_error("find_base_address string not found");
	  case -2: line_error("find_base_address string not unique");
	  default: base_addr = ret;
	}
        fprintf(stderr,"find_base_address: unique string at %.8lX\n",base_addr);
      }
    }
    else if (strncasecmp(buf,"find_xor8_mask",strlen("find_xor8_mask")) == 0) {
      char delim, *s, *t;
      int slen;
      int ret;
      if (!in_patch)
	line_error("misplaced find_xor8_mask");
      else if (patch_enabled) {
	if (sscanf(buf, "%*[^=]=%*[^`\'\"]%c", &delim) != 1)
	  line_error("malformed find_xor8_mask");
	if ((s = strchr(buf,delim)) == NULL)
	  line_error("malformed find_xor_mask string");
	s++;
	if ((slen = unescape_line(s,delim,&t)) < 0)
	  line_error("malformed find_xor8_mask string");
	else if (slen == 0)
	  line_error("zero length find_xor8_mask string");
	switch ((ret = find_xor8_mask(data,data_len,s,slen))) {
	  case -1: line_error("find_xor_mask string not found");
	  case -2: line_error("find_xor_mask string not unique");
	  default: xor8 = ret;
	}
	fprintf(stderr,"find_xor8_mask: unique string with mask %.2X\n",xor8);
      }
    }
    else if (strncasecmp(buf,"replace_bytes",strlen("replace_bytes")) == 0) {
      char b[2][96];
      unsigned char B[2][32];
      int len;
      if (!in_patch)
	line_error("misplaced replace_bytes");
      else if (patch_enabled) {
	if (sscanf(buf, "%*[^=]=%lx,%95[^,],%95[ a-zA-Z0-9]",
		   &addr, b[0], b[1]) != 3)
	  line_error("malformed replace_bytes");
	if ((len = parse_byte_list(b[0],B[0])) == 0)
	  line_error("malformed replace_bytes argument");
	if (len != parse_byte_list(b[1],B[1]))
	  line_error("malformed replace_bytes argument");
	addr = (addr + base_addr) & 0xFFFFFFFF;
	if (addr + len > data_len)
	  line_error("replace_bytes address beyond end of input file");
	else if (memcmp_xor8(data+addr,B[revert],len,xor8))
	  line_error("replace_bytes mismatched data");
	memcpy_xor8(data+addr,B[!revert],len,xor8);
	patch_started = 1;
      }
    }
    else if (strncasecmp(buf,"replace_float",strlen("replace_float")) == 0) {
      double D[2];
      if (!in_patch)
	line_error("misplaced replace_float");
      else if (patch_enabled) {
	if (sscanf(buf, "%*[^=]=%lx,%lf,%lf", &addr, &D[0], &D[1]) != 3)
	  line_error("malformed replace_float");
	addr = (addr + base_addr) & 0xFFFFFFFF;
	if (addr + sizeof(double) > data_len)
	  line_error("replace_float address beyond end of input file");
	set_htole_double(&D[0]);
	set_htole_double(&D[1]);
	if (memcmp_xor8(data+addr,&D[revert],sizeof(double),xor8))
	  line_error("replace_float mismatched data");
	memcpy_xor8(addr+data,&D[!revert],sizeof(double),xor8);
	patch_started = 1;
      }
    }
    else if (strncasecmp(buf,"replace_int",strlen("replace_int")) == 0) {
      int I[2];
      unsigned char U[2];
      if (!in_patch)
	line_error("misplaced replace_int");
      else if (patch_enabled) {
	if (sscanf(buf, "%*[^=]=%lx,%d,%d", &addr, &I[0], &I[1]) != 3)
	  line_error("malformed replace_int");
	addr = (addr + base_addr) & 0xFFFFFFFF;
	if (I[0] < 0 || I[0] > 255 || I[1] < 0 || I[1] > 255)
	  line_error("replace_int value out of range 0-255");
	if (addr + sizeof(unsigned char) > data_len)
	  line_error("replace_int address beyond end of input file");
	U[0] = I[0];
	U[1] = I[1];
	if (memcmp_xor8(data+addr,&U[revert],sizeof(unsigned char),xor8))
	  line_error("replace_int mismatched data");
	memcpy_xor8(addr+data,&U[!revert],sizeof(unsigned char),xor8);
	patch_started = 1;
      }
    }
    else if (strncasecmp(buf,"replace_string",strlen("replace_string")) == 0) {
      char delim, *s, *S[2];
      int L[2];
      if (!in_patch)
	line_error("misplaced replace_string");
      else if (patch_enabled) {
	if (sscanf(buf, "%*[^=]=%lx%*[^`\'\"]%c", &addr, &delim) != 2)
	  line_error("malformed replace_string address");
	addr = (addr + base_addr) & 0xFFFFFFFF;
	if ((S[0] = strchr(buf,delim)) == NULL)
	  line_error("malformed replace_string (original string)");
	S[0]++;
	if ((L[0] = unescape_line(S[0],delim,&s)) < 0)
	  line_error("malformed replace_string (original string)");
	else if (L[0] == 0)
	  line_error("zero length replace_string");
	if ((S[1] = strchr(s,delim)) == NULL)
	  line_error("malformed replace_string (replacement string)");
	S[1]++;
	if ((L[1] = unescape_line(S[1],delim,&s)) < 0)
	  line_error("malformed replace_string (replacement string)");
	if (L[1] > L[0])
	  line_error("replace_string replacement too long");
	else if (addr + L[!revert] + 1 > data_len)
	  line_error("replace_string address beyond end of input file");
	else if (memcmp_xor8(data+addr, S[revert], L[revert], xor8))
	  line_error("replace_string mismatched data");
	memcpy_xor8(data+addr, S[!revert], L[!revert], xor8);
	if (L[!revert] < L[revert]) /* Replacement is shorter than original. */
	  /* Append a zero  byte to the end of the string. Note that Patcher 10
	     doesn't seem to do this when using its replace_xor_ methods, so
	     this will cause a one byte difference when compared to Patcher 10
	     output if encode_xor8 is active and the -k switch is not used. */
	  if (kpg_compat == 0 || xor8 == 0)
	    memcpy_xor8(data + addr + L[!revert], "", 1, xor8);
	patch_started = 1;
      }
    }
    else if (!strncasecmp(buf,"replace_utf8chars",strlen("replace_utf8chars"))){
      if (!in_patch)
	line_error("misplaced replace_utf8chars");
      else if (patch_enabled)
	line_error("replace_utf8chars not yet implemented"); /* TODO */
    }
    else if (strncasecmp(buf,"replace_zlib",strlen("replace_zlib")) == 0) {
      if (!in_patch)
	line_error("misplaced replace_zlib");
      else if (patch_enabled)
	line_error("replace_zlib not yet implemented"); /* TODO */
    }
    else if (strncasecmp(buf,"replace_xor",strlen("replace_xor")) == 0) {
      if (!in_patch)
	line_error("misplaced replace_xor");
      else if (patch_enabled)
	line_error("replace_xor not implemented, use encode_xor8 + replace_string instead");
    }
    else if (strncasecmp(buf,"replace_",strlen("replace_")) == 0) {
      if (!in_patch)
	line_error("misplaced replace_");
      else if (patch_enabled)
	line_error("unknown replace_ method");
    }
    else if (strncasecmp(buf,"encode_",strlen("encode_")) == 0) {
      if (!in_patch)
	line_error("misplaced encode_");
      else if (patch_enabled)
	line_error("unknown encode_ method");
    }
    else
      line_error("cannot parse line");
  } /* while */

  if (!feof(patch_file))
    line_error("error while reading patch file");

  if (out_filename == NULL) {
    fprintf(stderr, "%s: no output file name supplied, no output written.\n",
	    program_name);
    exit(EXIT_SUCCESS);
  }
  if ((out_file = fopen(out_filename,"wb")) == NULL)
    file_error(out_filename, "failed to open output file");
  if (fwrite(data,data_len,1,out_file) != 1)
    file_error(out_filename, "failed while writing output file");
  fclose(out_file);

  fprintf(stderr, "Wrote %ld bytes to output file '%s'\n",
	  data_len, out_filename);


  exit(EXIT_SUCCESS);
}
