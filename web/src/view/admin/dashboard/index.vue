<template>
  <div class="admin-dashboard">
    <div class="dashboard-header">
      <h1>管理员仪表盘</h1>
      <p>系统概览与监控</p>
    </div>

    <!-- 统计卡片 -->
    <el-row
      :gutter="20"
      class="stats-row"
    >
      <el-col :span="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-icon user-icon">
              <el-icon><User /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-number">
                {{ dashboardData.totalUsers }}
              </div>
              <div class="stat-label">
                总用户数
              </div>
            </div>
          </div>
        </el-card>
      </el-col>
      
      <el-col :span="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-icon server-icon">
              <el-icon><Monitor /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-number">
                {{ dashboardData.totalProviders }}
              </div>
              <div class="stat-label">
                服务器数量
              </div>
            </div>
          </div>
        </el-card>
      </el-col>
      
      <el-col :span="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-icon vm-icon">
              <el-icon><Box /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-number">
                {{ dashboardData.totalVMs }}
              </div>
              <div class="stat-label">
                虚拟机数量
              </div>
            </div>
          </div>
        </el-card>
      </el-col>
      
      <el-col :span="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-icon container-icon">
              <el-icon><Grid /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-number">
                {{ dashboardData.totalContainers }}
              </div>
              <div class="stat-label">
                容器数量
              </div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { User, Monitor, Box, Grid } from '@element-plus/icons-vue'
import { getAdminDashboard } from '@/api/admin'

const dashboardData = reactive({
  totalUsers: 0,
  totalProviders: 0,
  totalVMs: 0,
  totalContainers: 0
})

const fetchDashboardData = async () => {
  try {
    const response = await getAdminDashboard()
    if (response.code === 0 || response.code === 200) {
      // 数据在 response.data.statistics 中
      if (response.data && response.data.statistics) {
        Object.assign(dashboardData, response.data.statistics)
      } else {
        // 兼容旧格式，数据直接在 response.data 中
        Object.assign(dashboardData, response.data)
      }
    }
  } catch (error) {
    ElMessage.error('获取仪表盘数据失败')
    console.error('Dashboard data fetch error:', error)
  }
}

onMounted(async () => {
  await fetchDashboardData()
})
</script>

<style scoped>
.admin-dashboard {
  padding: 20px;
}

.dashboard-header {
  margin-bottom: 30px;
}

.dashboard-header h1 {
  margin: 0 0 10px 0;
  color: #333;
  font-size: 28px;
  font-weight: 600;
}

.dashboard-header p {
  margin: 0;
  color: #666;
  font-size: 16px;
}

.stats-row {
  margin-bottom: 30px;
}

.stat-card {
  height: 120px;
}

.stat-content {
  display: flex;
  align-items: center;
  height: 100%;
}

.stat-icon {
  width: 60px;
  height: 60px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 20px;
  font-size: 24px;
  color: white;
}

.user-icon {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.server-icon {
  background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
}

.vm-icon {
  background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
}

.container-icon {
  background: linear-gradient(135deg, #43e97b 0%, #38f9d7 100%);
}

.stat-info {
  flex: 1;
}

.stat-number {
  font-size: 32px;
  font-weight: bold;
  color: #333;
  line-height: 1;
}

.stat-label {
  font-size: 14px;
  color: #666;
  margin-top: 5px;
}
</style>