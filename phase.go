// Package bossphase 是 BOSS 阶段机：按血量阈值推进阶段，只前进不回退，跳级逐级触发。
// 缺陷：回血会掉回旧阶段、跳级只算一次、同一阶段重复触发、阈值边界差一、全扫阈值表。
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
	return &Machine{phases: append([]Phase(nil), phases...)}
}

// Update 结算一次血量，返回本次新触发的阶段名。
// 缺陷：用严格小于判阈值（正好到阈值不进阶段）、每次都重新定位（回血会回退）、
// 跳级只返回最后一个、同一阶段重复触发、全扫阈值表。
func (m *Machine) Update(hpPct float64) []string {
	var fired []string
	for index, phase := range m.phases {
		m.scanned++
		if hpPct < phase.BelowPct {
			m.index = index
			fired = append(fired, phase.Name)
		}
	}
	m.events += len(fired)
	if len(fired) == 0 {
		return nil
	}
	return fired[len(fired)-1:]
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
