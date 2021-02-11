package dist


/*
#cgo darwin CFLAGS: -v '-DMDBX_CONFIG_H="config.h"' -std=gnu++2a -O2 -g -Wall -Werror -Wextra -Wpedantic -fPIC -fvisibility=hidden -std=gnu11 -Wno-error=attributes
#cgo darwin LDFLAGS: -lpthread

//#cgo linux  CFLAGS: -DMDBX_CONFIG_H="config_linux.h" -std=gnu++2a -O2 -g -Wall -Werror -Wextra -Wpedantic -fPIC -fvisibility=hidden -std=gnu11 -pthread -Wno-error=attributes
//#cgo linux  LDLAGS: -Wl,--gc-sections,-z,relro,-O1  -lrt  -ffunction-sections
//
//#cgo win  CFLAGS: -DMDBX_CONFIG_H="config_win.h" -std=gnu++2a -O2 -g -Wall -Werror -Wextra -Wpedantic -fPIC -fvisibility=hidden -std=gnu11 -pthread -Wno-error=attributes
*/
import "C"
