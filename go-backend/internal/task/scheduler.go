package task

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/lijt/flux-panel/internal/model"
	"github.com/lijt/flux-panel/internal/pkg"
	"gorm.io/gorm"
)

// StartScheduler 啟動所有排程任務
func StartScheduler(db *gorm.DB) {
	slog.Info("排程任務啟動")

	// 每日 00:00:05 — 流量重置 + 過期處理
	go runDaily(db)

	// 每小時整點 — 流量統計
	go runHourly(db)
}

// ──────────────────── 每日任務 ────────────────────

func runDaily(db *gorm.DB) {
	// 算出距離下一個 00:00:05 的時間
	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 5, 0, now.Location())
	time.Sleep(time.Until(next))

	for {
		slog.Info("開始執行每日排程任務")
		safeRun("流量重置", func() { resetFlow(db) })
		safeRun("過期帳號處理", func() { expireUsers(db) })
		safeRun("過期隧道處理", func() { expireUserTunnels(db) })
		slog.Info("每日排程任務完成")

		// 等到明天 00:00:05
		now = time.Now()
		next = time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 5, 0, now.Location())
		time.Sleep(time.Until(next))
	}
}

// ──────────────────── 流量重置 ────────────────────

func resetFlow(db *gorm.DB) {
	now := time.Now()
	currentDay := now.Day()
	lastDay := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location()).Day()

	slog.Info("流量重置", "currentDay", currentDay, "lastDayOfMonth", lastDay)

	// 重置用戶流量
	resetUserFlow(db, currentDay, lastDay)
	// 重置用戶隧道流量
	resetUserTunnelFlow(db, currentDay, lastDay)
}

func resetUserFlow(db *gorm.DB, currentDay, lastDay int) {
	var users []model.User
	query := db.Where("flow_reset_time != 0")
	if currentDay == lastDay {
		query = query.Where("flow_reset_time = ? OR flow_reset_time > ?", currentDay, lastDay)
	} else {
		query = query.Where("flow_reset_time = ?", currentDay)
	}
	query.Find(&users)

	if len(users) == 0 {
		slog.Info("沒有需要重置流量的用戶")
		return
	}

	slog.Info("需要重置流量的用戶", "count", len(users))
	for _, u := range users {
		db.Model(&model.User{}).Where("id = ?", u.ID).
			Updates(map[string]interface{}{"in_flow": 0, "out_flow": 0})
		slog.Info("用戶流量已重置", "userID", u.ID, "resetDay", u.FlowResetTime)
	}
}

func resetUserTunnelFlow(db *gorm.DB, currentDay, lastDay int) {
	var tunnels []model.UserTunnel
	query := db.Where("flow_reset_time != 0")
	if currentDay == lastDay {
		query = query.Where("flow_reset_time = ? OR flow_reset_time > ?", currentDay, lastDay)
	} else {
		query = query.Where("flow_reset_time = ?", currentDay)
	}
	query.Find(&tunnels)

	if len(tunnels) == 0 {
		slog.Info("沒有需要重置流量的用戶隧道")
		return
	}

	slog.Info("需要重置流量的用戶隧道", "count", len(tunnels))
	for _, ut := range tunnels {
		db.Model(&model.UserTunnel{}).Where("id = ?", ut.ID).
			Updates(map[string]interface{}{"in_flow": 0, "out_flow": 0})
		slog.Info("用戶隧道流量已重置", "utID", ut.ID, "userID", ut.UserID, "tunnelID", ut.TunnelID)
	}
}

// ──────────────────── 過期帳號處理 ────────────────────

func expireUsers(db *gorm.DB) {
	now := time.Now().UnixMilli()
	var users []model.User
	db.Where("role_id != 0 AND status = 1 AND exp_time > 0 AND exp_time < ?", now).Find(&users)

	for _, user := range users {
		var forwards []model.Forward
		db.Where("user_id = ? AND status = 1", user.ID).Find(&forwards)
		for _, fwd := range forwards {
			var ut model.UserTunnel
			if err := db.Where("user_id = ? AND tunnel_id = ?", fwd.UserID, fwd.TunnelID).First(&ut).Error; err == nil {
				pauseForwardService(db, &fwd, ut.ID)
			}
			db.Model(&model.Forward{}).Where("id = ?", fwd.ID).Update("status", 0)
		}
		db.Model(&model.User{}).Where("id = ?", user.ID).Update("status", 0)
		slog.Info("用戶已過期，停用", "userID", user.ID)
	}
}

