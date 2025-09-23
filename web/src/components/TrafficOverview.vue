<template>
  <div class="traffic-overview">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>流量使用统计</span>
          <el-button
            size="small"
            :loading="loading"
            @click="loadTrafficData"
          >
            <el-icon><Refresh /></el-icon>
            刷新
          </el-button>
        </div>
      </template>

      <div v-if="loading" class="loading-container">
        <el-skeleton :rows="3" animated />
      </div>

      <div v-else-if="trafficData" class="traffic-content">
        <!-- 数据源指示 -->
        <div class="data-source-indicator">
          <el-tag 
            :type="trafficData.vnstat_available ? 'success' : 'warning'"
            size="small"
          >
            {{ trafficData.vnstat_available ? 'vnStat实时数据' : '基础数据' }}
          </el-tag>
        </div>

        <!-- 流量使用进度 -->
        <div class="traffic-usage">
          <div class="usage-header">
            <span class="usage-title">本月流量使用</span>
            <span class="usage-values">
              {{ formatTraffic(trafficData.current_month_usage) }} / 
              {{ formatTraffic(trafficData.total_limit) }}
            </span>
          </div>
          <el-progress 
            :percentage="Math.min(trafficData.usage_percent || 0, 100)"
            :color="getProgressColor(trafficData.usage_percent || 0)"
            :stroke-width="12"
            :status="trafficData.is_limited ? 'exception' : undefined"
          />
          <div class="usage-info">
            <span class="usage-percent">{{ (trafficData.usage_percent || 0).toFixed(2) }}%</span>
            <span 
              v-if="trafficData.is_limited" 
              class="limit-warning"
            >
              <el-icon><Warning /></el-icon>
              流量已超限
            </span>
          </div>
        </div>

        <!-- 重置时间 -->
        <div v-if="trafficData.reset_time" class="reset-info">
          <el-text type="info" size="small">
            <el-icon><Clock /></el-icon>
            流量重置时间: {{ formatDate(trafficData.reset_time) }}
          </el-text>
        </div>

        <!-- vnStat详细数据 -->
        <div v-if="trafficData.vnstat_available && showDetails" class="vnstat-details">
          <el-divider content-position="left">详细统计</el-divider>
          <div class="details-grid">
            <div class="detail-item">
              <span class="detail-label">今日使用</span>
              <span class="detail-value">{{ formatTraffic(trafficData.today_usage || 0) }}</span>
            </div>
            <div class="detail-item">
              <span class="detail-label">本周使用</span>
              <span class="detail-value">{{ formatTraffic(trafficData.week_usage || 0) }}</span>
            </div>
            <div class="detail-item">
              <span class="detail-label">历史总量</span>
              <span class="detail-value">{{ formatTraffic(trafficData.alltime_usage || 0) }}</span>
            </div>
          </div>
        </div>

        <!-- 展开/收起按钮 -->
        <div v-if="trafficData.vnstat_available" class="toggle-details">
          <el-button
            text
            size="small"
            @click="showDetails = !showDetails"
          >
            {{ showDetails ? '收起详情' : '查看详情' }}
            <el-icon><ArrowDown v-if="!showDetails" /><ArrowUp v-else /></el-icon>
          </el-button>
        </div>
      </div>

      <div v-else class="error-state">
        <el-empty description="暂无流量数据" />
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { getUserTrafficOverview } from '@/api/user'
import { ElMessage } from 'element-plus'
import { Refresh, Warning, Clock, ArrowDown, ArrowUp } from '@element-plus/icons-vue'

const loading = ref(false)
const trafficData = ref(null)
const showDetails = ref(false)

const loadTrafficData = async () => {
  loading.value = true
  try {
    const response = await getUserTrafficOverview()
    if (response.code === 0) {
      trafficData.value = response.data
    } else {
      ElMessage.error(`获取流量数据失败: ${response.msg}`)
    }
  } catch (error) {
    console.error('获取流量数据失败:', error)
    ElMessage.error('获取流量数据失败，请稍后重试')
  } finally {
    loading.value = false
  }
}

const formatTraffic = (mb) => {
  if (!mb || mb === 0) return '0 B'
  
  const GB_IN_MB = 1024
  const TB_IN_MB = 1024 * 1024
  
  if (mb >= TB_IN_MB) {
    return `${(mb / TB_IN_MB).toFixed(2)} TB`
  } else if (mb >= GB_IN_MB) {
    return `${(mb / GB_IN_MB).toFixed(2)} GB`
  } else if (mb >= 1) {
    return `${mb.toFixed(2)} MB`
  } else if (mb > 0) {
    return `${(mb * 1024).toFixed(2)} KB`
  }
  return '0 B'
}

const getProgressColor = (percentage) => {
  if (percentage < 60) return '#67c23a'
  if (percentage < 80) return '#e6a23c'
  return '#f56c6c'
}

const formatDate = (dateString) => {
  if (!dateString) return '未设置'
  return new Date(dateString).toLocaleString('zh-CN')
}

onMounted(() => {
  loadTrafficData()
})
</script>

<style scoped>
.traffic-overview {
  margin-bottom: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.loading-container {
  padding: 20px;
}

.traffic-content {
  padding: 10px 0;
}

.data-source-indicator {
  margin-bottom: 15px;
}

.traffic-usage {
  margin-bottom: 15px;
}

.usage-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
}

.usage-title {
  font-weight: 500;
  color: var(--el-text-color-primary);
}

.usage-values {
  font-family: monospace;
  font-size: 14px;
  color: var(--el-text-color-regular);
}

.usage-info {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 8px;
}

.usage-percent {
  font-weight: 500;
  color: var(--el-text-color-primary);
}

.limit-warning {
  color: var(--el-color-danger);
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
}

.reset-info {
  margin-bottom: 15px;
  text-align: center;
}

.vnstat-details {
  margin-top: 15px;
}

.details-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
  gap: 15px;
  margin-top: 10px;
}

.detail-item {
  text-align: center;
  padding: 10px;
  background: var(--el-fill-color-lighter);
  border-radius: 6px;
}

.detail-label {
  display: block;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-bottom: 5px;
}

.detail-value {
  display: block;
  font-weight: 500;
  font-family: monospace;
  color: var(--el-text-color-primary);
}

.toggle-details {
  text-align: center;
  margin-top: 15px;
}

.error-state {
  padding: 20px;
  text-align: center;
}
</style>
