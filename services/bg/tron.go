package bg

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"sync"

	"github.com/suiguo/yscan/services/utils"

	"github.com/go-resty/resty/v2"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/shopspring/decimal"
)

// 只要直接转账的，不接受合约转账的 数据
var trxDecimal = decimal.New(1, 6)

type SolidityData struct {
	BlockID      string              `json:"blockID"`
	BlockHeader  SolidityBlockHeader `json:"block_header"`
	Transactions []Transaction       `json:"transactions"`
}

type TxMeta struct {
	Type         string // 顶层合约类型：TriggerSmartContract / TransferContract / ...
	TopContract  string // 顶层调用目标合约（规范化为 hex41，如 41xxxxxxxx...）
	OwnerAddress string // 顶层调用者
}
type SolidityBlockHeader struct {
	RawData          BlockHeaderRawData `json:"raw_data"`
	WitnessSignature string             `json:"witness_signature"`
}

type BlockHeaderRawData struct {
	Number         int64  `json:"number"`
	TxTrieRoot     string `json:"txTrieRoot"`
	WitnessAddress string `json:"witness_address"`
	ParentHash     string `json:"parentHash"`
	Version        int64  `json:"version"`
	Timestamp      int64  `json:"timestamp"`
}

type Transaction struct {
	Ret        []Ret              `json:"ret"`
	Signature  []string           `json:"signature"`
	TxID       string             `json:"txID"`
	RawData    TransactionRawData `json:"raw_data"`
	RawDataHex string             `json:"raw_data_hex"`
}

type TransactionRawData struct {
	Contract      []SolidityContract `json:"contract"`
	RefBlockBytes string             `json:"ref_block_bytes"`
	RefBlockHash  string             `json:"ref_block_hash"`
	Expiration    int64              `json:"expiration"`
	Timestamp     int64              `json:"timestamp,omitempty"`
	FeeLimit      int64              `json:"fee_limit,omitempty"`
	Data          string             `json:"data,omitempty"`
}

type SolidityContract struct {
	Parameter    Parameter `json:"parameter"`
	Type         string    `json:"type"`
	PermissionID *int64    `json:"Permission_id,omitempty"`
}

type Parameter struct {
	Value   Value  `json:"value"`
	TypeURL string `json:"type_url"`
}

type Value struct {
	Amount          int64   `json:"amount,omitempty"`
	OwnerAddress    string  `json:"owner_address"`
	ToAddress       string  `json:"to_address,omitempty"`
	Data            *string `json:"data,omitempty"`
	ContractAddress string  `json:"contract_address,omitempty"`
	Resource        *string `json:"resource,omitempty"`
	ReceiverAddress *string `json:"receiver_address,omitempty"`
	AssetName       *string `json:"asset_name,omitempty"`
	Balance         *int64  `json:"balance,omitempty"`
	AccountAddress  *string `json:"account_address,omitempty"`
	Votes           []Vote  `json:"votes"`
	CallValue       *int64  `json:"call_value,omitempty"`
}

type Vote struct {
	VoteAddress string `json:"vote_address"`
	VoteCount   int64  `json:"vote_count"`
}

type Ret struct {
	ContractRet string `json:"contractRet"`
}

const (
	AccountCreateContract      = "AccountCreateContract"
	DelegateResourceContract   = "DelegateResourceContract"
	TransferAssetContract      = "TransferAssetContract"
	TransferContract           = "TransferContract"
	TriggerSmartContract       = "TriggerSmartContract"
	UnDelegateResourceContract = "UnDelegateResourceContract"
	UnfreezeBalanceContract    = "UnfreezeBalanceContract"
	VoteWitnessContract        = "VoteWitnessContract"
	WithdrawBalanceContract    = "WithdrawBalanceContract"
)

const (
	OutOfEnergy = "OUT_OF_ENERGY"
	Success     = "SUCCESS"
)

type Element struct {
	Log             []Log    `json:"log"`
	Fee             int64    `json:"fee,omitempty"`
	BlockNumber     int64    `json:"blockNumber"`
	ContractResult  []string `json:"contractResult"`
	BlockTimeStamp  int64    `json:"blockTimeStamp"`
	Receipt         Receipt  `json:"receipt"`
	ID              string   `json:"id"`
	ContractAddress string   `json:"contract_address,omitempty"`
}

type Log struct {
	Address string   `json:"address"`
	Data    string   `json:"data"`
	Topics  []string `json:"topics"`
}

type Receipt struct {
	Result             string `json:"result,omitempty"`
	EnergyPenaltyTotal int64  `json:"energy_penalty_total,omitempty"`
	EnergyFee          int64  `json:"energy_fee,omitempty"`
	EnergyUsageTotal   int64  `json:"energy_usage_total,omitempty"`
	OriginEnergyUsage  int64  `json:"origin_energy_usage,omitempty"`
	NetUsage           int64  `json:"net_usage,omitempty"`
	EnergyUsage        int64  `json:"energy_usage,omitempty"`
	NetFee             int64  `json:"net_fee,omitempty"`
}

// 全部使用固化区块方法
const getNowBlock = "/walletsolidity/getblock"
const getLastBlock = "/walletsolidity/getblock"

