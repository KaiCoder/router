go clean
del /q "../bin/*.txt"
del /q ."./bin/cpu.prof"
del /q "../bin/*.exe"
RD /s /q "../bin/xConf"
go build -v -o ../bin/router.exe
pause
cd ../bin
router.exe