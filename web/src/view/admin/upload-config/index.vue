<template>
  <div class="upload-config-container">
    <el-card shadow="hover">
      <template #header>
        <div class="card-header">
          <span>文件上传配置</span>
          <el-button
            type="primary"
            :loading="saving"
            @click="saveConfig"
          >
            保存配置
          </el-button>
        </div>
      </template>

      <el-form
        ref="configFormRef"
        :model="uploadConfig"
        :rules="configRules"
        label-width="150px"
      >
        <!-- 基础配置 -->
        <el-divider content-position="left">
          基础配置
        </el-divider>
        
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item
              label="头像最大大小"
              prop="maxAvatarSize"
            >
              <el-input-number
                v-model="uploadConfig.maxAvatarSize"
                :min="1"
                :max="10"
                :step="0.5"
                :precision="1"
                style="width: 100%"
              />
              <span class="form-tip">MB，建议不超过5MB</span>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item
              label="文件最大大小"
              prop="maxFileSize"
            >
              <el-input-number
                v-model="uploadConfig.maxFileSize"
                :min="1"
                :max="100"
                :step="1"
                :precision="1"
                style="width: 100%"
              />
              <span class="form-tip">MB，建议不超过50MB</span>
            </el-form-item>
          </el-col>
        </el-row>

        <el-form-item
          label="上传目录"
          prop="uploadDir"
        >
          <el-input
            v-model="uploadConfig.uploadDir"
            placeholder="上传文件存储目录"
          />
          <span class="form-tip">服务器上的文件存储路径</span>
        </el-form-item>

        <!-- 安全配置 -->
        <el-divider content-position="left">
          安全配置
        </el-divider>

        <el-form-item label="启用安全扫描">
          <el-switch
            v-model="uploadConfig.enableSecurityScan"
            active-text="开启"
            inactive-text="关闭"
          />
          <span class="form-tip">扫描文件内容以检测恶意代码</span>
        </el-form-item>

        <el-form-item label="允许的文件类型">
          <el-select
            v-model="uploadConfig.allowedTypes"
            multiple
            filterable
            allow-create
            placeholder="选择或添加允许的MIME类型"
            style="width: 100%"
          >
            <el-option
              v-for="type in commonMimeTypes"
              :key="type"
              :label="type"
              :value="type"
            />
          </el-select>
          <span class="form-tip">MIME类型白名单，如：image/jpeg</span>
        </el-form-item>

        <el-form-item label="禁止的文件扩展名">
          <el-select
            v-model="uploadConfig.deniedExts"
            multiple
            filterable
            allow-create
            placeholder="选择或添加禁止的文件扩展名"
            style="width: 100%"
          >
            <el-option
              v-for="ext in commonDangerousExts"
              :key="ext"
              :label="ext"
              :value="ext"
            />
          </el-select>
          <span class="form-tip">危险文件扩展名黑名单，如：.exe</span>
        </el-form-item>

        <!-- 存储管理 -->
        <el-divider content-position="left">
          存储管理
        </el-divider>

        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item
              label="清理间隔"
              prop="cleanupInterval"
            >
              <el-input-number
                v-model="uploadConfig.cleanupInterval"
                :min="1"
                :max="168"
                style="width: 100%"
              />
              <span class="form-tip">小时，自动清理过期文件的间隔</span>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item
              label="文件保留天数"
              prop="retentionDays"
            >
              <el-input-number
                v-model="uploadConfig.retentionDays"
                :min="1"
                :max="365"
                style="width: 100%"
              />
              <span class="form-tip">天，超过此天数的文件将被清理</span>
            </el-form-item>
          </el-col>
        </el-row>

        <!-- 操作按钮 -->
        <el-form-item>
          <el-button
            type="danger"
            :loading="cleaning"
            @click="cleanupNow"
          >
            立即清理过期文件
          </el-button>
          <el-button
            type="warning"
            @click="resetToDefault"
          >
            恢复默认配置
          </el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- 统计信息 -->
    <el-card
      shadow="hover"
      style="margin-top: 20px;"
    >
      <template #header>
        <div class="card-header">
          <span>存储统计</span>
          <el-button
            size="small"
            :loading="false"
            @click="loadStats"
          >
            刷新统计
          </el-button>
        </div>
      </template>
      
      <el-row :gutter="20">
        <el-col :span="4">
          <el-statistic
            title="总文件数"
            :value="stats.totalFiles"
          />
        </el-col>
        <el-col :span="4">
          <el-statistic
            title="总存储大小"
            :value="stats.totalSize"
            suffix="MB"
          />
        </el-col>
        <el-col :span="4">
          <el-statistic
            title="今日上传"
            :value="stats.todayUploads"
          />
        </el-col>
        <el-col :span="4">
          <el-statistic
            title="可清理文件"
            :value="stats.cleanableFiles"
          />
        </el-col>
        <el-col :span="4">
          <el-statistic
            title="头像文件数"
            :value="stats.avatarCount"
          />
        </el-col>
        <el-col :span="4">
          <el-statistic
            title="平均文件大小"
            :value="stats.avgFileSize"
            suffix="KB"
          />
        </el-col>
      </el-row>
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { 
  getUploadConfig, 
  updateUploadConfig, 
  getUploadStats, 
  cleanupExpiredFiles 
} from '@/api/admin'

