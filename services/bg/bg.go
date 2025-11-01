package bg

import (
	"context"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

type ChainType int

const (
	UNKNOW     ChainType = iota
	CHAIN_BSC  ChainType = iota + 1000
	CHAIN_ETH  ChainType = iota + 1000
	CHAIN_TRON ChainType = iota + 1000
)

func (c ChainType) Name() string {
	switch c {
	case CHAIN_BSC:
		return "BSC"
	case CHAIN_ETH:
		return "ETH"
	case CHAIN_TRON:
		return "Tron"
	}
	return "Unknow"
}
func (c ChainType) IsValid() bool { //暂不支持这种链
	return c.Name() != "Unknow"
}

func NewNodeChain(chain string) ChainType {
	switch strings.ToLower(chain) {
	case strings.ToLower(CHAIN_BSC.Name()):
		return CHAIN_BSC
	case strings.ToLower(CHAIN_ETH.Name()):
		return CHAIN_ETH
	case strings.ToLower(CHAIN_TRON.Name()):
		return CHAIN_TRON
	}

	return UNKNOW
}

type WorkHandler struct {
	ctx    context.Context
	zap_l  *zap.Logger
	cancel context.CancelFunc
	once   sync.Once
	scan   *Scan
}

func (w *WorkHandler) Run() {
	w.once.Do(func() {
		go w.work()
	})
}
func (w *WorkHandler) work() {
	idx := 0
	for {
		idx++
		select {
		case <-w.ctx.Done():
			return
		default:
			w.scan.Process()
			time.Sleep(time.Second * 2)
		}
		if idx%5 == 0 {
			idx = 0
		}
	}
}
func (w *WorkHandler) Stop() {
	w.cancel()
}
func (w *WorkHandler) Result() <-chan []*ContractTokenTran {
	return w.scan.Result()
}

// NewWork maxGoNum 最大执行分组 cfg 链的配置
func NewWork(maxGoNum int, disk Disk, log *zap.Logger, cfgs ...ChainScanCfg) *WorkHandler {
	scan := NewScan(int64(maxGoNum), disk, log, cfgs...)
	ctx, cancel := context.WithCancel(context.Background())
	return &WorkHandler{
		zap_l:  log,
		ctx:    ctx,
		cancel: cancel,
		scan:   scan,
	}
}
