package bossphase

import (
	"fmt"
	"math/rand"
	"sort"
	"testing"
)

func sample() *Machine {
	return NewMachine([]Phase{
		{Name: "p1", BelowPct: 80},
		{Name: "p2", BelowPct: 50},
		{Name: "p3", BelowPct: 20},
	})
}

func TestFullHealthFiresNothing(t *testing.T) {
	if fired := sample().Update(100); len(fired) != 0 {
		t.Fatalf("满血不该触发阶段：%v", fired)
	}
}

func TestLowHealthFires(t *testing.T) {
	if fired := sample().Update(10); len(fired) == 0 {
		t.Fatal("血量很低应该触发阶段")
	}
}

func TestCurrentAfterTrigger(t *testing.T) {
	machine := sample()
	machine.Update(10)
	if machine.Current() == "" {
		t.Fatal("触发之后当前阶段不该为空")
	}
}

func TestScannedStartsAtZero(t *testing.T) {
	if NewMachine(nil).Scanned() != 0 {
		t.Fatal("初始扫描数应该是 0")
	}
}

func TestBoundaryInclusive(t *testing.T) {
	machine := sample()
	fired := machine.Update(50)
	if machine.Current() != "p2" {
		t.Fatalf("正好落在 50%% 阈值应该进入 p2，当前 %q", machine.Current())
	}
	want := []string{"p1", "p2"}
	if len(fired) != len(want) || fired[0] != want[0] || fired[1] != want[1] {
		t.Fatalf("50%% 应该依次触发 p1、p2：%v", fired)
	}
}

func TestHealDoesNotRegress(t *testing.T) {
	machine := sample()
	machine.Update(10)
	machine.Update(60)
	machine.Update(100)
	if machine.Current() != "p3" {
		t.Fatalf("回血不许退回旧阶段，当前 %q", machine.Current())
	}
	if machine.Events() != 3 {
		t.Fatalf("回血不该新增事件，events=%d", machine.Events())
	}
}

func TestSkipFiresEveryPhaseInOrder(t *testing.T) {
	machine := sample()
	fired := machine.Update(10)
	want := []string{"p1", "p2", "p3"}
	if len(fired) != len(want) {
		t.Fatalf("掉血跨多个阶段要逐级触发：%v", fired)
	}
	for index := range want {
		if fired[index] != want[index] {
			t.Fatalf("触发顺序必须按阶段顺序：%v", fired)
		}
	}
	if machine.Current() != "p3" {
		t.Fatalf("当前阶段应该是 p3，实际 %q", machine.Current())
	}
	if machine.Events() != 3 {
		t.Fatalf("events 应该累计 3 次，实际 %d", machine.Events())
	}
}

func TestRepeatedUpdateDoesNotRefire(t *testing.T) {
	machine := sample()
	first := machine.Update(10)
	if len(first) != 3 {
		t.Fatalf("首次应该触发 3 个阶段：%v", first)
	}
	for repeat := 0; repeat < 5; repeat++ {
		if fired := machine.Update(10); len(fired) != 0 {
			t.Fatalf("同一血量重复结算不该再触发：%v", fired)
		}
	}
	if machine.Events() != 3 {
		t.Fatalf("events 应该仍然是 3，实际 %d", machine.Events())
	}
}

func TestScannedStaysLogarithmic(t *testing.T) {
	phases := make([]Phase, 0, 3000)
	for index := 0; index < 3000; index++ {
		phases = append(phases, Phase{Name: fmt.Sprintf("p-%d", index), BelowPct: 100 - float64(index)/100.0})
	}
	machine := NewMachine(phases)
	for query := 0; query < 3000; query++ {
		machine.Update(99.5)
	}
	if machine.Scanned() > 6000 {
		t.Fatalf("扫描条数不该随阶段数线性增长：scanned=%d", machine.Scanned())
	}
}

// refMachine 是参考实现：全表二分定位 + 只前进 + 逐级触发。
type refMachine struct {
	phases []Phase
	index  int
}

func (r *refMachine) update(hpPct float64) []string {
	lo, hi := 0, len(r.phases)
	for lo < hi {
		mid := lo + (hi-lo)/2
		if r.phases[mid].BelowPct < hpPct {
			hi = mid
		} else {
			lo = mid + 1
		}
	}
	target := lo - 1
	if target <= r.index {
		return nil
	}
	fired := make([]string, 0, target-r.index)
	for index := r.index + 1; index <= target; index++ {
		fired = append(fired, r.phases[index].Name)
	}
	r.index = target
	return fired
}

func (r *refMachine) current() string {
	if r.index < 0 || r.index >= len(r.phases) {
		return ""
	}
	return r.phases[r.index].Name
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for index := range a {
		if a[index] != b[index] {
			return false
		}
	}
	return true
}

func TestRandomizedAgainstReference(t *testing.T) {
	rng := rand.New(rand.NewSource(20261001))
	for trial := 0; trial < 200; trial++ {
		count := 1 + rng.Intn(50)
		thresholds := make([]int, count)
		for index := range thresholds {
			thresholds[index] = 1 + rng.Intn(200) // 允许重复阈值
		}
		sort.Sort(sort.Reverse(sort.IntSlice(thresholds)))
		phases := make([]Phase, count)
		for index, value := range thresholds {
			phases[index] = Phase{Name: fmt.Sprintf("p%d", index), BelowPct: float64(value) / 2}
		}

		machine := NewMachine(phases)
		replay := NewMachine(phases)
		ref := &refMachine{phases: phases, index: -1}
		refEvents := 0

		steps := 1 + rng.Intn(200)
		for step := 0; step < steps; step++ {
			var hpPct float64
			switch rng.Intn(3) {
			case 0:
				hpPct = float64(rng.Intn(101)) // 含回血、跳级
			case 1:
				hpPct = phases[rng.Intn(count)].BelowPct // 恰好落在阈值上
			default:
				hpPct = float64(rng.Intn(201)) / 2
			}

			fired := machine.Update(hpPct)
			refFired := ref.update(hpPct)
			replayFired := replay.Update(hpPct)
			refEvents += len(refFired)

			if !sameStrings(fired, refFired) {
				t.Fatalf("trial=%d step=%d hp=%v 触发集合不一致：got=%v want=%v", trial, step, hpPct, fired, refFired)
			}
			if machine.Current() != ref.current() {
				t.Fatalf("trial=%d step=%d hp=%v 当前阶段不一致：got=%q want=%q", trial, step, hpPct, machine.Current(), ref.current())
			}
			if machine.Events() != refEvents {
				t.Fatalf("trial=%d step=%d events 不一致：got=%d want=%d", trial, step, machine.Events(), refEvents)
			}
			if machine.index != ref.index {
				t.Fatalf("trial=%d step=%d 阶段索引不一致：got=%d want=%d", trial, step, machine.index, ref.index)
			}
			if !sameStrings(replayFired, fired) {
				t.Fatalf("trial=%d step=%d 同一序列重复执行结果不同", trial, step)
			}
		}
	}
}

func TestIndexMonotonicAndEventsCount(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	machine := sample()
	previous := machine.index
	totalFired := 0
	for step := 0; step < 500; step++ {
		hpPct := float64(rng.Intn(101))
		fired := machine.Update(hpPct)
		if machine.index < previous {
			t.Fatalf("阶段索引倒退：%d -> %d", previous, machine.index)
		}
		previous = machine.index
		totalFired += len(fired)
		if machine.Events() != totalFired {
			t.Fatalf("events 不等于真正触发过的阶段次数：got=%d want=%d", machine.Events(), totalFired)
		}
	}
}
