# Telos EVM Proxy

## Description
This proxy is intended to set the signature values for transactions to zero, if the signature is invalid (i.e. from a bridge transaction or a transaction signed and sent from the Telos native network)

## Dependencies

* Golang

## Build

```
go build cmd/main.go
```

```
//first arg is rpc url, second arg is port number
./main.go <URL> <PORT>

//example:
./main.go https://rpc3.telos.net/ 8545
```
