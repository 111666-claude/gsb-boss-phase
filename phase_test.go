package bossphase

import (
	"math/rand"
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
	if fired := machine.Update(50); len(fired) != 2 || fired[0] != "p1" || fired[1] != "p2" {
		t.Fatalf("血量正好 50%% 应含边界，逐级触发 p1,p2：%v", fired)
	}
	if machine.Current() != "p2" {
		t.Fatalf("当前阶段应为 p2，实际 %q", machine.Current())
	}

	machine = sample()
	if fired := machine.Update(80); len(fired) == 0 || fired[0] != "p1" {
		t.Fatalf("血量正好 80%% 应含边界进入 p1：%v", fired)
	}
}

func TestHealNeverRollsBack(t *testing.T) {
	machine := sample()
	machine.Update(10)
	machine.Update(60)
	machine.Update(100)
	if machine.Current() != "p3" {
		t.Fatalf("回血后阶段不该回退，应仍为 p3，实际 %q", machine.Current())
	}
	if machine.Events() != 3 {
		t.Fatalf("回血不该重复触发，events 应仍为 3，实际 %d", machine.Events())
	}
}

func TestSkipFiresEveryPhaseInOrder(t *testing.T) {
	machine := sample()
	fired := machine.Update(10)
	want := []string{"p1", "p2", "p3"}
	if len(fired) != len(want) {
		t.Fatalf("满血掉到 10%% 应逐级触发 3 个阶段：%v", fired)
	}
	for index := range want {
		if fired[index] != want[index] {
			t.Fatalf("触发顺序应为 %v，实际 %v", want, fired)
		}
	}
	if machine.Events() != 3 || machine.Current() != "p3" {
		t.Fatalf("events=3 current=p3 期望，实际 events=%d current=%q", machine.Events(), machine.Current())
	}
}

func TestRepeatedSettlementIsIdempotent(t *testing.T) {
	machine := sample()
	machine.Update(10)
	again := machine.Update(10)
	if len(again) != 0 {
		t.Fatalf("同一阶段重复结算不该再次触发：%v", again)
	}
	if machine.Events() != 3 {
		t.Fatalf("重复结算后 events 应仍为 3，实际 %d", machine.Events())
	}

	machine.Update(50)
	machine.Update(0)
	if machine.Events() != 3 {
		t.Fatalf("已触发过的阶段在任何血量下都不该重复触发，实际 events=%d", machine.Events())
	}
}

func TestStepwiseThenSkipAdvancesMonotonically(t *testing.T) {
	machine := sample()
	if fired := machine.Update(70); len(fired) != 1 || fired[0] != "p1" {
		t.Fatalf("70%% 应只触发 p1：%v", fired)
	}
	if fired := machine.Update(10); len(fired) != 2 || fired[0] != "p2" || fired[1] != "p3" {
		t.Fatalf("再掉到 10%% 应按顺序补触发 p2,p3：%v", fired)
	}
}

func TestScannedNotLinearInPhaseCount(t *testing.T) {
	phases := make([]Phase, 0, 3000)
	for index := 0; index < 3000; index++ {
		phases = append(phases, Phase{Name: "p", BelowPct: 99.99 - float64(index)/100.0})
	}
	machine := NewMachine(phases)
	machine.Update(100)
	if scanned := machine.Scanned(); scanned != 1 {
		t.Fatalf("不推进的结算应被哨兵挡住只扫 1 条，实际 %d", scanned)
	}

	phases = make([]Phase, 0, 3000)
	for index := 0; index < 3000; index++ {
		phases = append(phases, Phase{Name: "p", BelowPct: 100 - float64(index)/100.0})
	}
	instance := NewMachine(phases)
	for query := 0; query < 3000; query++ {
		instance.Update(99.5)
	}
	if instance.Scanned() > 6000 {
		t.Fatalf("3000 次结算总扫描数应在 6000 以内，实际 %d", instance.Scanned())
	}
}

// referenceMachine 用独立写法实现同一口径：线性定位最深阶段 + 只前进 + 逐级触发。
type referenceMachine struct {
	phases []Phase
	index  int
	events int
}

func newReference(phases []Phase) *referenceMachine {
	return &referenceMachine{phases: append([]Phase(nil), phases...), index: -1}
}

