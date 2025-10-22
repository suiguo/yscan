package utils

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/shopspring/decimal"
)

var client *resty.Client
var tenPowers sync.Map

func init() {
	tsp := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   3 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          500,
		MaxIdleConnsPerHost:   500,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: 8 * time.Second,
	}
	cli := &http.Client{
		Transport: tsp,
	}
	client = resty.NewWithClient(cli).
		SetRetryCount(3).
		SetTimeout(time.Second * 3).
		SetRetryWaitTime(300 * time.Millisecond).
		SetRetryMaxWaitTime(3 * time.Second)
}

type Method string

const (
	Post Method = "POST"
	Get  Method = "GET"
)

// Request 请求url获得返回结果，queryParams 格式化在参数后面的?a=b&c=d格式，jsonData 是 body json 的数据格式
func Req(method Method, url string, queryParams map[string]string, jsonData any) ([]byte, int, error) {
	r := client.R()
	if queryParams != nil {
		r = r.SetQueryParams(queryParams)
	}
	if jsonData != nil {
		r = r.SetBody(jsonData)
	}
	switch method {
	case Post:
		respon, err := r.Post(url)
		if err != nil {
			return nil, 0, err
		}
		if respon != nil {
			return respon.Body(), respon.StatusCode(), nil
		}
		return nil, 0, fmt.Errorf("nil respon")
	case Get:
		respon, err := r.Get(url)
		if err != nil {
			return nil, 0, err
		}
		if respon != nil {
			return respon.Body(), respon.StatusCode(), nil
		}
		return nil, 0, fmt.Errorf("nil respon")
	}
	return nil, 0, fmt.Errorf("unknown method")
}

func ChainValue(input string, decimals uint8) (decimal.Decimal, error) {
	out, err := decimal.NewFromString(input)
	if err != nil {
		return out, err
	}
	if v, ok := tenPowers.Load(decimals); ok {
		return out.Div(v.(decimal.Decimal)), nil
	}
	pow := decimal.New(1, int32(decimals))
	tenPowers.Store(decimals, pow)
	return out.Div(pow), nil
}
