setlocal

set GOOS=linux
set GOARCH=amd64

call go build main.go

call scp main csc@koth.cyber.lab:~
call scp -r public csc@koth.cyber.lab:~/

endlocal