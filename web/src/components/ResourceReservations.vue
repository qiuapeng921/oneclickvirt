// 前端资源预留状态显示组件示例
// 用于展示用户当前的资源预留情况

<template>
  <div class="resource-reservations">
    <el-card class="reservation-card">
      <template #header>
        <div class="card-header">
          <span>资源预留状态</span>
          <el-button 
            type="text" 
            :loading="loading"
            @click="refreshReservations"
          >
            刷新
          </el-button>
        </div>
      </template>
      
      <div
        v-if="reservations.length === 0"
        class="no-reservations"
      >
        <el-empty description="暂无资源预留" />
      </div>
      
      <div v-else>
        <el-table
          :data="reservations"
          stripe
        >
          <el-table-column
            prop="providerName"
            label="节点"
            width="120"
          />
          <el-table-column
            prop="instanceType"
            label="类型"
            width="80"
          >
            <template #default="scope">
              <el-tag 
                :type="scope.row.instanceType === 'vm' ? 'primary' : 'success'"
                size="small"
              >
                {{ scope.row.instanceType === 'vm' ? '虚拟机' : '容器' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column
            label="资源配置"
            width="200"
          >
            <template #default="scope">
              <div class="resource-config">
                <div>CPU: {{ scope.row.cpu }} 核</div>
                <div>内存: {{ scope.row.memory }}MB</div>
                <div>磁盘: {{ scope.row.disk }}MB</div>
                <div>带宽: {{ scope.row.bandwidth }}Mbps</div>
              </div>
            </template>
          </el-table-column>
          <el-table-column
            prop="status"
            label="状态"
            width="80"
          >
            <template #default="scope">
              <el-tag 
                :type="getStatusType(scope.row.status)"
                size="small"
              >
                {{ getStatusText(scope.row.status) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column
            label="过期时间"
            width="150"
          >
            <template #default="scope">
              <div class="expiry-time">
                <div>{{ formatDateTime(scope.row.expiresAt) }}</div>
                <div
                  class="countdown"
                  :class="getCountdownClass(scope.row.expiresAt)"
                >
                  {{ getTimeRemaining(scope.row.expiresAt) }}
                </div>
              </div>
            </template>
          </el-table-column>
          <el-table-column
            prop="createdAt"
            label="创建时间"
            width="150"
          >
            <template #default="scope">
              {{ formatDateTime(scope.row.createdAt) }}
            </template>
          </el-table-column>
        </el-table>
        
        <div
          v-if="totalReserved"
          class="reservation-summary"
        >
          <el-divider content-position="left">
            预留资源统计
          </el-divider>
          <el-row :gutter="16">
            <el-col :span="6">
              <el-statistic
                title="预留CPU"
                :value="totalReserved.cpu"
                suffix="核"
              />
            </el-col>
            <el-col :span="6">
              <el-statistic
                title="预留内存"
                :value="totalReserved.memory"
                suffix="MB"
              />
            </el-col>
            <el-col :span="6">
              <el-statistic
                title="预留磁盘"
                :value="totalReserved.disk"
                suffix="MB"
              />
            </el-col>
            <el-col :span="6">
              <el-statistic
                title="预留带宽"
                :value="totalReserved.bandwidth"
                suffix="Mbps"
              />
            </el-col>
          </el-row>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { getActiveReservations } from '@/api/user'

const reservations = ref([])
const loading = ref(false)

// 获取活跃预留列表
const fetchReservations = async () => {
  try {
    loading.value = true
    const response = await getActiveReservations()
    reservations.value = response.data || []
  } catch (error) {
    console.error('获取资源预留失败:', error)
    ElMessage.error('获取资源预留失败')
  } finally {
    loading.value = false
  }
}

// 刷新预留列表
const refreshReservations = () => {
  fetchReservations()
}

// 计算总预留资源
const totalReserved = computed(() => {
  if (reservations.value.length === 0) return null
  
  return reservations.value.reduce((total, reservation) => {
    return {
      cpu: total.cpu + reservation.cpu,
      memory: total.memory + reservation.memory,
      disk: total.disk + reservation.disk,
      bandwidth: total.bandwidth + reservation.bandwidth
    }
  }, { cpu: 0, memory: 0, disk: 0, bandwidth: 0 })
})

// 状态类型映射
const getStatusType = (status) => {
  const statusMap = {
    'reserved': 'warning',
    'consumed': 'success',
    'released': 'info',
    'expired': 'danger'
  }
  return statusMap[status] || 'info'
}

// 状态文本映射
const getStatusText = (status) => {
  const statusMap = {
    'reserved': '预留中',
    'consumed': '已消费',
    'released': '已释放',
    'expired': '已过期'
  }
  return statusMap[status] || status
}

// 格式化日期时间
const formatDateTime = (dateTime) => {
  return new Date(dateTime).toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

// 计算剩余时间
const getTimeRemaining = (expiresAt) => {
  const now = new Date()
  const expiry = new Date(expiresAt)
  const diff = expiry - now
  
  if (diff <= 0) {
    return '已过期'
  }
  
  const minutes = Math.floor(diff / (1000 * 60))
  const hours = Math.floor(minutes / 60)
  const days = Math.floor(hours / 24)
  
  if (days > 0) {
    return `${days}天${hours % 24}小时`
  } else if (hours > 0) {
    return `${hours}小时${minutes % 60}分钟`
  } else {
    return `${minutes}分钟`
  }
}

// 获取倒计时样式类
const getCountdownClass = (expiresAt) => {
  const now = new Date()
  const expiry = new Date(expiresAt)
  const diff = expiry - now
  const minutes = Math.floor(diff / (1000 * 60))
  
  if (minutes <= 0) {
    return 'expired'
  } else if (minutes <= 5) {
    return 'warning'
  } else if (minutes <= 15) {
    return 'attention'
  } else {
    return 'normal'
  }
}

onMounted(() => {
  fetchReservations()
  
  // 每分钟刷新一次，更新倒计时
  const interval = setInterval(() => {
    // 只更新时间显示，不重新请求数据
    reservations.value = [...reservations.value]
  }, 60000)
  
  // 每5分钟刷新一次数据
  const dataInterval = setInterval(fetchReservations, 5 * 60 * 1000)
  
  // 组件卸载时清理定时器
  onUnmounted(() => {
    clearInterval(interval)
    clearInterval(dataInterval)
  })
})
</script>

<style scoped>
.resource-reservations {
  margin: 20px 0;
}

.reservation-card {
  border-radius: 8px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.no-reservations {
  text-align: center;
  padding: 40px 0;
}

.resource-config {
  font-size: 12px;
  line-height: 1.4;
}

.expiry-time {
  font-size: 12px;
}

.countdown {
  font-weight: bold;
  margin-top: 4px;
}

.countdown.normal {
  color: #67C23A;
}

.countdown.attention {
  color: #E6A23C;
}

.countdown.warning {
  color: #F56C6C;
}

.countdown.expired {
  color: #909399;
}

.reservation-summary {
  margin-top: 20px;
}
</style>
