package ACS

import (
	"sync"
	"time"
)

// DelayMonitor 延迟监控器
type DelayMonitor struct {
	// 各协议组件的延迟记录
	RBCDelays        map[uint64]time.Duration // [epoch]延迟
	ABADelays        map[uint64]time.Duration // [epoch]延迟
	CommonCoinDelays map[uint64]time.Duration // [epoch]延迟
	TPKEDelays       map[uint64]time.Duration // [epoch]延迟
	TBLSDelays       map[uint64]time.Duration // [epoch]延迟

	// 各协议组件的开始时间
	RBCStartTimes        map[uint64]time.Time // [epoch]开始时间
	ABAStartTimes        map[uint64]time.Time // [epoch]开始时间
	CommonCoinStartTimes map[uint64]time.Time // [epoch]开始时间
	TPKEStartTimes       map[uint64]time.Time // [epoch]开始时间
	TBLSStartTimes       map[uint64]time.Time // [epoch]开始时间

	// 统计信息
	TotalRBCDelays        []time.Duration
	TotalABADelays        []time.Duration
	TotalCommonCoinDelays []time.Duration
	TotalTPKEDelays       []time.Duration
	TotalTBLSDelays       []time.Duration

	// 锁
	mutex sync.RWMutex
}

// NewDelayMonitor 创建延迟监控器
func NewDelayMonitor() *DelayMonitor {
	return &DelayMonitor{
		RBCDelays:             make(map[uint64]time.Duration),
		ABADelays:             make(map[uint64]time.Duration),
		CommonCoinDelays:      make(map[uint64]time.Duration),
		TPKEDelays:            make(map[uint64]time.Duration),
		TBLSDelays:            make(map[uint64]time.Duration),
		RBCStartTimes:         make(map[uint64]time.Time),
		ABAStartTimes:         make(map[uint64]time.Time),
		CommonCoinStartTimes:  make(map[uint64]time.Time),
		TPKEStartTimes:        make(map[uint64]time.Time),
		TBLSStartTimes:        make(map[uint64]time.Time),
		TotalRBCDelays:        make([]time.Duration, 0),
		TotalABADelays:        make([]time.Duration, 0),
		TotalCommonCoinDelays: make([]time.Duration, 0),
		TotalTPKEDelays:       make([]time.Duration, 0),
		TotalTBLSDelays:       make([]time.Duration, 0),
	}
}

// StartRBC 开始RBC延迟计时
func (dm *DelayMonitor) StartRBC(epoch uint64) {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()
	dm.RBCStartTimes[epoch] = time.Now()
}

// EndRBC 结束RBC延迟计时
func (dm *DelayMonitor) EndRBC(epoch uint64) time.Duration {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	if startTime, exists := dm.RBCStartTimes[epoch]; exists {
		delay := time.Since(startTime)
		dm.RBCDelays[epoch] = delay
		dm.TotalRBCDelays = append(dm.TotalRBCDelays, delay)
		delete(dm.RBCStartTimes, epoch)
		return delay
	}
	return 0
}

// StartABA 开始ABA延迟计时
func (dm *DelayMonitor) StartABA(epoch uint64) {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()
	dm.ABAStartTimes[epoch] = time.Now()
}

// EndABA 结束ABA延迟计时
func (dm *DelayMonitor) EndABA(epoch uint64) time.Duration {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	if startTime, exists := dm.ABAStartTimes[epoch]; exists {
		delay := time.Since(startTime)
		dm.ABADelays[epoch] = delay
		dm.TotalABADelays = append(dm.TotalABADelays, delay)
		delete(dm.ABAStartTimes, epoch)
		return delay
	}
	return 0
}

// StartCommonCoin 开始CommonCoin延迟计时
func (dm *DelayMonitor) StartCommonCoin(epoch uint64) {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()
	dm.CommonCoinStartTimes[epoch] = time.Now()
}

// EndCommonCoin 结束CommonCoin延迟计时
func (dm *DelayMonitor) EndCommonCoin(epoch uint64) time.Duration {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	if startTime, exists := dm.CommonCoinStartTimes[epoch]; exists {
		delay := time.Since(startTime)
		dm.CommonCoinDelays[epoch] = delay
		dm.TotalCommonCoinDelays = append(dm.TotalCommonCoinDelays, delay)
		delete(dm.CommonCoinStartTimes, epoch)
		return delay
	}
	return 0
}

