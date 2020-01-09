package notify

import (
	"bytes"
	"context"
	"io/ioutil"
	"net"
	"net/http"
	"time"
)

type HttpClient struct {
	client    *http.Client
	targetURL string
}

// NewHttpClient returns a reference to a HttpClient.
func NewHttpClient(targetURL string) *HttpClient {
	// we use a custom transport to control the idle connections settings.
	// thus, we can avoid closing connections to quickly. since we connect
	// to the same host and port always we aim to save handshakes
	t := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 90 * time.Second,
		}).DialContext,
		MaxIdleConns:        150, // keep idle connections open for reuse
		MaxIdleConnsPerHost: 150, // we connect to the same host:post always
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 5 * time.Second,
	}
	return &HttpClient{
		client:    &http.Client{Transport: t},
		targetURL: targetURL,
	}
}

func (ns *HttpClient) Post(ctx context.Context, msg []byte) PostResult {
	req, err := http.NewRequest(http.MethodPost, ns.targetURL, bytes.NewBuffer(msg))
	if err != nil {
		return PostResult{
			Err: err,
		}
	}
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "text/plain")

	resp, err := ns.client.Do(req)
	if err != nil {
		return PostResult{
			Err: err,
		}
	}

	// note, it is ensured, that a Body is always present in the Response so the
	// call to Close cannot result in a runtime panic.
	// further, it is the callers responsibility to read and close the body
	// otherwise the default HTTP client's Transport may not reuse HTTP/1.x
	// "keep-alive" TCP connections. although, we use a custom transport, we
	// still make sure to follow the best practices here.
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return PostResult{
			Err: PostErr{
				Err:      err.Error(),
				Response: resp,
			},
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return PostResult{
			Err: PostErr{
				Err:      string(body),
				Response: resp,
			},
		}
	}

	// success
	return PostResult{
		Body: body,
	}
}
