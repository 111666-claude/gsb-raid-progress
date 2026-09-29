// Package raidprogress 处理副本进度上报与奖励发放。
// 幂等键为 (玩家, boss, 周次)，与数据库唯一索引一致；重试只做幂等校验；
// 周重置按服务器周期（周一 05:00），与本地时间无关。
package raidprogress

import (
	"fmt"
)

// Report 是一次进度上报。
type Report struct {
	Player string
	Boss   int
	At     int64
	Retry  bool
	Week   string
}

// Key 返回上报的幂等键：(玩家, boss, 周次)，与库里的唯一索引一致。
// 键里不含上报时间戳，同一进度的任何重复上报都命中同一个键。
func Key(report Report) string {
	return fmt.Sprintf("%s#%d#%s", report.Player, report.Boss, report.Week)
}

// Apply 处理一批上报，返回发放奖励次数与被识别的重复次数。
// 每个 (玩家, boss, 周次) 最多发一次奖；重试（Retry=true）只做幂等校验，
// 命中已有进度就跳过，是新进度也只发一次。
// 时间 O(len(reports))，内存 O(键数)，不做全表扫描。
func Apply(state map[string]bool, reports []Report) (rewards int, duplicated int) {
	for _, report := range reports {
		key := Key(report)
		if state[key] {
			duplicated++
			continue
		}
		state[key] = true
		rewards++
	}
	return rewards, duplicated
}

// ShouldReset 判断服务器时间某一天是否处于本周重置点（周一 05:00）及之后。
// 与本地时间无关，只看服务器周期。
func ShouldReset(day string) bool {
	return ShouldResetAt(day, 5, 0)
}

// ShouldResetAt 判断服务器时间 day 的 hour:min 是否已过本周重置点（周一 05:00）。
// 周一 05:00 之前的上报仍属于上一周。
func ShouldResetAt(day string, hour, min int) bool {
	if day != "Mon" {
		return false
	}
	return hour > 5 || (hour == 5 && min >= 0)
}

// Reconnect 样本：同一进度先上报 3 次，断线重连后又用新时间戳重报 3 次。
func Reconnect() []Report {
	reports := []Report{}
	for boss := 1; boss <= 3; boss++ {
		reports = append(reports, Report{Player: "p1", Boss: boss, At: 100, Week: "w1"})
	}
	for boss := 1; boss <= 3; boss++ {
		reports = append(reports, Report{Player: "p1", Boss: boss, At: 200, Retry: true, Week: "w1"})
	}
	return reports
}
