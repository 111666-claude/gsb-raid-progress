// Package raidprogress 处理副本进度上报与奖励发放。
// 幂等键为 (玩家, boss, 周次)，与数据库唯一索引一致；重试请求只做幂等校验；
// 周重置按服务器周期（周一 05:00），与本地时间无关。
package raidprogress

import (
	"fmt"
	"time"
)

// serverZone 是服务器周期使用的固定时区（UTC+8），不依赖运行环境的本地时间。
var serverZone = time.FixedZone("CST", 8*3600)

// Report 是一次进度上报。
type Report struct {
	Player string
	Boss   int
	At     int64
	Retry  bool
	Week   string
}

// key 返回幂等键：玩家 + boss + 周次。不带上报时间戳，
// 同一进度的任何重复上报（重连重报、重试）都映射到同一个键。
func key(report Report) string {
	return fmt.Sprintf("%s#%d#%s", report.Player, report.Boss, report.Week)
}

// Apply 处理一批上报，返回发放奖励次数与被识别的重复次数。
// 每个 (玩家, boss, 周次) 最多发一次奖；Retry=true 的请求只做幂等校验，
// 命中即跳过，未命中也只发一次奖。state 以幂等键为索引，单条上报 O(1)。
func Apply(state map[string]bool, reports []Report) (rewards int, duplicated int) {
	for _, report := range reports {
		k := key(report)
		if state[k] {
			duplicated++
			continue
		}
		state[k] = true
		rewards++
	}
	return rewards, duplicated
}

// ShouldReset 判断指定星期几是否为服务器的周重置日（周一）。
func ShouldReset(day string) bool {
	return day == "Mon"
}

// ShouldResetAt 判断服务器时间 t 是否已过本周重置点（周一 05:00）且仍在周一。
// 换算到固定的服务器时区再判断，与运行环境的本地时间无关。
func ShouldResetAt(t time.Time) bool {
	st := t.In(serverZone)
	return st.Weekday() == time.Monday && st.Hour() >= 5
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