const getTranByNum = "/walletsolidity/gettransactioninfobyblocknum"
const getTrxTranByNum = "/walletsolidity/getblockbynum"

type TronBlockInfo struct {
	BlockID     string      `json:"blockID"`
	BlockHeader BlockHeader `json:"block_header"`
}

type BlockHeader struct {
	RawData          RawData `json:"raw_data"`
	WitnessSignature string  `json:"witness_signature"`
}

type RawData struct {
	Number         int64  `json:"number"`
	TxTrieRoot     string `json:"txTrieRoot"`
	WitnessAddress string `json:"witness_address"`
	ParentHash     string `json:"parentHash"`
	Version        int64  `json:"version"`
	Timestamp      int64  `json:"timestamp"`
}

func normT(s string) string {
	if s == "" {
		return ""
	}
	ss := strings.TrimSpace(s)
	// 已经是 T 开头的地址，直接返回
	if strings.HasPrefix(ss, "T") {
		return ss
	}
	// 去掉 0x 前缀
	ss = strings.TrimPrefix(ss, "0x")
	ss = strings.TrimPrefix(ss, "0X")
	if strings.HasPrefix(ss, "41") && len(ss) == 42 {
		_, err := hex.DecodeString(ss)
		if err == nil {
			return address.HexToAddress(ss).String() // 转成 Base58 T...
		}
	}
	if len(ss) == 40 {
		hex41 := "41" + ss
		return address.HexToAddress(hex41).String()
	}
	if addr, err := address.Base58ToAddress(ss); err == nil {
		return addr.String()
	}
	return ss
}

type tronTool struct {
	httpclient *resty.Client
	monitorMap sync.Map // map[string]*Contract
}

func (t *tronTool) R() *resty.Request {
	return t.httpclient.R()
}

func (t *tronTool) GetBlockNum() (int64, error) {
	block := &TronBlockInfo{}
	_, err := t.R().SetResult(block).Post(getNowBlock)
	if err != nil {
		return 0, err
	}
	if block.BlockHeader.RawData.Number > 0 {
		return block.BlockHeader.RawData.Number, nil
	}
	return 0, fmt.Errorf("block numer is zero")
}

