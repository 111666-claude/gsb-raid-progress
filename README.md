# raid-progress

副本进度上报与奖励发放（幂等、重试、周重置），只用 Go 标准库。

```
go test ./...
go run ./cmd/raid-progress --sample reconnect
go run ./cmd/raid-progress --sample week
```

## 口径（README 为准）

- **幂等键不能带时间戳**：同一次进度要用 `(玩家, boss, 周次)` 做键。
  `--sample reconnect`（3 次正常上报 + 3 次断线重连重报）按口径是 `rewards=3 duplicated=3`，
  现在键里带了 `At`，给 `rewards=6 duplicated=0`（玩家会收到双份奖励）。
- **重试不能重复发奖**：`Retry=true` 的请求只做幂等校验，命中就跳过；重试新进度也只发一次。
- **周重置按服务器周期**：服务器周期是周一 05:00 重置。`--sample week` 按口径是
  `reset_mon=true reset_tue=false`，现在按本地时间的周二，给 `reset_mon=false reset_tue=true`。
- 不变量：任意重复上报序列下，每个 `(玩家, boss, 周次)` 最多发一次奖；
  同一批上报重复处理结果相同；幂等键与数据库里的唯一索引一致。
- 规模：10 万条上报 1 秒内处理完，内存 O(键数)；不许每次上报都全表扫描。

## 现在的行为

- `key := fmt.Sprintf("%s#%d#%d", 玩家, boss, At)`：换一个时间戳就变成新进度。
- `Retry` 分支照样 `rewards++`。
- `ShouldReset` 判断的是 `"Tue"`。

## 计划新增的能力

```
--sample reconnect|week
```

## 输出

```
reports=6 rewards=3 duplicated=3
reset_mon=true reset_tue=false
```

## 目录

```
progress.go              幂等、重试与周重置
cmd/raid-progress        命令行入口
progress_test.go         go test 用例
```
