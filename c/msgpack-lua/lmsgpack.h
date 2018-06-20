// lmsgpack

#ifndef __LMSGPACK_H__
#define __LMSGPACK_H__

#include "cmsgpack.h"

int luaopen_msgpack(lua_State *L);
void lmsgpack_pushpacket(lua_State *L, unsigned char * packet, size_t sz);
const unsigned char * lmsgpack_check_buffer(lua_State *L, int idx, size_t * sz);
unsigned char * lmsgpack_check_buffer2(lua_State *L, int idx, size_t * sz);

#endif