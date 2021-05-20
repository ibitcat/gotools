#!/bin/bash

#rm -rf ./l-xlsx
go build -ldflags "-w -s" -o ../../../config/xlsx2lua xlsx2lua.go
chmod u+x ./upx
./upx ../../../config/xlsx2lua
