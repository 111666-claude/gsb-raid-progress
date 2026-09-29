package raidprogress

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func TestFirstReportRewards(t *testing.T) {
	rewards, duplicated := Apply(map[string]bool{}, []Report{{Player: "p", Boss: 1, At: 1}})
	if rewards != 1 || duplicated != 0 {
		t.Fatalf("首报应该发 1 次奖：%d %d", rewards, duplicated)
	}
}

func TestSameTimestampIsDuplicate(t *testing.T) {
	state := map[string]bool{}
	Apply(state, []Report{{Player: "p", Boss: 1, At: 1}})
	rewards, duplicated := Apply(state, []Report{{Player: "p", Boss: 1, At: 1}})
	if rewards != 0 || duplicated != 1 {
		t.Fatalf("同一时间戳应该算重复：%d %d", rewards, duplicated)
	}
}

func TestDifferentBossRewards(t *testing.T) {
	rewards, _ := Apply(map[string]bool{}, []Report{{Player: "p", Boss: 1, At: 1}, {Player: "p", Boss: 2, At: 2}})
	if rewards != 2 {
		t.Fatalf("两个 boss 应该发 2 次奖：%d", rewards)
	}
}

// 周重置口径：服务器周期为周一 05:00，不是本地时间的周二。
func TestResetDay(t *testing.T) {
	if !ShouldReset("Mon") {
		t.Fatal("服务器周期应该在周一重置")
	}
	if ShouldReset("Tue") {
		t.Fatal("周二不该重置")
	}
}

// 重连重报：同一进度换了时间戳重报，仍算重复。
func TestReconnectResendIsDuplicate(t *testing.T) {
	state := map[string]bool{}
	Apply(state, []Report{{Player: "p", Boss: 1, At: 100, Week: "w1"}})
	rewards, duplicated := Apply(state, []Report{{Player: "p", Boss: 1, At: 200, Week: "w1"}})
	if rewards != 0 || duplicated != 1 {
		t.Fatalf("换时间戳的重连重报应该算重复：%d %d", rewards, duplicated)
	}
}

// README 样本：3 次正常上报 + 3 次重连重报，应发 3 次奖、识别 3 次重复。
func TestReconnectSample(t *testing.T) {
	rewards, duplicated := Apply(map[string]bool{}, Reconnect())
	if rewards != 3 || duplicated != 3 {
		t.Fatalf("reconnect 样本应为 rewards=3 duplicated=3：%d %d", rewards, duplicated)
	}
}

// 重试命中已有进度：只做幂等校验，跳过不发奖。
func TestRetryHitIsSkipped(t *testing.T) {
	state := map[string]bool{}
	Apply(state, []Report{{Player: "p", Boss: 1, At: 1, Week: "w1"}})
	rewards, duplicated := Apply(state, []Report{{Player: "p", Boss: 1, At: 2, Retry: true, Week: "w1"}})
	if rewards != 0 || duplicated != 1 {
		t.Fatalf("重试命中应该跳过：%d %d", rewards, duplicated)
	}
}

// 重试新进度也只发一次奖：连续重试同一新进度，奖励恒为 1。
func TestRetryNewProgressRewardsOnce(t *testing.T) {
	reports := []Report{}
	for i := 0; i < 5; i++ {
		reports = append(reports, Report{Player: "p", Boss: 1, At: int64(i), Retry: true, Week: "w1"})
	}
	rewards, duplicated := Apply(map[string]bool{}, reports)
	if rewards != 1 || duplicated != 4 {
		t.Fatalf("重试新进度只发一次奖：%d %d", rewards, duplicated)
	}
}

// 周次是幂等键的一部分：同一玩家同一 boss 不同周次是新进度。
func TestDifferentWeekRewards(t *testing.T) {
	rewards, _ := Apply(map[string]bool{}, []Report{
		{Player: "p", Boss: 1, At: 1, Week: "w1"},
		{Player: "p", Boss: 1, At: 1, Week: "w2"},
	})
	if rewards != 2 {
		t.Fatalf("不同周次应该各发一次奖：%d", rewards)
	}
}