// StartTPKE 开始TPKE延迟计时
func (dm *DelayMonitor) StartTPKE(epoch uint64) {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()
	dm.TPKEStartTimes[epoch] = time.Now()
}

// EndTPKE 结束TPKE延迟计时
func (dm *DelayMonitor) EndTPKE(epoch uint64) time.Duration {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	if startTime, exists := dm.TPKEStartTimes[epoch]; exists {
		delay := time.Since(startTime)
		dm.TPKEDelays[epoch] = delay
		dm.TotalTPKEDelays = append(dm.TotalTPKEDelays, delay)
		delete(dm.TPKEStartTimes, epoch)
		return delay
	}
	return 0
}

// StartTBLS 开始TBLS延迟计时
func (dm *DelayMonitor) StartTBLS(epoch uint64) {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()
	dm.TBLSStartTimes[epoch] = time.Now()
}

// EndTBLS 结束TBLS延迟计时
func (dm *DelayMonitor) EndTBLS(epoch uint64) time.Duration {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	if startTime, exists := dm.TBLSStartTimes[epoch]; exists {
		delay := time.Since(startTime)
		dm.TBLSDelays[epoch] = delay
		dm.TotalTBLSDelays = append(dm.TotalTBLSDelays, delay)
		delete(dm.TBLSStartTimes, epoch)
		return delay
	}
	return 0
}

// GetAverageDelays 获取平均延迟
func (dm *DelayMonitor) GetAverageDelays() map[string]time.Duration {
	dm.mutex.RLock()
	defer dm.mutex.RUnlock()

	averages := make(map[string]time.Duration)

	// 计算RBC平均延迟
	if len(dm.TotalRBCDelays) > 0 {
		var total time.Duration
		for _, delay := range dm.TotalRBCDelays {
			total += delay
		}
		averages["RBC"] = total / time.Duration(len(dm.TotalRBCDelays))
	}

	// 计算ABA平均延迟
	if len(dm.TotalABADelays) > 0 {
		var total time.Duration
		for _, delay := range dm.TotalABADelays {
			total += delay
		}
		averages["ABA"] = total / time.Duration(len(dm.TotalABADelays))
	}

	// 计算CommonCoin平均延迟
	if len(dm.TotalCommonCoinDelays) > 0 {
		var total time.Duration
		for _, delay := range dm.TotalCommonCoinDelays {
			total += delay
		}
		averages["CommonCoin"] = total / time.Duration(len(dm.TotalCommonCoinDelays))
	}

	// 计算TPKE平均延迟
	if len(dm.TotalTPKEDelays) > 0 {
		var total time.Duration
		for _, delay := range dm.TotalTPKEDelays {
			total += delay
		}
		averages["TPKE"] = total / time.Duration(len(dm.TotalTPKEDelays))
	}

	// 计算TBLS平均延迟
	if len(dm.TotalTBLSDelays) > 0 {
		var total time.Duration
		for _, delay := range dm.TotalTBLSDelays {
			total += delay
		}
		averages["TBLS"] = total / time.Duration(len(dm.TotalTBLSDelays))
	}

	return averages
}

// GetDelayStats 获取延迟统计信息
func (dm *DelayMonitor) GetDelayStats() map[string]interface{} {
	dm.mutex.RLock()
	defer dm.mutex.RUnlock()

	stats := make(map[string]interface{})

	// RBC统计
	stats["RBC"] = map[string]interface{}{
		"count":  len(dm.TotalRBCDelays),
		"delays": dm.TotalRBCDelays,
	}

	// ABA统计
	stats["ABA"] = map[string]interface{}{
		"count":  len(dm.TotalABADelays),
		"delays": dm.TotalABADelays,
	}

	// CommonCoin统计
	stats["CommonCoin"] = map[string]interface{}{
		"count":  len(dm.TotalCommonCoinDelays),
		"delays": dm.TotalCommonCoinDelays,
	}

	// TPKE统计
	stats["TPKE"] = map[string]interface{}{
		"count":  len(dm.TotalTPKEDelays),
		"delays": dm.TotalTPKEDelays,
	}

	// TBLS统计
	stats["TBLS"] = map[string]interface{}{
		"count":  len(dm.TotalTBLSDelays),
		"delays": dm.TotalTBLSDelays,
	}

	return stats
}