const configFormRef = ref()
const saving = ref(false)
const cleaning = ref(false)

// 上传配置
const uploadConfig = reactive({
  maxAvatarSize: 2,
  maxFileSize: 10,
  uploadDir: './uploads',
  enableSecurityScan: true,
  allowedTypes: ['image/jpeg', 'image/png', 'image/webp', 'image/gif'],
  deniedExts: ['.exe', '.bat', '.cmd', '.com', '.scr', '.pif', '.msi', '.dll'],
  cleanupInterval: 24,
  retentionDays: 30
})

// 常用MIME类型
const commonMimeTypes = [
  'image/jpeg',
  'image/png', 
  'image/webp',
  'image/gif',
  'image/svg+xml',
  'text/plain',
  'application/pdf',
  'application/json',
  'application/zip'
]

// 常见危险扩展名
const commonDangerousExts = [
  '.exe', '.bat', '.cmd', '.com', '.scr', '.pif', '.msi', '.dll',
  '.sh', '.bash', '.zsh', '.fish', '.ps1', '.vbs', '.js', '.jar',
  '.php', '.asp', '.jsp', '.py', '.rb', '.pl', '.cgi', '.htaccess'
]

// 表单验证规则
const configRules = {
  maxAvatarSize: [
    { required: true, message: '请设置头像最大大小', trigger: 'blur' },
    { type: 'number', min: 0.5, max: 10, message: '大小应在0.5-10MB之间', trigger: 'blur' }
  ],
  maxFileSize: [
    { required: true, message: '请设置文件最大大小', trigger: 'blur' },
    { type: 'number', min: 1, max: 100, message: '大小应在1-100MB之间', trigger: 'blur' }
  ],
  uploadDir: [
    { required: true, message: '请设置上传目录', trigger: 'blur' }
  ]
}

// 统计信息
const stats = reactive({
  totalFiles: 0,
  totalSize: 0,
  todayUploads: 0,
  cleanableFiles: 0,
  avatarCount: 0,
  avgFileSize: 0
})

