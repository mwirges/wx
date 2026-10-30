#include <stdio.h>
#include <stdlib.h>

#include "libwxmobile.h"

/* Calls WxRunJSON once and prints the JSON. The process must not crash.
 * When WX_BRIDGE_SET_CACHE_DIR is set, call WxSetCacheDir first so the
 * Go side sees the override even though the variable was not in the
 * process environment at dyld/c-archive init (the iOS setenv failure). */
int main(int argc, char **argv) {
    char *out;
    const char *cache_dir;

    if (argc != 2) {
        fprintf(stderr, "usage: mobile-bridge-harness <args-json>\n");
        return 2;
    }

    cache_dir = getenv("WX_BRIDGE_SET_CACHE_DIR");
    if (cache_dir != NULL && cache_dir[0] != '\0') {
        if (WxSetCacheDir((char *)cache_dir) != 0) {
            fprintf(stderr, "WxSetCacheDir rejected %s\n", cache_dir);
            return 1;
        }
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
