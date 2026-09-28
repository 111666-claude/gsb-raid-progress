package raidprogress

import "testing"

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
	if !ShouldReset("Tue") {
		t.Fatal("现在应该在周二重置")
	}
	if ShouldReset("Mon") {
		t.Fatal("现在周一不该重置")
	}
}
