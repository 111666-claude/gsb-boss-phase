// Command boss-phase 跑阶段机样例。
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"example.com/bossphase"
)

func machine() *bossphase.Machine {
	return bossphase.NewMachine([]bossphase.Phase{
		{Name: "p1", BelowPct: 80},
		{Name: "p2", BelowPct: 50},
		{Name: "p3", BelowPct: 20},
	})
}

func firedText(fired []string) string {
	if len(fired) == 0 {
		return "none"
	}
	return strings.Join(fired, ",")
}

// Run 执行一次命令行调用，返回退出码。
func Run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("boss-phase", flag.ContinueOnError)
	flags.SetOutput(stderr)
	sample := flags.String("sample", "skip", "skip / heal / boundary / idem / scan")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	switch *sample {
	case "skip":
		// 血量从满血直接掉到 10%。
		instance := machine()
		fired := instance.Update(10)
		fmt.Fprintf(stdout, "fired=%s events=%d\n", firedText(fired), instance.Events())
	case "heal":
		instance := machine()
		instance.Update(10)
		instance.Update(60)
		fmt.Fprintf(stdout, "current=%s\n", instance.Current())
	case "boundary":
		instance := machine()
		instance.Update(50)
		fmt.Fprintf(stdout, "current=%s\n", instance.Current())
	case "idem":
		instance := machine()
		instance.Update(10)
		instance.Update(10)
		fmt.Fprintf(stdout, "events=%d\n", instance.Events())
	case "scan":
		phases := make([]bossphase.Phase, 0, 3000)
		for index := 0; index < 3000; index++ {
			phases = append(phases, bossphase.Phase{Name: fmt.Sprintf("p-%d", index), BelowPct: 100 - float64(index)/100.0})
		}
		instance := bossphase.NewMachine(phases)
		for query := 0; query < 3000; query++ {
			instance.Update(99.5)
		}
		fmt.Fprintf(stdout, "scanned=%d\n", instance.Scanned())
	default:
		fmt.Fprintln(stderr, "需要 --sample skip|heal|boundary|idem|scan")
		return 2
	}
	return 0
}

func main() {
	os.Exit(Run(os.Args[1:], os.Stdout, os.Stderr))
}
