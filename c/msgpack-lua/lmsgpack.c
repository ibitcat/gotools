// gcc -c -g -Wall -I/usr/include/lua5.1 -fPIC cmsgpack.c
// gcc -c -g -Wall -I/usr/include/lua5.1 -fPIC lmsgpack.c
// gcc -Wall -fPIC -W -shared -o msgpack.so *.o

#include "lmsgpack.h"

#define	MP_LUA_LIB_NAME "msgpack"
#define MSGPACKET_META_NAME "msgpacket_mate"
#define MSGPACKET_PACKET_NAME "msgPacket"

int lua_ismsgpack(lua_State *L, int idx) {
	if( lua_getmetatable(L, idx) == 0 ) {
		return 0;
	}

	lua_pushstring(L, "__name");
	lua_rawget(L, -2);

	char* name = (char*)lua_tostring(L, -1);
	if( !name || strcmp(name, MSGPACKET_PACKET_NAME) ) {
		lua_pop(L, 1);
		return 0;
	}

	lua_pop(L, 2);
	return 1;
}

mp_buf* lua_tompbuf(lua_State *L, int idx) {
	mp_buf *pkt = NULL;
	if( lua_ismsgpack(L, idx) == 0 ) {
		return NULL;
	}

	mp_buf **ud = (mp_buf**)lua_touserdata(L, idx);
	if (!ud) return NULL;

	pkt = *ud;
	return pkt;
}

void lmsgpack_pushpacket(lua_State *L, unsigned char *buf, size_t sz) {
	mp_buf** ud = (mp_buf**)lua_newuserdata(L, sizeof(mp_buf*));
	mp_buf* pkt = mp_buf_new(L);
	pkt->b = buf;
    pkt->len = sz;
	*ud = pkt;

	// set metatable
	luaL_getmetatable(L, MSGPACKET_META_NAME);
	lua_setmetatable(L, -2);
}

const unsigned char * lmsgpack_check_buffer(lua_State *L, int idx, size_t * sz) {
	struct mp_buf * mp = lua_tompbuf(L, idx);
	if (!mp) {
		*sz = 0;
		return NULL;
	}

	if (mp->stacks && mp->wrap_stack_p!=0){
		printf("maybe msgpack wrap error\n");
	}

	*sz = mp->len;
    return mp->b;
}

unsigned char * lmsgpack_check_buffer2(lua_State *L, int idx, size_t * sz) {
	struct mp_buf * mp = lua_tompbuf(L, idx);
	if (!mp) {
		*sz = 0;
		return NULL;
	}

	*sz = mp->len;
	unsigned char * b = mp->b;
	if (mp->stacks) {
		free(mp->stacks);
	}
    memset(mp, 0, sizeof(*mp));
    return b;
}

static void lua_msgpack_error(lua_State *L, const char* fmt, ...) {
 	char text[128] = "[MsgPack error]: ";

 	va_list args;
 	va_start(args, fmt);
 	vsprintf(text + strlen(text), fmt, args);
 	va_end(args);

 	lua_pushstring(L, text);
 	lua_error(L);
}

static int lua_msgpacket_new(lua_State *L) {
	mp_buf *pkt = mp_buf_new(L);
	if (!pkt) {
		lua_msgpack_error(L,"Create Pack needs a packet buf.");
	}

	mp_buf **ud = (mp_buf**)lua_newuserdata(L, sizeof(mp_buf*));
	*ud = pkt;
	luaL_getmetatable(L, MSGPACKET_META_NAME);
	lua_setmetatable(L, -2);
	return 1;
}

static int lua_msgpacket_gc(lua_State *L){
	mp_buf** c = (mp_buf**)luaL_checkudata(L, 1, MSGPACKET_META_NAME);
	luaL_argcheck(L, c != NULL && *c != NULL, 1, "Packet free needs a packet.");
	if (*c){
		mp_buf *pkt = *c;
		mp_buf_free(L, pkt); // free
		*c = NULL;
	}
	return 0;
}

static int lua_msgpack_wrap_pre(lua_State *L) {
	mp_buf* pkt = lua_tompbuf(L,1);
	if (!pkt) {
		lua_msgpack_error(L,"Pack table needs a packet.");
	}

	int pos = mp_buf_get_write_pos(pkt);
	if (pkt->wrap_stack_p >= pkt->wrap_stack_n) {
		uint32_t nn = pkt->wrap_stack_n + 2;
		if (nn > 0xffff) {
			lua_msgpack_error(L, "Pack wrap pre expect deep < 0xffff");
		}
		struct wrap_stack_t *stacks = (struct wrap_stack_t *)realloc(pkt->stacks, nn*sizeof(struct wrap_stack_t));
		if (NULL == stacks) {
			lua_msgpack_error(L, "Pack wrap pre realloc wrap_stack fail");
		}
		pkt->stacks = stacks;
		pkt->wrap_stack_n = nn;
	}

	if (pkt->wrap_stack_p > 0) {
		uint8_t preStack = pkt->wrap_stack_p - 1;
		if (pkt->stacks[preStack].len >= 0xffff) {
			lua_msgpack_error(L,"Pre pack wrap array max!!!");
		}

		// 上一层个数+1
		++pkt->stacks[preStack].len;
	}
	uint16_t p = pkt->wrap_stack_p++;
	pkt->stacks[p].pos = pos;
	pkt->stacks[p].len = 0;
	mp_encode_array(L, pkt, 0xffff);//假设不超过0xff
	return 0;
}

