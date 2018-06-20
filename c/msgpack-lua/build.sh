#!/bin/bash

gcc -c -g -Wall -I/usr/include/lua5.1 -fPIC cmsgpack.c
gcc -c -g -Wall -I/usr/include/lua5.1 -fPIC lmsgpack.c
gcc -Wall -fPIC -W -shared -o msgpack.so *.o