func (ref *referenceMachine) update(hpPct float64) []string {
	target := -1
	for index, phase := range ref.phases {
		if hpPct <= phase.BelowPct {
			target = index
		}
	}
	var fired []string
	for index := ref.index + 1; index <= target; index++ {
		fired = append(fired, ref.phases[index].Name)
	}
	ref.events += len(fired)
	if target > ref.index {
		ref.index = target
	}
	return fired
}

func (ref *referenceMachine) current() string {
	if ref.index < 0 || ref.index >= len(ref.phases) {
		return ""
	}
	return ref.phases[ref.index].Name
}

func TestRandomDifferentialAgainstReference(t *testing.T) {
	random := rand.New(rand.NewSource(20261001))
	for trial := 0; trial < 200; trial++ {
		count := random.Intn(30) + 1
		phases := make([]Phase, count)
		thresholds := map[float64]bool{}
		for index := range phases {
			var threshold float64
			for {
				threshold = float64(random.Intn(101))
				if !thresholds[threshold] {
					thresholds[threshold] = true
					break
				}
			}
			phases[index] = Phase{Name: "p" + string(rune('a'+index%26)) + string(rune('a'+(index/26)%26)), BelowPct: threshold}
		}
		// 按 BelowPct 从大到小排序，符合阶段表有序前提。
		for index := 1; index < len(phases); index++ {
			for previous := index; previous > 0 && phases[previous-1].BelowPct < phases[previous].BelowPct; previous-- {
				phases[previous-1], phases[previous] = phases[previous], phases[previous-1]
			}
		}
		// 重新给唯一名，避免重名掩盖触发集合差异。
		for index := range phases {
			phases[index].Name = phaseName(index)
		}

		actual := NewMachine(phases)
		reference := newReference(phases)

		sequenceLength := random.Intn(40) + 1
		hpSequence := make([]float64, 0, sequenceLength)
		for step := 0; step < sequenceLength; step++ {
			var hpPct float64
			switch random.Intn(10) {
			case 0, 1:
				hpPct = phases[random.Intn(len(phases))].BelowPct // 恰好落在阈值上
			default:
				hpPct = float64(random.Intn(101)) // 含回血、跳级
			}
			hpSequence = append(hpSequence, hpPct)

			gotFired := actual.Update(hpPct)
			wantFired := reference.update(hpPct)
			if !equalStrings(gotFired, wantFired) {
				t.Fatalf("trial=%d step=%d hp=%v phases=%v 触发集合不一致：got=%v want=%v",
					trial, step, hpPct, phaseThresholds(phases), gotFired, wantFired)
			}
			if actual.Current() != reference.current() {
				t.Fatalf("trial=%d step=%d hp=%v 当前阶段不一致：got=%q want=%q",
					trial, step, hpPct, actual.Current(), reference.current())
			}
			if actual.Events() != reference.events {
				t.Fatalf("trial=%d step=%d hp=%v Events 不一致：got=%d want=%d",
					trial, step, hpPct, actual.Events(), reference.events)
			}
		}

		// 同一串血量序列在终态上重复执行，不再产生新触发，结果相同。
		checkpointIndex := actual.index
		checkpointEvents := actual.Events()
		for _, hpPct := range hpSequence {
			if fired := actual.Update(hpPct); len(fired) != 0 {
				t.Fatalf("trial=%d 终态后重复执行血量序列却再次触发：hp=%v fired=%v", trial, hpPct, fired)
			}
		}
		if actual.index != checkpointIndex || actual.Events() != checkpointEvents {
			t.Fatalf("trial=%d 到终态后重复结算改变了状态：index %d->%d events %d->%d",
				trial, checkpointIndex, actual.index, checkpointEvents, actual.Events())
		}
	}
}

func phaseName(index int) string {
	return "p" + string(rune('a'+index%26)) + string(rune('a'+(index/26)%26))
}

func equalStrings(a, b []string) bool {
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

func phaseThresholds(phases []Phase) []float64 {
	thresholds := make([]float64, len(phases))
	for index, phase := range phases {
		thresholds[index] = phase.BelowPct
	}
	return thresholds
}

func BenchmarkUpdate3000Phases(b *testing.B) {
	phases := make([]Phase, 0, 3000)
	for index := 0; index < 3000; index++ {
		phases = append(phases, Phase{Name: "p", BelowPct: 100 - float64(index)/100.0})
	}
	machine := NewMachine(phases)
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		machine.Update(99.5)
	}
}
