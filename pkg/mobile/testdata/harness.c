#include <stdio.h>
#include <stdlib.h>

#include "libwxmobile.h"

/* Calls WxRunJSON once and prints the JSON. The process must not crash. */
int main(int argc, char **argv) {
    char *out;

    if (argc != 2) {
        fprintf(stderr, "usage: mobile-bridge-harness <args-json>\n");
        return 2;
    }

    WxFreeString(NULL);

    out = WxRunJSON(argv[1]);
    if (out == NULL) {
        fprintf(stderr, "WxRunJSON returned NULL\n");
        return 1;
    }
    fputs(out, stdout);
    WxFreeString(out);
    return 0;
}
