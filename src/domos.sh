export GOROOT=/root/.gvm/gos/go1.8
export GOPATH=/root/go
stty -F /dev/ttyS0 9600 clocal cread cs8 -cstopb -parenb
go run *.go
