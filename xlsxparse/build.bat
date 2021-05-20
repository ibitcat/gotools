echo off

go build -ldflags "-w -s"
.\upx ./xlsxparse.exe
pause