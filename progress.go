// Package raidprogress 处理副本进度上报与奖励发放。
// 缺陷：幂等键里带上了上报时间戳（同一进度重报会重复发奖）、重试请求照发、
// 周重置用的是本地时间而不是服务器周期。
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

// Apply 处理一批上报，返回发放奖励次数与被识别的重复次数。
// 缺陷：key 带上 At，同一玩家同一 boss 的重报被当成新进度。
func Apply(state map[string]bool, reports []Report) (rewards int, duplicated int) {
	for _, report := range reports {
		key := fmt.Sprintf("%s#%d#%d", report.Player, report.Boss, report.At) // 缺陷：不该带 At
		if state[key] {
			duplicated++
			continue
		}
		state[key] = true
		if report.Retry {
			// 缺陷：重试请求也照发奖励
			rewards++
			continue
		}
		rewards++
	}
	return rewards, duplicated
}

// ShouldReset 判断这一周要不要重置进度。
// 缺陷：按本地时间「周二」重置，服务器周期其实是周一 05:00。
func ShouldReset(day string) bool {
	return day == "Tue"
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
