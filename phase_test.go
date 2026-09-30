package bossphase

import "testing"

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
