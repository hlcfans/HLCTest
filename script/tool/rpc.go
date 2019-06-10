package tool

import (
	"net/http"
	"net"
	"crypto/tls"
	"io/ioutil"
	"crypto/x509"
	"encoding/json"
	"bytes"
	"log"
	"time"
	"dag-bench-test-script/tool/socks"
)
const (
	MaxIdleConnections int = 20
	RequestTimeout     int = 5
)

type RpcClient struct {
	Cfg *Config
}
// newHTTPClient returns a new HTTP client that is configured according to the
// proxy and TLS settings in the associated connection configuration.
func (rpc *RpcClient)newHTTPClient() (*http.Client, error) {
	// Configure proxy if needed.
	var dial func(network, addr string) (net.Conn, error)
	if rpc.Cfg.Proxy != "" {
		proxy := &socks.Proxy{
			Addr:     rpc.Cfg.Proxy,
			Username: rpc.Cfg.ProxyUser,
			Password: rpc.Cfg.ProxyPass,
		}
		dial = func(network, addr string) (net.Conn, error) {
			c, err := proxy.Dial(network, addr)
			if err != nil {
				return nil, err
			}
			return c, nil
		}
	}

	// Configure TLS if needed.
	var tlsConfig *tls.Config
	if !rpc.Cfg.NoTLS && rpc.Cfg.RPCCert != "" {
		pem, err := ioutil.ReadFile(rpc.Cfg.RPCCert)
		if err != nil {
			return nil, err
		}

		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(pem)
		tlsConfig = &tls.Config{
			RootCAs:            pool,
			InsecureSkipVerify: true,
		}
	}

	// Create and return the new HTTP client potentially configured with a
	// proxy and TLS.
	client := http.Client{
		Transport: &http.Transport{
			Dial:            dial,
			TLSClientConfig: tlsConfig,
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 5 * time.Second,
				DualStack: true,
			}).DialContext,
		},
	}
	return &client, nil
}

func (rpc *RpcClient)RpcResult(method string,params []interface{}) []byte{
	protocol := "http"
	if !rpc.Cfg.NoTLS {
		protocol = "https"
	}
	paramStr,err := json.Marshal(params)
	if err != nil {
		log.Println("rpc params error:",err)
		return nil
	}
	url := protocol + "://" + rpc.Cfg.RPCServer
	jsonStr := []byte(`{"jsonrpc": "2.0", "method": "`+method+`", "params": `+string(paramStr)+`, "id": 1}`)
	bodyBuff := bytes.NewBuffer(jsonStr)
	httpRequest, err := http.NewRequest("POST", url, bodyBuff)
	if err != nil {
		log.Println("rpc connect failed",err)
		return nil
	}
	httpRequest.Close = true
	httpRequest.Header.Set("Content-Type", "application/json")
	// Configure basic access authorization.
	httpRequest.SetBasicAuth(rpc.Cfg.RPCUser, rpc.Cfg.RPCPassword)

	// Create the new HTTP client that is configured according to the user-
	// specified options and submit the request.
	httpClient, err := rpc.newHTTPClient()
	if err != nil {
		log.Println("rpc auth faild",err)
		return nil
	}
	httpResponse, err := httpClient.Do(httpRequest)
	if err != nil {
		log.Println("rpc request faild",err)
		return nil
	}
	body, err := ioutil.ReadAll(httpResponse.Body)
	httpResponse.Body.Close()
	if err != nil {
		log.Println("error reading json reply:", err)
		return nil
	}

	if httpResponse.Status != "200 OK" {
		log.Println("error http response :",  httpResponse.Status, body)
		return nil
	}
	return body
}
