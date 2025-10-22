package bg

import (
	"net"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
)

const ChainTransferTopic = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"

type Contract struct {
	Addr      string
	TokenName string
	Decimals  uint8
}

type ScanTool interface {
	GetBlockNum() (int64, error)
	GetLog(blockNum int64) ([]*ContractTokenTran, error)
	ChainType() ChainType
	AddContract(...Contract)
}

func NewTool(chain ChainType, rpc []string, contracts ...Contract) ScanTool {
	dail := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 3 * time.Second,
	}
	transport := &http.Transport{
		DialContext:         dail.DialContext,
		MaxIdleConnsPerHost: 200,
		MaxIdleConns:        200,
		TLSHandshakeTimeout: 2 * time.Second,
	}
	cli := &http.Client{
		Transport: transport,
	}
	tmp := resty.NewWithClient(cli)
	tmp = tmp.SetBaseURL(rpc[0])
	switch chain {
	case CHAIN_TRON:
		//波场处理
		if len(rpc) > 2 {
			tmp = tmp.SetHeader("TRON-PRO-API-KEY", rpc[1])
		}
		t := &tronTool{httpclient: tmp}
		t.AddContract(contracts...)
		return t
	case CHAIN_ETH, CHAIN_BSC:
		t := &ethTool{chain_type: chain, httpclient: tmp}
		t.AddContract(contracts...)
		return t
	}
	return nil
}