func (t *tronTool) getLastBlockNum() (int64, error) {
	block := &TronBlockInfo{}
	_, err := t.R().SetResult(block).Post(getLastBlock)
	if err != nil {
		return 0, err
	}
	if block.BlockHeader.RawData.Number > 0 {
		return block.BlockHeader.RawData.Number, nil
	}
	return 0, fmt.Errorf("block numer is zero")
}
func (t *tronTool) GetLog(blockNum int64) ([]*ContractTokenTran, error) {
	lastBlockNum, err := t.getLastBlockNum()
	if err != nil {
		return nil, err
	}
	// 这里返回 trx 转账 和 txMeta
	trx, txMeta, err := t.TrxTransfer(blockNum, lastBlockNum)
	if err != nil {
		return nil, err
	}
	// 传入 txMeta 做过滤
	out, err := t.Trc20Transfer(blockNum, lastBlockNum, trx, txMeta)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// 之前是 (map[string]*ContractTokenTran, error)
// 现在加一个返回：txMeta map
func (t *tronTool) TrxTransfer(blockNum int64, lastBlock int64) (map[string]*ContractTokenTran, map[string]*TxMeta, error) {
	param := map[string]any{"num": blockNum, "visible": true}
	data := &SolidityData{}
	_, err := t.R().SetResult(data).SetBody(param).Post(getTrxTranByNum)
	if err != nil {
		return nil, nil, err
	}

	trxOut := make(map[string]*ContractTokenTran)
	txMeta := make(map[string]*TxMeta)

	for _, rawTran := range data.Transactions {
		// 记录顶层合约信息
		m := &TxMeta{}
		if len(rawTran.RawData.Contract) > 0 {
			c0 := rawTran.RawData.Contract[0]
			m.Type = c0.Type
			m.OwnerAddress = c0.Parameter.Value.OwnerAddress
			if c0.Type == TriggerSmartContract {
				m.TopContract = normT(c0.Parameter.Value.ContractAddress)
			}
		}
		txMeta[rawTran.TxID] = m

		// === 你原有的 TRX 直转处理（保持不变） ===
		if len(rawTran.Ret) == 0 || len(rawTran.RawData.Contract) == 0 {
			continue
		}
		tmp := &ContractTokenTran{
			Type:              t.ChainType(),
			BlockNum:          data.BlockHeader.RawData.Number,
			TransferTimestamp: rawTran.RawData.Timestamp,
			Confirmations:     lastBlock - data.BlockHeader.RawData.Number,
			TxId:              rawTran.TxID,
			Success:           rawTran.Ret[0].ContractRet == Success,
			Remark:            rawTran.Ret[0].ContractRet,
			Transfers:         make([]*CallbackTransfer, 0),
		}
		for idx, contract := range rawTran.RawData.Contract {
			if contract.Type != TransferContract {
				continue
			}
			value := contract.Parameter.Value
			amount := decimal.NewFromInt(value.Amount)
			if value.OwnerAddress == "" || value.ToAddress == "" || amount.LessThanOrEqual(decimal.Zero) {
				continue
			}
			tmp.Transfers = append(tmp.Transfers, &CallbackTransfer{
				FromAddress: value.OwnerAddress,
				ToAddress:   value.ToAddress,
				Contract:    "TRX",
				Symbol:      "TRX",
				Amount:      amount.Div(trxDecimal).String(),
				LogIdx:      idx,
			})
		}
		if len(tmp.Transfers) > 0 {
			trxOut[rawTran.TxID] = tmp
		}
	}
	return trxOut, txMeta, nil
}

// 之前是 (blockNum, lastBlock, trx map)
// 现在加一个入参 txMeta
func (t *tronTool) Trc20Transfer(blockNum int64, lastBlock int64, trx map[string]*ContractTokenTran, txMeta map[string]*TxMeta) ([]*ContractTokenTran, error) {
	param := map[string]any{"num": blockNum, "visible": true}
	showData := make([]Element, 0)
	_, err := t.R().SetResult(&showData).SetBody(param).Post(getTranByNum)
	if err != nil {
		return nil, err
	}

	out := make([]*ContractTokenTran, 0)

	for _, logs := range showData {
		// 先给 TRX（若存在）补手续费
		if logs.Receipt.Result == "" {
			if tmp, ok := trx[logs.ID]; ok {
				feeCoin := decimal.NewFromInt(logs.Receipt.EnergyFee + logs.Receipt.NetFee + logs.Receipt.EnergyPenaltyTotal).Div(trxDecimal)
				tmp.FeeSymbol = "TRX"
				tmp.FeeAmountCoin = feeCoin.String()
			}
			continue
		}
		feeCoin := decimal.NewFromInt(logs.Receipt.EnergyFee + logs.Receipt.NetFee).Div(trxDecimal)
		transferData := &ContractTokenTran{
			Type:              t.ChainType(),
			BlockNum:          logs.BlockNumber,
			TransferTimestamp: logs.BlockTimeStamp,
			TxId:              logs.ID,
			FeeSymbol:         "TRX",
			FeeAmountCoin:     feeCoin.String(),
			Confirmations:     lastBlock - logs.BlockNumber,
			Success:           logs.Receipt.Result == Success,
			Remark:            logs.Receipt.Result,
		}

		token := normT(logs.ContractAddress)

		for idx, lg := range logs.Log {
			if len(lg.Topics) != 3 {
				continue
			}
			if !strings.HasPrefix(lg.Topics[0], "0x") {
				lg.Topics[0] = "0x" + lg.Topics[0]
			}
			if !strings.HasPrefix(lg.Topics[1], "0x") {
				lg.Topics[1] = "0x" + lg.Topics[1]
			}
			if !strings.HasPrefix(lg.Topics[2], "0x") {
				lg.Topics[2] = "0x" + lg.Topics[2]
			}
			if lg.Topics[0] != ChainTransferTopic {
				continue
			}
			meta := txMeta[logs.ID]
			if meta == nil || meta.Type != TriggerSmartContract || meta.TopContract == "" || meta.TopContract != token {
				continue
			}
			if len(lg.Topics[0]) != 66 || len(lg.Topics[1]) != 66 || len(lg.Topics[2]) != 66 {
				continue
			}
			fromHex := lg.Topics[1][26:]
			toHex := lg.Topics[2][26:]
			from := address.HexToAddress("41" + fromHex).String()
			to := address.HexToAddress("41" + toHex).String()
			tranVal, err := hex.DecodeString(strings.TrimPrefix(lg.Data, "0x"))
			if err != nil {
				continue
			}
			val := new(big.Int).SetBytes(tranVal)
			contractInfo, ok := t.GetContract(logs.ContractAddress)
			if !ok {
				continue
			}
			amount, err := utils.ChainValue(val.String(), contractInfo.Decimals)
			if err != nil {
				continue
			}
			//只检查首层所以后续没有交易了
			transferData.Transfers = append(transferData.Transfers, &CallbackTransfer{
				FromAddress: from,
				ToAddress:   to,
				Contract:    contractInfo.Addr,
				Symbol:      contractInfo.TokenName,
				Amount:      amount.String(),
				LogIdx:      idx,
			})
			break
		}

		if len(transferData.Transfers) > 0 {
			out = append(out, transferData)
		}
	}

	// 合并同块里的 TRX 直转
	for _, val := range trx { // map → slice 无序 OK
		out = append(out, val)
	}
	return out, nil
}

func (t *tronTool) AddContract(c ...Contract) {
	for idx := range c {
		data := c[idx]
		data.Addr = normT(data.Addr)
		t.monitorMap.Store(data.Addr, &data)
	}
}

func (t *tronTool) GetContract(address string) (*Contract, bool) {
	info, ok := t.monitorMap.Load(address)
	if !ok {
		return nil, false
	}
	contractInfo := info.(*Contract)
	if contractInfo.Addr == "" {
		return nil, false
	}
	return contractInfo, true
}

func (t *tronTool) ChainType() ChainType {
	return CHAIN_TRON
}
