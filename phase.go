// Package bossphase 是 BOSS 阶段机：按血量阈值推进阶段，只前进不回退，跳级逐级触发。
package bossphase

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
// 阈值含边界（血量不大于 BelowPct 即进入）；只前进不回退；
// 跨过多个阶段时按顺序逐级各触发一次；定位用二分，单次 O(log n)。
func (m *Machine) Update(hpPct float64) []string {
	next := m.index + 1
	if next >= len(m.phases) {
		return nil
	}
	m.scanned++
	if hpPct > m.phases[next].BelowPct {
		return nil
	}
	// 只在尚未进入的尾部里二分：找第一个 BelowPct < hpPct 的位置。
	lo, hi := next, len(m.phases)
	for lo < hi {
		mid := lo + (hi-lo)/2
		m.scanned++
		if m.phases[mid].BelowPct < hpPct {
			hi = mid
		} else {
			lo = mid + 1
		}
	}
	target := lo - 1
	fired := make([]string, 0, target-next+1)
	for index := next; index <= target; index++ {
		fired = append(fired, m.phases[index].Name)
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