static int lua_msgpack_wrap_end(lua_State *L) {
	mp_buf* pkt = lua_tompbuf(L,1);
	if (!pkt) {
		lua_msgpack_error(L,"Pack table needs a packet.");
	}

	if (pkt->wrap_stack_p == 0) {
		lua_msgpack_error(L,"lua_msgpack_wrap_end exepect wrap_pre first");
	}

	// 当前层元素总数量
	const uint16_t p = pkt->wrap_stack_p - 1;
	int wpos = pkt->stacks[p].pos;
	int len = pkt->stacks[p].len;
	int cpos = mp_buf_get_write_pos(pkt);
	if (wpos > cpos) {
		lua_msgpack_error(L,"lua_msgpack_wrap_end exepect wpos < cpos.");
	}

	if (len < 0) {
		lua_msgpack_error(L,"lua_msgpack_wrap_end exepect len >=0.");
	}

	if (len > 0xffff) {
		lua_msgpack_error(L,"lua_msgpack_wrap_end exepect len <= 0xffff.");
	}

	unsigned char* b = pkt->b + wpos;
	if (b[0] != 0xdc) {
		lua_msgpack_error(L,"lua_msgpack_wrap_end exepect wpos was mp_array16.");
	}

	mp_encode_array16(b, len);
	pkt->wrap_stack_p = p;
	return 0;
}

static int lua_msgpack_wrap(lua_State *L) {
	int nargs = lua_gettop(L);
    if (nargs == 0){
        return luaL_argerror(L, 0, "MessagePack wrap needs input.");
	}

	mp_buf *pkt = lua_tompbuf(L,1);
	if (!pkt) {
		lua_msgpack_error(L,"Packs needs a msgpack.");
	}

    mp_encode_array(L, pkt, nargs-1);
	mp_pack(L, pkt, 2, nargs);

    // stack
    if (pkt->wrap_stack_p > 0) {
    	uint8_t curIdx = pkt->wrap_stack_p - 1;
		if (pkt->stacks[curIdx].len >= 0xffff) {
			lua_msgpack_error(L,"Pre pack wrap array max!!!");
		}

		// 当前层元素数量+1
		++pkt->stacks[curIdx].len;
	}
    return 0;
}

static int lua_msgpack_unwrap(lua_State *L) {
	mp_buf* pkt = lua_tompbuf(L,1);
	if (!pkt) {
		lua_msgpack_error(L,"Unwrap needs a msgpack.");
	}

	int l = -1;
	mp_decode_to_array_len(pkt, &l);
	if (l<0){
		lua_msgpack_error(L, "mp_lua_unwrap expect array type first");
		return 0;
	}else{
		return mp_unpack_full(L, pkt, l);
	}
}

static int lua_msgpack_unwrap_header(lua_State *L){
	mp_buf* pkt = lua_tompbuf(L,1);
	if (!pkt) {
		lua_msgpack_error(L,"Unwrap Header needs a msgpack.");
	}
	int l = -1;
	mp_decode_to_array_len(pkt, &l);
	if (l>=0){
		lua_pushnumber(L, l);
		return 1;
	}else{
		lua_msgpack_error(L, "mp_lua_unwrap expect array type first");
		return 0;
	}
}

static int lua_msgpack_packs(lua_State *L) {
	int nargs = lua_gettop(L);
	if (nargs == 0){
    	return luaL_argerror(L, 0, "MessagePack pack needs input.");
    }

	mp_buf *pkt = lua_tompbuf(L,1);
	if (!pkt) {
		lua_msgpack_error(L,"Packs needs a msgpack.");
	}

    int amt = mp_pack(L, pkt, 2, nargs);
	if (pkt->wrap_stack_p > 0) {
		uint8_t curIdx = pkt->wrap_stack_p - 1;
		if (pkt->stacks[curIdx].len >= 0xffff) {
			lua_msgpack_error(L,"Pre pack wrap array max!!!");
		}

		// 当前层元素数量+amt
		pkt->stacks[curIdx].len += amt;
	}
    return 0;
}

static int lua_msgpack_unpacks(lua_State *L){
	mp_buf *pkt = lua_tompbuf(L,1);
	if (!pkt) {
		lua_msgpack_error(L,"Unpacks needs a msgpack.");
	}

    int limit = luaL_optinteger(L, 2, 0);
	return mp_unpack_full(L, pkt, limit);
}

