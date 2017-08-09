@echo off

set /p version=<"version.txt"
set curpath=%cd%
echo svnÂ·¾¶=%curpath%
echo.

xcopy %curpath%\docs\tools %curpath%\tools\ /EXCLUDE:%curpath%\docs\tools\exclude_xcopy.txt /s/e/y/r
%curpath%\docs\tools\7z a -tzip "bombsvr_v"%version%.zip %curpath% -xr@"%curpath%\docs\tools\exclude_zip.txt"
rd /s /Q %curpath%\tools
pause