echo off

go build -ldflags "-w -s" -o ../../../config/xlsx2lua.exe ./xlsx2lua.go
.\upx ../../../config/xlsx2lua.exe
pause