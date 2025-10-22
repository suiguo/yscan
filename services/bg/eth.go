package bg

///采用新的bsc client
import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/suiguo/yscan/services/utils"

	"github.com/go-resty/resty/v2"
)

const TransferFix = "0xa9059cbb"

type BlockByNumberResp struct {
	Jsonrpc string              `json:"jsonrpc"`
	ID      int64               `json:"id"`
	Error   Error               `json:"error"`
	Result  BlockByNumberResult `json:"result"`
}

type BlockByNumberResult struct {
	Difficulty       string                     `json:"difficulty"`
	ExtraData        string                     `json:"extraData"`
	GasLimit         string                     `json:"gasLimit"`
	GasUsed          string                     `json:"gasUsed"`
	Hash             string                     `json:"hash"`
	LogsBloom        string                     `json:"logsBloom"`
	Miner            string                     `json:"miner"`
	MixHash          string                     `json:"mixHash"`
	Nonce            string                     `json:"nonce"`
	Number           string                     `json:"number"`
	ParentHash       string                     `json:"parentHash"`
	ReceiptsRoot     string                     `json:"receiptsRoot"`
	Sha3Uncles       string                     `json:"sha3Uncles"`
	Size             string                     `json:"size"`
	StateRoot        string                     `json:"stateRoot"`
	Timestamp        string                     `json:"timestamp"`
	TotalDifficulty  string                     `json:"totalDifficulty"`
	Transactions     []BlockByNumberTransaction `json:"transactions"`
	TransactionsRoot string                     `json:"transactionsRoot"`
	Uncles           []interface{}              `json:"uncles"`
}

type BlockByNumberTransaction struct {
	BlockHash        string `json:"blockHash"`
	BlockNumber      string `json:"blockNumber"`
	From             string `json:"from"`
	Gas              string `json:"gas"`
	GasPrice         string `json:"gasPrice"`
	Hash             string `json:"hash"`
	Input            string `json:"input"`
	Nonce            string `json:"nonce"`
	To               string `json:"to"`
	TransactionIndex string `json:"transactionIndex"`
	Value            string `json:"value"`
	Type             string `json:"type"`
	ChainID          string `json:"chainId"`
}

///

type BlockNumber struct {
	Jsonrpc string `json:"jsonrpc"`
	Error   Error  `json:"error"`
	ID      int64  `json:"id"`
	Result  string `json:"result"`
}

type Error struct {
	Code    int64  `json:"code"`
	Message string `json:"message"`
}

type ReceiptRespon struct {
	Jsonrpc string `json:"jsonrpc"`
	ID      int64  `json:"id"`
	Result  Result `json:"result"`
}

type Result struct {
	BlockHash         string      `json:"blockHash"`
	BlockNumber       string      `json:"blockNumber"`
	ContractAddress   interface{} `json:"contractAddress"`
	CumulativeGasUsed string      `json:"cumulativeGasUsed"`
	EffectiveGasPrice string      `json:"effectiveGasPrice"`
	From              string      `json:"from"`
	GasUsed           string      `json:"gasUsed"`
	LogsBloom         string      `json:"logsBloom"`
	Status            Status      `json:"status"`
	To                string      `json:"to"`
	TransactionHash   string      `json:"transactionHash"`
	TransactionIndex  string      `json:"transactionIndex"`
	Type              string      `json:"type"`
}

type Status string

const (
	FailStatus    Status = "0x0"
	SuccessStatus Status = "0x1"
)

type JsonRpcParam struct {
	Jsonrpc string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int64         `json:"id"`
}

// ///////
type ContractTokenTran struct {
	BlockNum int64 `json:"blockNum"`
	// Chain          string    `json:"chain"`
	Type          ChainType `json:"-"`
	Confirmations int64     `json:"confirmations"`
	FeeAmountCoin string    `json:"feeAmountCoin"`
	// FeeAmountUsdt string    `json:"feeAmountUsdt"`
	FeeSymbol string `json:"feeSymbol"`
	// FeeSymbolPrice string    `json:"feeSymbolPrice"`
	Remark    string `json:"remark"`
	RequestId string `json:"requestId"`
	Success   bool   `json:"result"` //是否是成功交易
	// Success bo
	Timestamp         int64               `json:"timestamp"`
	TransferTimestamp int64               `json:"transferTimestamp"`
	Transfers         []*CallbackTransfer `json:"transfers"`
	TxId              string              `json:"txid"`
}

type CallbackTransfer struct {
	Amount      string `json:"amount"`
	Contract    string `json:"contract"`
	FromAddress string `json:"fromAddress"`
	LogIdx      int    `json:"logIdx"`
	Symbol      string `json:"symbol"`
	ToAddress   string `json:"toAddress"`
}

type ethTool struct {
	monitorMap sync.Map // map[string]*Contract
	requestId  atomic.Int64
	chain_type ChainType
	httpclient *resty.Client
}

func (t *ethTool) AddContract(c ...Contract) {
	for idx := range c {
		data := c[idx]
		data.Addr = strings.ToLower(data.Addr)
		t.monitorMap.Store(data.Addr, &data)
	}
}
func (t *ethTool) R() *resty.Request {
	return t.httpclient.R()
}
func (t *ethTool) GetBlockNum() (int64, error) {
	idx := t.requestId.Add(1)
	resp := &BlockNumber{}
	_, err := t.R().SetResult(resp).SetBody(&JsonRpcParam{
		Jsonrpc: "2.0",
		Method:  "eth_blockNumber",
		ID:      idx,
	}).Post("")
	if err != nil {
		return 0, err
	}
	if resp.Error.Code != 0 {
		return 0, fmt.Errorf(resp.Error.Message)
	}
	return strconv.ParseInt(resp.Result, 0, 64)
}

