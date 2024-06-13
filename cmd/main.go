package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

type JsonRPC struct {
	Id      int    `json:"id"`
	Jsonrpc string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

type JsonRPCResponse struct {
	Jsonrpc string `json:"jsonrpc"`
	Result  any    `json:"result"`
	Error   any    `json:"error"`
}

type RoundTripper func(*http.Request) (*http.Response, error)

func (rt RoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	return rt(r)
}

func main() {
	//first arg is RPC URL, second arg is port number
	rpcUrl := os.Args[1]
	rpcPort := ":" + os.Args[2]

	// telosUrl, err := url.Parse("https://rpc3.telos.net/")
	telosUrl, err := url.Parse(rpcUrl)
	if err != nil {
		panic(err)
	}

	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(telosUrl)
			r.Out.Host = r.In.Host // if desired

		},
		Transport: RoundTripper(func(r *http.Request) (*http.Response, error) {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				return nil, err
			}

			r.Body = io.NopCloser(bytes.NewReader(body))

			jsonrpcRequests, err := parseJsonRPCRequests(body)
			if err != nil {
				return nil, err
			}

			for _, req := range jsonrpcRequests {
				fmt.Printf("Proxying request %d %s %v to %s\n", req.Id, req.Method, req.Params, r.URL.String())
			}

			resp, err := http.DefaultTransport.RoundTrip(r)
			if err != nil {
				return nil, err
			}

			if hasETHGetBlockByNumber(jsonrpcRequests) {
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					return nil, err
				}

				resp.Body = io.NopCloser(bytes.NewReader(body))

				jsonrpcResponses, err := parseJsonRPCResponses(body)
				if err != nil {
					return nil, err
				}

				for _, respJsonRPC := range jsonrpcResponses {
					if respJsonRPC.Error != nil {
						return resp, nil
					}

					if respJsonRPC.Result != nil {
						if block, ok := respJsonRPC.Result.(map[string]any); ok {
							if transactions, ok := block["transactions"].([]any); ok {
								for _, tx := range transactions {
									if txTyped, ok := tx.(map[string]any); ok {
										//TODO: should "42" be in hex?
										if txTyped["from"] == "0x0" || txTyped["from"] == "0x0000000000000000000000000000000000000000" || txTyped["v"] == "0x2a" {
											fmt.Println("Found a transaction with from 0x0, setting r, s, v to 0x0")
											txTyped["r"] = "0x0"
											txTyped["s"] = "0x0"
											txTyped["v"] = "0x0"
										}
									}
								}
							}
						}
					}
				}

				var newBody []byte
				if len(jsonrpcResponses) == 1 {
					newBody, err = json.Marshal(jsonrpcResponses[0])
					if err != nil {
						return nil, err
					}
				} else {
					newBody, err = json.Marshal(jsonrpcResponses)
					if err != nil {
						return nil, err
					}
				}

				fmt.Println(string(newBody))

				resp.Header.Set("Content-Length", fmt.Sprintf("%d", len(newBody)))

				resp.Body = io.NopCloser(bytes.NewReader(newBody))
				resp.ContentLength = int64(len(newBody))
			}

			return resp, nil
		}),
	}

	// err = http.ListenAndServe(":8545", proxy)
	err = http.ListenAndServe(rpcPort, proxy)
	if err != nil {
		panic(err)
	}
}

func parseJsonRPCRequests(body []byte) ([]JsonRPC, error) {
	var jsonrpc JsonRPC
	err := json.Unmarshal(body, &jsonrpc)

	var jsonrpcs []JsonRPC
	if err != nil {
		err = json.Unmarshal(body, &jsonrpcs)
		if err != nil {
			return nil, err
		}
	} else {
		jsonrpcs = append(jsonrpcs, jsonrpc)
	}

	return jsonrpcs, nil
}

func parseJsonRPCResponses(body []byte) ([]*JsonRPCResponse, error) {
	var jsonrpcResponse *JsonRPCResponse
	err := json.Unmarshal(body, &jsonrpcResponse)

	var jsonrpcResponses []*JsonRPCResponse
	if err != nil {
		err = json.Unmarshal(body, &jsonrpcResponses)
		if err != nil {
			return nil, err
		}
	} else {
		jsonrpcResponses = append(jsonrpcResponses, jsonrpcResponse)
	}

	return jsonrpcResponses, nil
}

func hasETHGetBlockByNumber(reqs []JsonRPC) bool {
	for _, req := range reqs {
		if req.Method == "eth_getBlockByNumber" {
			return true
		}
	}
	return false
}