// 不变量：同一批上报重复处理，第二遍全部识别为重复，结果一致。
func TestReprocessSameBatchIsStable(t *testing.T) {
	batch := Reconnect()
	state := map[string]bool{}
	rewards1, duplicated1 := Apply(state, batch)
	rewards2, duplicated2 := Apply(state, batch)
	if rewards1 != 3 || duplicated1 != 3 {
		t.Fatalf("第一遍应为 rewards=3 duplicated=3：%d %d", rewards1, duplicated1)
	}
	if rewards2 != 0 || duplicated2 != len(batch) {
		t.Fatalf("第二遍应全部算重复：%d %d", rewards2, duplicated2)
	}
}

// 周重置边界：周一 05:00（服务器时区 UTC+8），与本地时间无关。
func TestResetBoundary(t *testing.T) {
	zone := time.FixedZone("CST", 8*3600)
	cases := []struct {
		at   time.Time
		want bool
	}{
		{time.Date(2026, 9, 28, 4, 59, 0, 0, zone), false}, // 周一 04:59 未到重置点
		{time.Date(2026, 9, 28, 5, 0, 0, 0, zone), true},   // 周一 05:00 重置
		{time.Date(2026, 9, 28, 23, 59, 0, 0, zone), true}, // 周一当天已过重置点
		{time.Date(2026, 9, 29, 6, 0, 0, 0, zone), false},  // 周二不重置
		{time.Date(2026, 9, 27, 6, 0, 0, 0, zone), false},  // 周日不重置
	}
	for _, c := range cases {
		if got := ShouldResetAt(c.at); got != c.want {
			t.Fatalf("ShouldResetAt(%v) = %v, want %v", c.at, got, c.want)
		}
	}
}

// 与本地时间无关：同一时刻在不同时区表示下，重置判断一致。
func TestResetIndependentOfLocalTime(t *testing.T) {
	// 2026-09-28 05:30 +08:00，即服务器周期周一重置点之后。
	at := time.Date(2026, 9, 27, 21, 30, 0, 0, time.UTC)
	if !ShouldResetAt(at) {
		t.Fatal("UTC 表示的同一时刻也应判定为已重置")
	}
	if ShouldResetAt(at.In(time.FixedZone("HST", -10*3600))) != ShouldResetAt(at) {
		t.Fatal("重置判断不该随本地时区变化")
	}
}

// reference 是参考实现：按唯一键 (玩家, boss, 周次) 去重后逐条发奖。
func reference(reports []Report) (rewards int, duplicated int) {
	seen := map[string]bool{}
	for _, r := range reports {
		k := fmt.Sprintf("%s#%d#%s", r.Player, r.Boss, r.Week)
		if seen[k] {
			duplicated++
			continue
		}
		seen[k] = true
		rewards++
	}
	return rewards, duplicated
}

// 对照自证：随机 200 组「正常上报 + 重连重报 + 重试」序列，与参考实现差异必须为 0。
func TestApplyMatchesReference(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	for round := 0; round < 200; round++ {
		n := 1 + rng.Intn(50)
		reports := make([]Report, 0, n)
		for i := 0; i < n; i++ {
			reports = append(reports, Report{
				Player: fmt.Sprintf("p%d", rng.Intn(4)),
				Boss:   1 + rng.Intn(3),
				At:     int64(rng.Intn(1000)),
				Retry:  rng.Intn(2) == 0,
				Week:   fmt.Sprintf("w%d", 1+rng.Intn(2)),
			})
		}
		gotRewards, gotDuplicated := Apply(map[string]bool{}, reports)
		wantRewards, wantDuplicated := reference(reports)
		if gotRewards != wantRewards || gotDuplicated != wantDuplicated {
			t.Fatalf("第 %d 组不一致：got (%d, %d), want (%d, %d)",
				round, gotRewards, gotDuplicated, wantRewards, wantDuplicated)
		}
	}
}

// 规模：10 万条上报 1 秒内处理完，单条 O(1)，无全表扫描。
func TestApplyScales(t *testing.T) {
	reports := make([]Report, 0, 100000)
	for i := 0; i < 100000; i++ {
		reports = append(reports, Report{
			Player: fmt.Sprintf("p%d", i%1000),
			Boss:   i % 10,
			At:     int64(i),
			Week:   fmt.Sprintf("w%d", i%4),
		})
	}
	start := time.Now()
	rewards, duplicated := Apply(map[string]bool{}, reports)
	elapsed := time.Since(start)
	if elapsed > time.Second {
		t.Fatalf("10 万条上报耗时 %v，超过 1 秒", elapsed)
	}
	if rewards+duplicated != 100000 {
		t.Fatalf("计数不完整：%d %d", rewards, duplicated)
	}
}