func (t *ethTool) GetContract(address string) (*Contract, bool) {
	address = strings.ToLower(address)
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
func IsSuccess(hash string, rpc string) (bool, error) {
	req := &JsonRpcParam{
		Jsonrpc: "2.0",
		Method:  "eth_getTransactionReceipt",
		Params:  []any{hash},
		ID:      0,
	}
	//直接写死没时间改了
	r, code, err := utils.Req(utils.Post, rpc, nil, req)
	if err != nil {
		return false, err
	}
	if code != 200 {
		return false, fmt.Errorf("code not 200")
	}
	resp := &ReceiptRespon{}
	err = json.Unmarshal(r, resp)
	if err != nil {
		return false, err
	}
	if resp.Result.TransactionHash != hash {
		return false, fmt.Errorf("please retry")
	}
	return resp.Result.Status == SuccessStatus, nil
}

func (t *ethTool) GetLog(blockNum int64) ([]*ContractTokenTran, error) {
	nowblock, err := t.GetBlockNum()
	if err != nil {
		return nil, err
	}
	var has bool
	for i := 0; i < 5; i++ {
		has, err = t.hasTransfer(blockNum)
		if has && err == nil {
			break
		}
		time.Sleep(time.Millisecond * 500)
	}
	if err != nil {
		return nil, err
	}
	if !has {
		return []*ContractTokenTran{}, nil
	}
	transfer, err := t.getBlockByNum(blockNum, nowblock)
	if err != nil {
		return nil, err
	}
	return transfer, err
}
func (t *ethTool) getBlockByNum(blockNum int64, nowblock int64) ([]*ContractTokenTran, error) {
	idx := t.requestId.Add(1)
	info := &BlockByNumberResp{}
	_, err := t.R().SetResult(info).SetBody(&JsonRpcParam{
		Jsonrpc: "2.0",
		Method:  "eth_getBlockByNumber",
		ID:      idx,
		Params:  []any{fmt.Sprintf("0x%x", blockNum), true},
	}).Post("")
	if err != nil {
		return nil, err
	}
	if info.Error.Code != 0 {
		return nil, fmt.Errorf(info.Error.Message)
	}
	outtransfer := make([]*ContractTokenTran, 0)
	//bnb本币
	for _, tran := range info.Result.Transactions {
		transfertmp := &ContractTokenTran{
			Type:          t.ChainType(),
			Confirmations: nowblock - blockNum,
			FeeAmountCoin: t.ChainType().Name(),
			BlockNum:      blockNum,
			TxId:          tran.Hash,
			Transfers:     make([]*CallbackTransfer, 0),
		}
		//bnb交易
		if tran.Input == "0x" {
			amout, ok := new(big.Int).SetString(strings.ReplaceAll(tran.Value, "0x", ""), 16)
			if !ok {
				continue
			}
			realamount, err := utils.ChainValue(amout.String(), 18)
			if err != nil {
				continue
			}
			transfertmp.Transfers = append(transfertmp.Transfers,
				&CallbackTransfer{
					FromAddress: tran.From,
					Contract:    "",
					Amount:      realamount.String(),
					ToAddress:   tran.To,
					Symbol:      t.chain_type.Name(),
				})
		} else {
			//合约交易
			contract, ok := t.GetContract(tran.To)
			if !ok {
				continue
			}
			if len(tran.Input) != 64*2+len(TransferFix) {
				continue
			}
			if !strings.HasPrefix(tran.Input, TransferFix) {
				continue
			}
			toAddr := tran.Input[len(TransferFix) : len(TransferFix)+64]
			toAddr = fmt.Sprintf("0x%s", strings.TrimLeft(toAddr, "0"))
			value := tran.Input[len(TransferFix)+64:]
			val := new(big.Int)
			tran_val, err := hex.DecodeString(value)
			if err != nil {
				return nil, err
			}
			val = val.SetBytes(tran_val)
			realamount, err := utils.ChainValue(val.String(), contract.Decimals)
			if err != nil {
				return nil, err
			}
			transfertmp.Transfers = append(transfertmp.Transfers,
				&CallbackTransfer{
					FromAddress: tran.From,
					Contract:    tran.To,
					Amount:      realamount.String(),
					ToAddress:   toAddr,
					Symbol:      contract.TokenName,
				})
		}
		outtransfer = append(outtransfer, transfertmp)
	}
	return outtransfer, nil
}

func (t *ethTool) ChainType() ChainType {
	return t.chain_type
}

func (t *ethTool) hasTransfer(block int64) (bool, error) {
	idx := t.requestId.Add(1)
	resp := &BlockNumber{}
	_, err := t.R().SetResult(resp).SetBody(&JsonRpcParam{
		Jsonrpc: "2.0",
		Method:  "eth_getBlockTransactionCountByNumber",
		ID:      idx,
		Params:  []any{fmt.Sprintf("0x%x", block)},
	}).Post("")
	if err != nil {
		return false, err
	}
	if resp.Error.Message != "" {
		return false, fmt.Errorf(resp.Error.Message)
	}
	num, err := strconv.ParseInt(resp.Result, 0, 64)
	if err != nil {
		return false, err
	}
	return num > 0, nil
}