// ──────────────────── 過期隧道處理 ────────────────────

func expireUserTunnels(db *gorm.DB) {
	now := time.Now().UnixMilli()
	var uts []model.UserTunnel
	db.Where("status = 1 AND exp_time > 0 AND exp_time < ?", now).Find(&uts)

	for _, ut := range uts {
		var forwards []model.Forward
		db.Where("tunnel_id = ? AND user_id = ? AND status = 1", ut.TunnelID, ut.UserID).Find(&forwards)
		for _, fwd := range forwards {
			pauseForwardService(db, &fwd, ut.ID)
			db.Model(&model.Forward{}).Where("id = ?", fwd.ID).Update("status", 0)
		}
		db.Model(&model.UserTunnel{}).Where("id = ?", ut.ID).Update("status", 0)
		slog.Info("用戶隧道已過期，停用", "utID", ut.ID)
	}
}

func pauseForwardService(db *gorm.DB, fwd *model.Forward, userTunnelID int) {
	var tunnel model.Tunnel
	if err := db.First(&tunnel, fwd.TunnelID).Error; err != nil {
		return
	}
	name := buildServiceName(fwd.ID, fwd.UserID, userTunnelID)
	pkg.PauseService(tunnel.InNodeID, name)
	if tunnel.Type == 2 {
		pkg.PauseRemoteService(tunnel.OutNodeID, name)
	}
}

func buildServiceName(forwardID int64, userID int, userTunnelID int) string {
	return fmt.Sprintf("%d_%d_%d", forwardID, userID, userTunnelID)
}

// ──────────────────── 每小時任務 ────────────────────

func runHourly(db *gorm.DB) {
	// 算出距離下一個整點的時間
	now := time.Now()
	next := now.Truncate(time.Hour).Add(time.Hour)
	time.Sleep(time.Until(next))

	for {
		slog.Info("開始執行每小時流量統計")
		safeRun("流量統計", func() { statisticsFlow(db) })
		slog.Info("每小時流量統計完成")

		// 等到下一個整點
		now = time.Now()
		next = now.Truncate(time.Hour).Add(time.Hour)
		time.Sleep(time.Until(next))
	}
}

func statisticsFlow(db *gorm.DB) {
	now := time.Now()
	hourStr := now.Format("15:04")
	nowMs := now.UnixMilli()

	// 刪除 48 小時前的資料
	cutoffMs := nowMs - 48*60*60*1000
	db.Where("created_time < ?", cutoffMs).Delete(&model.StatisticsFlow{})

	// 遍歷所有用戶
	var users []model.User
	db.Find(&users)

	var records []model.StatisticsFlow
	for _, user := range users {
		currentFlow := user.InFlow + user.OutFlow

		// 查上次記錄
		var lastRecord model.StatisticsFlow
		incrementFlow := currentFlow
		if err := db.Where("user_id = ?", user.ID).Order("id DESC").First(&lastRecord).Error; err == nil {
			incrementFlow = currentFlow - lastRecord.TotalFlow
			if incrementFlow < 0 {
				incrementFlow = currentFlow
			}
		}

		records = append(records, model.StatisticsFlow{
			UserID:      user.ID,
			Flow:        incrementFlow,
			TotalFlow:   currentFlow,
			Time:        hourStr,
			CreatedTime: nowMs,
		})
	}

	if len(records) > 0 {
		db.CreateInBatches(&records, 100)
		slog.Info("流量統計寫入完成", "count", len(records))
	}
}

// ──────────────────── 工具 ────────────────────

func safeRun(name string, fn func()) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("排程任務 panic", "task", name, "error", r)
		}
	}()
	fn()
}
