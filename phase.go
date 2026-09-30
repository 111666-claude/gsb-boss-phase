// Package bossphase 是 BOSS 阶段机：按血量阈值推进阶段，只前进不回退，跳级逐级触发。
package bossphase

import "sort"

// Phase 是一个阶段，血量降到 BelowPct（含）时进入。
type Phase struct {
	Name     string
	BelowPct float64
}

// Machine 是阶段机。phases 按 BelowPct 从大到小排列。
type Machine struct {
	phases  []Phase
	index   int
	events  int
	scanned int
}

// NewMachine 建一个阶段机。
func NewMachine(phases []Phase) *Machine {
	return &Machine{phases: append([]Phase(nil), phases...), index: -1}
}

// Update 结算一次血量，返回本次新触发的阶段名。
// 阈值含边界（hpPct <= BelowPct 即进入）；当前阶段索引只增不减；
// 一次掉血跨过多个阶段时，中间每个阶段按顺序各触发一次并累计 Events。
// 定位用二分（单次 O(log n)），且先以一条哨兵阈值挡住不推进的结算，
// 使重复/回血这类无推进结算为 O(1)，Scanned 不随阶段数线性增长。
func (m *Machine) Update(hpPct float64) []string {
	next := m.index + 1
	if next >= len(m.phases) {
		return nil
	}
	m.scanned++
	if hpPct > m.phases[next].BelowPct {
		return nil
	}

	// hpPct <= 下一个阶段的阈值，在后缀 phases[next:] 上二分最深可达阶段。
	target := next + sort.Search(len(m.phases)-next, func(off int) bool {
		m.scanned++
		return hpPct > m.phases[next+off].BelowPct
	}) - 1

	fired := make([]string, 0, target-m.index)
	for ; next <= target; next++ {
		m.scanned++
		fired = append(fired, m.phases[next].Name)
	}
	m.index = target
	m.events += len(fired)
	return fired
}

// Current 是当前阶段名。
func (m *Machine) Current() string {
	if m.index < 0 || m.index >= len(m.phases) {
		return ""
	}
	return m.phases[m.index].Name
}

// Events 是累计触发次数。
func (m *Machine) Events() int { return m.events }

// Scanned 是累计扫过的阈值条数（规模观测）。
func (m *Machine) Scanned() int { return m.scanned }
