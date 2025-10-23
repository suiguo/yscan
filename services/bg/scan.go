package bg

import (
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

type Disk interface {
	Save(key string, val int64) error
	Get(key string) int64
}

type ChainScanCfg struct {
	Chain        ChainType
	ConfirmNum   int
	ContractList []Contract
	Rpc          []string
}

type storeTool struct {
	Working []chan struct{} // 每个分片一个信号量，防重入
	GoNum   int64
	ScanTool
	cfg   ChainScanCfg
	disk  Disk
	cache sync.Map
}

// NewScan gonum=并发分片数
func NewScan(gonum int64, disk Disk, log *zap.Logger, cfgs ...ChainScanCfg) *Scan {
	if gonum <= 0 {
		gonum = 1
	}
	s := &Scan{
		popChan: make(chan []*ContractTokenTran, 2000),
		zap_l:   log,
	}
	for _, cfg := range cfgs {
		t := &storeTool{
			ScanTool: NewTool(cfg.Chain, cfg.Rpc, cfg.ContractList...),
			cfg:      cfg,
			GoNum:    gonum,
			disk:     disk,
			Working:  make([]chan struct{}, gonum),
		}
		for i := range t.Working {
			t.Working[i] = make(chan struct{}, 1)
		}
		s.chain.Store(cfg.Chain, t)
	}
	return s
}

type Scan struct {
	chain   sync.Map
	zap_l   *zap.Logger
	popChan chan []*ContractTokenTran
}

func (s *Scan) Result() <-chan []*ContractTokenTran { return s.popChan }

func (s *Scan) AddContract(chainType ChainType, contracts ...Contract) {
	tool, ok := s.chain.Load(chainType)
	if !ok {
		return
	}
	if t, ok := tool.(*storeTool); ok {
		t.AddContract(contracts...)
	}
}

func (s *Scan) Process() {
	s.chain.Range(func(_, v any) bool {
		t, ok := v.(*storeTool)
		if !ok {
			return true
		}
		nowBlockNum, err := t.GetBlockNum()
		if err != nil {
			if s.zap_l != nil {
				s.zap_l.Error("scan.process",
					zap.String("event", "get_head_failed"),
					zap.String("chain", t.ChainType().Name()),
					zap.String("err", err.Error()),
				)
			}
			return true
		}
		for i := 0; i < int(t.GoNum); i++ {
			idx := i
			go s.process(t, idx, nowBlockNum)
		}
		return true
	})
}

func (s *Scan) getSaveKey(t *storeTool, idx int) string {
	return fmt.Sprintf("%s:scan:shard:%d:checkpoint", t.ChainType().Name(), idx)
}

func (s *Scan) get(t *storeTool, idx int) int64 {
	key := s.getSaveKey(t, idx)
	if t.disk == nil {
		if v, ok := t.cache.Load(key); ok && v != nil {
			return v.(int64)
		}
		return 0
	}
	return t.disk.Get(key)
}

func (s *Scan) save(t *storeTool, idx int, val int64) error {
	key := s.getSaveKey(t, idx)
	if t.disk == nil {
		t.cache.Store(key, val)
		return nil
	}
	return t.disk.Save(key, val)
}

func (s *Scan) process(t *storeTool, idx int, nowBlockNum int64) {
	chainName := t.ChainType().Name()

	select {
	case t.Working[idx] <- struct{}{}:
		// ok
	default:
		return
	}

	// startTs := time.Now()
	// var scanBlock int64
	defer func() {
		<-t.Working[idx]
	}()

	// 读取 checkpoint（已处理的最后高度）
	last := s.get(t, idx)
	if last == 0 {
		init := nowBlockNum - int64(t.cfg.ConfirmNum) - 1
		if init < 0 {
			init = 0
		}
		last = init
		if err := s.save(t, idx, last); err != nil {
			if s.zap_l != nil {
				s.zap_l.Error("scan.process",
					zap.String("event", "init_save_failed"),
					zap.String("chain", chainName),
					zap.Int("shard", idx),
					zap.Int64("checkpoint", last),
					zap.String("err", err.Error()),
				)
			}
			return
		}
	}

	// 对齐到该分片的起始高度：最小的 h >= last+1 且 h % GoNum == idx
	start := last + 1
	rem := start % t.GoNum // int64
	want := int64(idx)
	if rem != want {
		delta := (want - rem + t.GoNum) % t.GoNum
		start += delta
	}
	startTs := time.Now()
	// 主循环：每次跨 GoNum 个高度
	for h := start; ; h += t.GoNum {
		// 确认数检查；不足确认先停，等待下次触发
		if lag := nowBlockNum - h; lag < int64(t.cfg.ConfirmNum) {
			return
		}
		// scanBlock = h
		results, err := t.GetLog(h)
		if err != nil {
			if s.zap_l != nil {
				s.zap_l.Error("scan.process",
					zap.String("event", "getlog_failed"),
					zap.String("chain", chainName),
					zap.Int("shard", idx),
					zap.Int64("block", h),
					zap.String("err", err.Error()),
				)
			}
			return
		}
		if s.zap_l != nil {
			s.zap_l.Info("scan.process",
				zap.String("event", "dispatch"),
				zap.Int64("scan", h),
				zap.Int("current_shard", idx),
				zap.Int64("elapsed_ms", time.Since(startTs).Milliseconds()),
			)
		}
		// 发送（at-least-once 语义）
		if len(results) > 0 {
			s.popChan <- results
		}

		// 保存 checkpoint（已完成 h）
		if err := s.save(t, idx, h); err != nil {
			if s.zap_l != nil {
				s.zap_l.Error("scan.process",
					zap.String("event", "save_failed"),
					zap.String("chain", chainName),
					zap.Int("shard", idx),
					zap.Int64("block", h),
					zap.String("err", err.Error()),
				)
			}
			return
		}
	}
}
