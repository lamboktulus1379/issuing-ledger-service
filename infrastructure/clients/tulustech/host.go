package tulustech

import (
	"encoding/json"
	"fmt"

	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/clients"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/clients/tulustech/models"
)

type ITulusHost interface {
	GetRandomTyping(reqHeader models.ReqHeader) (models.ResTypingRandom, error)
}

type TulusHost struct {
	host string
}

func NewTulusHost(host string) ITulusHost {
	return &TulusHost{host: host}
}

func (TulusHost *TulusHost) GetRandomTyping(
	reqHeader models.ReqHeader,
) (models.ResTypingRandom, error) {
	var res models.ResTypingRandom

	endpoint := "/api/typings/random"
	method := "POST"

	reqMapHeader := map[string]string{
		"Accept":       reqHeader.Accept,
		"Content-Type": reqHeader.ContentType,
		"Cookie":       reqHeader.Cookie,
	}
	hostClient := clients.NewHost(TulusHost.host, endpoint, method, nil, reqMapHeader, nil)
	byteData, statusCode, err := hostClient.HTTPPost()
	if err != nil {
		return res, err
	}

	if err := json.Unmarshal(byteData, &res); err != nil {
		return res, err
	}

	if statusCode < 200 || statusCode > 299 {
		return res, fmt.Errorf("something occurred with server")
	}

	return res, nil
}
