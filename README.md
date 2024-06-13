# Telos EVM Proxy

## Dependencies

* Golang

## Build

```
go build cmd/main.go
```

```
//first arg is rpc url, second arg is port number
go run cmd/main.go <URL> <PORT>

//example:
./main.go https://rpc3.telos.net/ 8545
```