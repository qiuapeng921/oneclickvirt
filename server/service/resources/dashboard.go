package resources

import (
	"oneclickvirt/global"
	"oneclickvirt/model/dashboard"
	"oneclickvirt/model/provider"
	"oneclickvirt/model/user"

	"go.uber.org/zap"
)

type DashboardService struct{}

func (s *DashboardService) GetDashboardStats() (*dashboard.DashboardStats, error) {
	global.APP_LOG.Debug("获取Dashboard统计信息")

	regionStats, err := s.getRegionStats()
	if err != nil {
		global.APP_LOG.Error("获取地区统计失败", zap.Error(err))
		return nil, err
	}

	quotaStats, err := s.getQuotaStats()
	if err != nil {
		global.APP_LOG.Error("获取配额统计失败", zap.Error(err))
		return nil, err
	}

	userStats, err := s.getUserStats()
	if err != nil {
		global.APP_LOG.Error("获取用户统计失败", zap.Error(err))
		return nil, err
	}

	global.APP_LOG.Debug("Dashboard统计信息获取成功",
		zap.Int("regionCount", len(regionStats)),
		zap.Int("totalUsers", userStats.TotalUsers))
	return &dashboard.DashboardStats{
		RegionStats: regionStats,
		QuotaStats:  *quotaStats,
		UserStats:   *userStats,
	}, nil
}

func (s *DashboardService) getRegionStats() ([]dashboard.RegionStat, error) {
	var providers []provider.Provider
	if err := global.APP_DB.Find(&providers).Error; err != nil {
		return nil, err
	}

	regionMap := make(map[string]*dashboard.RegionStat)

	for _, p := range providers {
		if regionMap[p.Region] == nil {
			regionMap[p.Region] = &dashboard.RegionStat{
				Region: p.Region,
				Count:  0,
				Used:   0,
				Total:  0,
			}
		}
		regionMap[p.Region].Count++
		regionMap[p.Region].Used += p.UsedQuota
		regionMap[p.Region].Total += p.TotalQuota
	}

	var regionStats []dashboard.RegionStat
	for _, stat := range regionMap {
		regionStats = append(regionStats, *stat)
	}

	return regionStats, nil
}

func (s *DashboardService) getQuotaStats() (*dashboard.QuotaStat, error) {
	var totalQuota, usedQuota int64

	global.APP_DB.Model(&provider.Provider{}).Select("COALESCE(SUM(total_quota), 0)").Scan(&totalQuota)
	global.APP_DB.Model(&provider.Provider{}).Select("COALESCE(SUM(used_quota), 0)").Scan(&usedQuota)

	return &dashboard.QuotaStat{
		Used:      int(usedQuota),
		Available: int(totalQuota - usedQuota),
		Total:     int(totalQuota),
	}, nil
}

func (s *DashboardService) getUserStats() (*dashboard.UserStat, error) {
	var totalUsers, activeUsers, adminUsers int64

	global.APP_DB.Model(&user.User{}).Count(&totalUsers)
	global.APP_DB.Model(&user.User{}).Where("status = ?", 1).Count(&activeUsers)
	global.APP_DB.Model(&user.User{}).Where("user_type = ?", "admin").Count(&adminUsers)

	return &dashboard.UserStat{
		TotalUsers:  int(totalUsers),
		ActiveUsers: int(activeUsers),
		AdminUsers:  int(adminUsers),
	}, nil
}
