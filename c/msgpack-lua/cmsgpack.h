// c msgpack

#ifndef __CMSGPACK_H__
#define __CMSGPACK_H__

#include <math.h>
#include <stdlib.h>
#include <stdint.h>
#include <string.h>
#include <assert.h>

#include "lua.h"
#include "lauxlib.h"

#ifdef __cplusplus
extern "C" {
#endif

struct wrap_stack_t {
	uint16_t pos;
	uint16_t len;
};

typedef struct mp_buf {
    unsigned char *b;
    size_t len, free;
    uint32_t offset; // for unpack
    struct wrap_stack_t *stacks;
    uint8_t wrap_stack_n; //最大层数
	uint8_t wrap_stack_p; //当前层数
} mp_buf;

#define mp_buf_get_write_pos(buf) buf->len
#define mp_encode_array16(b, n) b[1] = (n & 0xff00) >> 8; b[2] = n & 0xff

mp_buf *mp_buf_new(lua_State *L);
void mp_buf_free(lua_State *L, mp_buf *buf);
void mp_buf_reset(lua_State *L, mp_buf *buf);

// encode
void mp_buf_append(lua_State *L, mp_buf *buf, const unsigned char *s, size_t len);
void mp_encode_lua_type(lua_State *L, mp_buf *buf, int level);
void mp_encode_array(lua_State *L, mp_buf *buf, int64_t n);
int mp_pack(lua_State *L, mp_buf *mp, int beginIdx, int nargs);

// decode
int mp_decode_is_string(mp_buf *mp);
void mp_decode_to_array_len(mp_buf *mp, int *l);
int mp_unpack_full(lua_State *L, mp_buf *mp, int limit);

#ifdef __cplusplus
}
#endif

#endif