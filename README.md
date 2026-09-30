# boss-phase

BOSS 阶段机：按血量阈值推进阶段，只前进不回退，跳级逐级触发，只用 Go 标准库。

```
go test ./...
go run ./cmd/boss-phase --sample skip
go run ./cmd/boss-phase --sample heal
go run ./cmd/boss-phase --sample boundary
```

## 口径（README 为准）

- **阈值含边界**：血量不大于 `BelowPct` 就进入该阶段。
- **只前进不回退**：血量回满也不许退回旧阶段。
- **跳级逐级触发**：一次掉血跨过多个阶段时，中间每个阶段都要按顺序触发一次并把
  `Events` 累加对应次数；同一阶段重复调用不重复触发。
- **查询代价**：阶段表有序，定位用二分，`Scanned` 不随阶段数线性增长。
- **规模**：单片 3000 个阶段、每秒 10 万次结算，单次 O(log n)，内存 O(阶段数)。

## 输出契约（不改格式）

```
fired=p1,p2,p3 events=3
current=p3
current=p2
events=3
scanned<=6000
```

## 目录

```
phase.go               阶段机
cmd/boss-phase         命令行入口
phase_test.go          go test 用例
```
