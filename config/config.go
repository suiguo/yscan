package config

import (
	"github.com/suiguo/yscan/services/bg"
)

func Rpc(chain bg.ChainType) string {
	switch chain {
	case bg.CHAIN_ETH:
		return "https://goerli.gateway.tenderly.co"
	case bg.CHAIN_TRON:
		return "https://api.trongrid.io"
	}
	return ""
}