// 加载配置
const loadConfig = async () => {
  try {
    const response = await getUploadConfig()
    if (response.code === 0 || response.code === 200) {
      const config = response.data
      // 转换字节到MB显示
      uploadConfig.maxAvatarSize = config.maxAvatarSize / (1024 * 1024)
      uploadConfig.maxFileSize = config.maxFileSize / (1024 * 1024)
      uploadConfig.uploadDir = config.uploadDir
      uploadConfig.enableSecurityScan = config.enableSecurityScan
      uploadConfig.allowedTypes = config.allowedTypes || []
      uploadConfig.deniedExts = config.deniedExts || []
      uploadConfig.cleanupInterval = config.cleanupInterval
      uploadConfig.retentionDays = config.retentionDays
    }
  } catch (error) {
    console.error('加载配置失败:', error)
    ElMessage.error('加载配置失败')
  }
}

// 保存配置
const saveConfig = async () => {
  if (!configFormRef.value) return
  
  try {
    await configFormRef.value.validate()
    saving.value = true
    
    // 转换单位（MB -> bytes）
    const configData = {
      ...uploadConfig,
      maxAvatarSize: uploadConfig.maxAvatarSize * 1024 * 1024,
      maxFileSize: uploadConfig.maxFileSize * 1024 * 1024
    }
    
    const response = await updateUploadConfig(configData)
    if (response.code === 0 || response.code === 200) {
      ElMessage.success('配置保存成功')
    } else {
      ElMessage.error(response.msg || '保存配置失败')
    }
  } catch (error) {
    console.error('保存配置失败:', error)
    if (error !== false) {
      ElMessage.error('保存配置失败')
    }
  } finally {
    saving.value = false
  }
}

// 立即清理
const cleanupNow = async () => {
  try {
    await ElMessageBox.confirm(
      '确定要立即清理过期文件吗？此操作不可恢复。头像文件不会被清理。',
      '确认清理',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    
    cleaning.value = true
    
    const response = await cleanupExpiredFiles()
    if (response.code === 0 || response.code === 200) {
      const result = response.data
      ElMessage.success(`文件清理完成，清理了 ${result.cleaned_files} 个文件`)
      loadStats() // 重新加载统计
    } else {
      ElMessage.error(response.msg || '文件清理失败')
    }
  } catch (error) {
    console.error('文件清理失败:', error)
    if (error !== 'cancel') {
      ElMessage.error('文件清理失败')
    }
  } finally {
    cleaning.value = false
  }
}

// 恢复默认配置
const resetToDefault = () => {
  uploadConfig.maxAvatarSize = 2
  uploadConfig.maxFileSize = 10
  uploadConfig.uploadDir = './uploads'
  uploadConfig.enableSecurityScan = true
  uploadConfig.allowedTypes = ['image/jpeg', 'image/png', 'image/webp', 'image/gif']
  uploadConfig.deniedExts = ['.exe', '.bat', '.cmd', '.com', '.scr', '.pif', '.msi', '.dll']
  uploadConfig.cleanupInterval = 24
  uploadConfig.retentionDays = 30
  
  ElMessage.success('已恢复默认配置')
}

// 加载统计信息
const loadStats = async () => {
  try {
    const response = await getUploadStats()
    if (response.code === 0 || response.code === 200) {
      const data = response.data
      stats.totalFiles = data.total_files
      stats.totalSize = (data.total_size / (1024 * 1024)).toFixed(2) // 转换为MB并保留2位小数
      stats.todayUploads = data.today_uploads
      stats.cleanableFiles = data.cleanable_files
      stats.avatarCount = data.avatar_count
      stats.avgFileSize = (data.avg_file_size / 1024).toFixed(2) // 转换为KB并保留2位小数
    } else {
      ElMessage.error(response.msg || '获取统计信息失败')
    }
  } catch (error) {
    console.error('加载统计信息失败:', error)
    ElMessage.error('加载统计信息失败')
  }
}

onMounted(() => {
  loadConfig()
  loadStats()
})
</script>

<style scoped>
.upload-config-container {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.form-tip {
  font-size: 12px;
  color: #999;
  margin-left: 10px;
}

.el-divider {
  margin: 30px 0 20px 0;
}

.el-statistic {
  text-align: center;
}
</style>
