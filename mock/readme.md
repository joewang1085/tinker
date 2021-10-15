## protoc
protoc -I . --go_out=plugins=grpc:./hello ./hello.proto  
## go mod
go mod init  