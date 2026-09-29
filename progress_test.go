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

func TestResetDay(t *testing.T) {
	if !ShouldReset("Mon") {
		t.Fatal("服务器周期应该在周一重置")
	}
	if ShouldReset("Tue") {
		t.Fatal("周二不该重置")
	}
}

func TestReconnectSample(t *testing.T) {
	reports := Reconnect()
	rewards, duplicated := Apply(map[string]bool{}, reports)
	if rewards != 3 || duplicated != 3 {
		t.Fatalf("断线重连重报应该 rewards=3 duplicated=3：%d %d", rewards, duplicated)
	}
}

func TestSameProgressNewTimestampIsDuplicate(t *testing.T) {
	state := map[string]bool{}
	Apply(state, []Report{{Player: "p", Boss: 1, At: 100, Week: "w1"}})
	rewards, duplicated := Apply(state, []Report{{Player: "p", Boss: 1, At: 200, Week: "w1"}})
	if rewards != 0 || duplicated != 1 {
		t.Fatalf("换了时间戳的同一进度也应该算重复：%d %d", rewards, duplicated)
	}
}

func TestRetryOnlyValidates(t *testing.T) {
	state := map[string]bool{}
	Apply(state, []Report{{Player: "p", Boss: 1, At: 1, Week: "w1"}})
	rewards, duplicated := Apply(state, []Report{{Player: "p", Boss: 1, At: 2, Retry: true, Week: "w1"}})
	if rewards != 0 || duplicated != 1 {
		t.Fatalf("重试命中已有进度应该跳过：%d %d", rewards, duplicated)
	}
}

func TestRetryNewProgressRewardsOnce(t *testing.T) {
	state := map[string]bool{}
	rewards, duplicated := Apply(state, []Report{{Player: "p", Boss: 1, At: 1, Retry: true, Week: "w1"}})
	if rewards != 1 || duplicated != 0 {
		t.Fatalf("重试新进度应该只发一次奖：%d %d", rewards, duplicated)
	}
	rewards, duplicated = Apply(state, []Report{{Player: "p", Boss: 1, At: 2, Retry: true, Week: "w1"}})
	if rewards != 0 || duplicated != 1 {
		t.Fatalf("再次重试应该算重复：%d %d", rewards, duplicated)
	}
}

func TestDifferentWeekRewards(t *testing.T) {
	rewards, _ := Apply(map[string]bool{}, []Report{
		{Player: "p", Boss: 1, At: 1, Week: "w1"},
		{Player: "p", Boss: 1, At: 2, Week: "w2"},
	})
	if rewards != 2 {
		t.Fatalf("不同周次应该各发一次奖：%d", rewards)
	}
}

func TestResetBoundary(t *testing.T) {
	cases := []struct {
		day       string
		hour, min int
		wantReset bool
	}{
		{"Mon", 4, 59, false},
		{"Mon", 5, 0, true},
		{"Mon", 5, 1, true},
		{"Mon", 23, 59, true},
		{"Sun", 23, 59, false},
		{"Tue", 5, 0, false},
	}
	for _, c := range cases {
		if got := ShouldResetAt(c.day, c.hour, c.min); got != c.wantReset {
			t.Fatalf("ShouldResetAt(%s, %d, %d) = %t，期望 %t", c.day, c.hour, c.min, got, c.wantReset)
		}
	}
}

func TestRepeatBatchSameResult(t *testing.T) {
	reports := Reconnect()
	rewards1, duplicated1 := Apply(map[string]bool{}, reports)
	rewards2, duplicated2 := Apply(map[string]bool{}, reports)
	if rewards1 != rewards2 || duplicated1 != duplicated2 {
		t.Fatalf("同一批上报重复处理结果应该相同：%d/%d vs %d/%d",
			rewards1, duplicated1, rewards2, duplicated2)
	}
}

// referenceApply 是参考实现：按唯一键去重后逐条发奖。
func referenceApply(reports []Report) (rewards int, duplicated int) {
	seen := map[string]bool{}
	for _, report := range reports {
		key := fmt.Sprintf("%s#%d#%s", report.Player, report.Boss, report.Week)
		if seen[key] {
			duplicated++
			continue
		}
		seen[key] = true
		rewards++
	}
	return rewards, duplicated
}

func TestApplyMatchesReference(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	players := []string{"p1", "p2", "p3", "p4", "p5"}
	weeks := []string{"w1", "w2"}
	for round := 0; round < 200; round++ {
		var reports []Report
		// 正常上报
		for i := 0; i < 1+rng.Intn(20); i++ {
			reports = append(reports, Report{
				Player: players[rng.Intn(len(players))],
				Boss:   1 + rng.Intn(4),
				At:     rng.Int63n(1000),
				Week:   weeks[rng.Intn(len(weeks))],
			})
		}
		// 断线重连重报：换时间戳重发
		for _, report := range reports {
			if rng.Intn(2) == 0 {
				dup := report
				dup.At = rng.Int63n(1000)
				reports = append(reports, dup)
			}
		}
		// 重试请求
		for _, report := range reports {
			if rng.Intn(3) == 0 {
				retry := report
				retry.At = rng.Int63n(1000)
				retry.Retry = true
				reports = append(reports, retry)
			}
		}
		rng.Shuffle(len(reports), func(i, j int) { reports[i], reports[j] = reports[j], reports[i] })

		gotRewards, gotDuplicated := Apply(map[string]bool{}, reports)
		wantRewards, wantDuplicated := referenceApply(reports)
		if gotRewards != wantRewards || gotDuplicated != wantDuplicated {
			t.Fatalf("第 %d 组与参考实现不一致：got %d/%d want %d/%d",
				round, gotRewards, gotDuplicated, wantRewards, wantDuplicated)
		}
	}
}

func TestApplyScales(t *testing.T) {
	reports := make([]Report, 0, 100000)
	for i := 0; i < 100000; i++ {
		reports = append(reports, Report{
			Player: fmt.Sprintf("p%d", i%1000),
			Boss:   i % 10,
			At:     int64(i),
			Week:   fmt.Sprintf("w%d", i%5),
		})
	}
	start := time.Now()
	rewards, duplicated := Apply(map[string]bool{}, reports)
	elapsed := time.Since(start)
	if rewards+duplicated != len(reports) {
		t.Fatalf("每条上报要么发奖要么算重复：%d + %d != %d", rewards, duplicated, len(reports))
	}
	if elapsed > time.Second {
		t.Fatalf("10 万条上报应在 1 秒内处理完，实际 %s", elapsed)
	}
}
