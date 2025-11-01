package main

import (
	"fmt"

	"github.com/suiguo/yscan/config"
	"github.com/suiguo/yscan/services/bg"
)

func main() {
	scan := bg.NewWork(10,
		nil,
		nil,
		// logger.GetLogger("tag").Zap(),
		bg.ChainScanCfg{
			Chain:      bg.CHAIN_TRON,
			ConfirmNum: 0, //确认区块数
			ContractList: []bg.Contract{
				{
					Addr:      "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",
					TokenName: "USDT",
					Decimals:  6,
				},
			}, //默认支持本币
			Rpc: []string{config.Rpc(bg.CHAIN_TRON), ""},
		})
	scan.Run()
	for slice := range scan.Result() {
		for _, val := range slice {
			if !val.Success {
				fmt.Println("scan", val)
			}
		}
	}
}
