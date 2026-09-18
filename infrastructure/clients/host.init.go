package clients

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/go-querystring/query"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/logger"
)

const (
	ErrorReadingRequest = "Error reading request."
)

// HostInterface abstract class
type HostInterface interface {
	HTTPPost() ([]byte, int, error)
	HTTPGet() ([]byte, int, error)
	HTTPPatch() ([]byte, int, error)

	Do(req *http.Request) ([]byte, int, error)
}

// HostStruct actual class implementation
type HostStruct struct {
	Host       string
	Endpoint   string
	Method     string
	Data       interface{}
	Header     map[string]string
	QueryParam interface{}

	HTTPClient   *http.Client
	HTTPRequest  *http.Request
	HTTPResponse *http.Response

	Err error
}

// NewHost return struct that will implement the abstract class
func NewHost(
	host string,
	endpoint string,
	method string,
	data interface{},
	header map[string]string,
	queryParam interface{},
) HostInterface {
	return &HostStruct{
		Host:       host,
		Endpoint:   endpoint,
		Method:     method,
		Data:       data,
		Header:     header,
		QueryParam: queryParam,
	}
}

// HTTPPost
func (host *HostStruct) HTTPPost() ([]byte, int, error) {
	dataByte, _ := json.Marshal(host.Data)
	dataByteBuffer := bytes.NewBuffer(dataByte)
	req, err := http.NewRequest("POST", host.Host+host.Endpoint, dataByteBuffer)
	if host.Data == nil {
		req, err = http.NewRequest("POST", host.Host+host.Endpoint, nil)
	}

	if err != nil {
		logger.GetLogger().
			WithField("request", req).
			WithField("error", err).
			Info("Error while post")
		log.Fatal(ErrorReadingRequest, err)
		return nil, 500, err
	}

	// Send request
	resp, statusCode, err := host.Do(req)
	if err != nil {
		return nil, statusCode, err
	}

	return resp, statusCode, nil
}

// HTTPGet
func (host *HostStruct) HTTPGet() ([]byte, int, error) {
	req, err := http.NewRequest("GET", host.Host+host.Endpoint, nil)
	if err != nil {
		log.Fatal(ErrorReadingRequest, err)
	}
	// Send request
	resp, statusCode, err := host.Do(req)
	if err != nil {
		return nil, statusCode, err
	}

	return resp, statusCode, nil
}

// HTTPPatch
func (host *HostStruct) HTTPPatch() ([]byte, int, error) {
	dataByte, _ := json.Marshal(host.Data)
	req, err := http.NewRequest("PUT", host.Host+host.Endpoint, bytes.NewBuffer(dataByte))
	if err != nil {
		log.Fatal(ErrorReadingRequest, err)
	}
	// Send request
	resp, statusCode, err := host.Do(req)
	if err != nil {
		return nil, statusCode, err
	}

	return resp, statusCode, nil
}

// Do request
func (host *HostStruct) Do(req *http.Request) ([]byte, int, error) {
	for key, val := range host.Header {
		req.Header.Set(key, val)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	if host.QueryParam != nil {
		v, _ := query.Values(host.QueryParam)
		req.URL.RawQuery = v.Encode()
	}

	// Validate cookie and headers are attached
	// fmt.Println(req.Cookies())
	logger.GetLogger().
		WithField("host", host.Host+host.Endpoint).
		WithField("headers", req.Header).
		WithField("request", req.Body).
		WithField("method", host.Method).
		Info("Client request")

	// Set client timeout
	tr := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		MaxConnsPerHost:     100,
		IdleConnTimeout:     600 * time.Second,
	}
	host.HTTPClient = &http.Client{Timeout: time.Second * 20, Transport: tr}

	host.HTTPResponse, host.Err = host.HTTPClient.Do(req)
	if os.IsTimeout(host.Err) {
		return nil, 0, host.Err
	}
	if host.Err != nil {
		fmt.Println("Error reading response. ", host.Err)
		return nil, 0, host.Err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.GetLogger().
				WithField("error", err).
				Error("Error while closing body")
		}
	}(host.HTTPResponse.Body)

	body, err := io.ReadAll(host.HTTPResponse.Body)
	if err != nil {
		fmt.Println("Error network. ", err)
		return nil, 0, err
	}
	logger.GetLogger().
		WithField("status", host.HTTPResponse.Status).
		WithField("headers", host.HTTPResponse.Header).
		WithField("body", string(body)).
		Info("Client response")

	return body, host.HTTPResponse.StatusCode, nil
}