static int lua_msgpack_pack_subpkt(lua_State *L){
	mp_buf *pkt = lua_tompbuf(L,1);
	if (!pkt) {
		lua_msgpack_error(L,"Pack sub packet needs a msgpack.");
	}

	mp_buf* subPkt = lua_tompbuf(L,2);
	if (!subPkt) {
		lua_msgpack_error(L, "Sub packet needs a msgpack.");
	}

	int isAppend = luaL_optinteger(L, 3, 0); //是否是追加模式
	int amt = 0;
	if (isAppend!=0){
		if (pkt->stacks == NULL){
			lua_msgpack_error(L, "Append packet need wrap pre.");
			return 0;
		}
		mp_buf_append(L, pkt, (const unsigned char*)subPkt->b, subPkt->len);
		amt = 1;
	}else{
		lua_pushlstring(L, (const char*)subPkt->b, subPkt->len);
		lua_replace(L, 2);
		amt = mp_pack(L, pkt, 2, 2);
	}

	// inc stack
	if (pkt->wrap_stack_p > 0) {
		uint8_t curIdx = pkt->wrap_stack_p - 1;
		if (pkt->stacks[curIdx].len >= 0xffff) {
			lua_msgpack_error(L,"Pre pack wrap array max!!!");
		}
		// 当前层元素数量+amt
		pkt->stacks[curIdx].len += amt;
	}
	return 0;
}

static int lua_msgpack_unpack_subpkt(lua_State *L){
	mp_buf *pkt = lua_tompbuf(L,1);
	if (!pkt) {
		lua_msgpack_error(L,"UnPack sub packet needs a msgpack.");
	}

	mp_buf* subPkt = lua_tompbuf(L,2);
	if (!subPkt) {
		lua_msgpack_error(L, "Sub packet needs a msgpack.");
	}

	if (mp_decode_is_string(pkt)!=0){
		lua_msgpack_error(L, "Unpack sub packet bad data format.");
		return 0;
	}

	int amt = mp_unpack_full(L, pkt, 1);
	if(amt==1){
		size_t len;
		const char *s = lua_tolstring(L, -1, &len);
		mp_buf_reset(L, subPkt);
		mp_buf_append(L, subPkt, (const unsigned char*)s, len);
		lua_pop(L, 1);
	}else{
		lua_msgpack_error(L,"UnPack sub packet error.");
	}
	return 0;
}

static int lua_msgpack_reset(lua_State *L) {
	mp_buf* pkt = lua_tompbuf(L,1);
	if (!pkt) {
		lua_msgpack_error(L,"MessagePack reset needs a msgpack.");
	}
	mp_buf_reset(L, pkt);
	return 0;
}

static int lua_msgpack_reread(lua_State *L){
	mp_buf* pkt = lua_tompbuf(L,1);
	if (!pkt) {
		lua_msgpack_error(L,"MessagePack reread needs a msgpack.");
		return 0;
	}

 	int offset = luaL_optinteger(L, 2, 0);
	if (offset<0) {
		lua_msgpack_error(L,"MessagePack reread offset need >=0.");
		return 0;
	}
	pkt->offset = offset;
	return 0;
}

static void lua_reg_msgpacket_meta(lua_State *L) {
	luaL_newmetatable(L, MSGPACKET_META_NAME);

	lua_pushstring(L, "__gc");
	lua_pushcfunction(L, lua_msgpacket_gc);
	lua_rawset(L, -3);

	lua_pushstring(L, "__name");
	lua_pushstring(L, MSGPACKET_PACKET_NAME);
	lua_rawset(L, -3);

	lua_pushvalue(L, -1);
	lua_setfield(L, -2, "__index");
	#define __inject(a, b) lua_pushstring(L, a); lua_pushcfunction(L, b); lua_rawset(L, -3);
	__inject("packs", lua_msgpack_packs)
	__inject("unpacks", lua_msgpack_unpacks)
	__inject("wrap", lua_msgpack_wrap)
	__inject("unwrap",lua_msgpack_unwrap)
	__inject("wrap_pre", lua_msgpack_wrap_pre)
	__inject("wrap_end", lua_msgpack_wrap_end)
	__inject("unwrap_header", lua_msgpack_unwrap_header)
	__inject("packPacket", lua_msgpack_pack_subpkt)
	__inject("unpackPacket", lua_msgpack_unpack_subpkt)
	__inject("reset", lua_msgpack_reset)
	__inject("reread", lua_msgpack_reread)
	#undef __inject

	lua_pop(L, 1);
}

static const struct luaL_reg thislib[] = {
	{"newPacket", lua_msgpacket_new},
    {NULL, NULL}
};

int luaopen_msgpack(lua_State *L) {
    luaL_register(L, MP_LUA_LIB_NAME, thislib);
	lua_reg_msgpacket_meta(L);
    return 1;
}