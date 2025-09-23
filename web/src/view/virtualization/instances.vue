<template>
  <div class="instance-management">
    <!-- 页面标题 -->
    <div class="page-header">
      <h1>实例管理</h1>
      <div class="header-actions">
        <el-select
          v-model="selectedProvider"
          placeholder="选择虚拟化类型"
          @change="loadInstances"
        >
          <el-option
            label="Docker"
            value="docker"
          />
          <el-option
            label="LXD"
            value="lxd"
          />
          <el-option
            label="Incus"
            value="incus"
          />
          <el-option
            label="Proxmox VE"
            value="proxmox"
          />
        </el-select>
        <el-button
          type="primary"
          @click="showCreateDialog"
        >
          <el-icon><Plus /></el-icon>
          创建实例 (管理员)
        </el-button>
        <el-button
          type="success"
          @click="goToUserApply"
        >
          <el-icon><Plus /></el-icon>
          安全创建实例
        </el-button>
        <el-button
          :loading="loading"
          @click="loadInstances"
        >
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <!-- 实例列表 -->
    <div class="instance-grid">
      <div
        v-for="instance in instances"
        :key="instance.id"
        class="instance-card"
      >
        <div class="instance-header">
          <div class="instance-info">
            <h3>{{ instance.name }}</h3>
            <el-tag
              :type="getStatusColor(instance.status)"
              size="small"
            >
              {{ getStatusText(instance.status) }}
            </el-tag>
          </div>
          <div class="instance-type">
            <el-tag
              type="info"
              size="small"
            >
              {{ instance.type }}
            </el-tag>
          </div>
        </div>
        
        <div class="instance-details">
          <div class="detail-row">
            <span class="label">ID:</span>
            <span class="value">{{ instance.id }}</span>
          </div>
          <div
            v-if="instance.privateIP"
            class="detail-row"
          >
            <span class="label">内网IP:</span>
            <span class="value">{{ instance.privateIP }}</span>
          </div>
          <div
            v-if="instance.publicIP"
            class="detail-row"
          >
            <span class="label">公网IP:</span>
            <span class="value">{{ instance.publicIP }}</span>
          </div>
          <div
            v-if="instance.ipv6Address"
            class="detail-row"
          >
            <span class="label">IPv6:</span>
            <span class="value">{{ instance.ipv6Address }}</span>
          </div>
          <div
            v-if="instance.cpu"
            class="detail-row"
          >
            <span class="label">CPU:</span>
            <span class="value">{{ instance.cpu }}</span>
          </div>
          <div
            v-if="instance.memory"
            class="detail-row"
          >
            <span class="label">内存:</span>
            <span class="value">{{ formatMemorySize(instance.memory) }}</span>
          </div>
          <div
            v-if="instance.created"
            class="detail-row"
          >
            <span class="label">创建时间:</span>
            <span class="value">{{ formatDate(instance.created) }}</span>
          </div>
        </div>
        
        <div class="instance-actions">
          <el-button 
            v-if="instance.status === 'stopped'" 
            size="small" 
            type="success" 
            :loading="actionLoading[instance.id]"
            @click="startInstance(instance)"
          >
            <el-icon><VideoPlay /></el-icon>
            启动
          </el-button>
          <el-button 
            v-if="instance.status === 'running'" 
            size="small" 
            type="warning" 
            :loading="actionLoading[instance.id]"
            @click="stopInstance(instance)"
          >
            <el-icon><VideoPause /></el-icon>
            停止
          </el-button>
          <el-button 
            size="small" 
            type="danger" 
            :loading="actionLoading[instance.id]"
            @click="deleteInstance(instance)"
          >
            <el-icon><Delete /></el-icon>
            删除
          </el-button>
        </div>
      </div>
    </div>

    <!-- 空状态 -->
    <div
      v-if="!loading && instances.length === 0"
      class="empty-state"
    >
      <el-empty description="暂无实例">
        <el-button
          type="primary"
          @click="showCreateDialog"
        >
          创建第一个实例
        </el-button>
      </el-empty>
    </div>

    <!-- 创建实例对话框 -->
    <el-dialog
      v-model="createDialogVisible"
      title="创建实例"
      width="600px"
    >
      <el-form
        ref="createFormRef"
        :model="createForm"
        :rules="createRules"
        label-width="100px"
      >
        <!-- 禁用自定义名称输入，由后端自动生成 -->
        <el-alert
          title="安全提示"
          type="warning"
          :closable="false"
          show-icon
          style="margin-bottom: 20px"
        >
          实例名称将由系统自动生成（Provider名称+随机字符），无法自定义。建议使用用户安全创建页面。
        </el-alert>
        
        <el-form-item
          v-if="supportedInstanceTypes.length > 1"
          label="实例类型"
          prop="instance_type"
        >
          <el-radio-group v-model="createForm.instance_type">
            <el-radio 
              v-for="type in supportedInstanceTypes" 
              :key="type" 
              :label="type"
            >
              {{ type === 'container' ? '容器' : '虚拟机' }}
            </el-radio>
          </el-radio-group>
          <div style="margin-top: 5px; color: #909399; font-size: 12px;">
            {{ createForm.instance_type === 'container' ? '轻量级容器，启动快，资源占用少' : '完整虚拟机，隔离性更好，支持不同操作系统' }}
          </div>
        </el-form-item>
        
        <el-form-item
          label="镜像"
          prop="image"
        >
          <el-select
            v-model="createForm.image"
            placeholder="请选择镜像"
            style="width: 100%"
            filterable
          >
            <el-option 
              v-for="image in availableImages" 
              :key="image.id" 
              :label="`${image.name}:${image.tag}`" 
              :value="`${image.name}:${image.tag}`" 
            />
          </el-select>
          <div style="margin-top: 5px;">
            <el-button
              size="small"
              @click="loadImages"
            >
              刷新镜像列表
            </el-button>
          </div>
        </el-form-item>
        
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="CPU">
              <el-input
                v-model="createForm.cpu"
                placeholder="如: 1 或 0.5"
              />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="内存">
              <el-input
                v-model="createForm.memory"
                placeholder="如: 512MB 或 1GB"
              />
            </el-form-item>
          </el-col>
        </el-row>
        
        <el-form-item
          v-if="selectedProvider === 'proxmox'"
          label="磁盘"
        >
          <el-input
            v-model="createForm.disk"
            placeholder="如: 10GB"
          />
        </el-form-item>
        
        <el-form-item
          v-if="selectedProvider === 'docker'"
          label="端口映射"
        >
          <el-input
            v-model="portsInput"
            placeholder="如: 8080:80,3306:3306"
            @blur="parsePorts"
          />
          <div style="margin-top: 5px; color: #909399; font-size: 12px;">
            格式: 主机端口:容器端口，多个端口用逗号分隔
          </div>
        </el-form-item>
        
        <el-form-item
          v-if="selectedProvider === 'docker'"
          label="环境变量"
        >
          <el-input 
            v-model="envInput" 
            type="textarea" 
            :rows="3" 
            placeholder="如: KEY1=value1&#10;KEY2=value2"
            @blur="parseEnv"
          />
        </el-form-item>
      </el-form>
      
      <template #footer>
        <el-button @click="createDialogVisible = false">
          取消
        </el-button>
        <el-button
          type="primary"
          :loading="creating"
          @click="createInstance"
        >
          创建
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Refresh, VideoPlay, VideoPause, Delete } from '@element-plus/icons-vue'
import { 
  getInstancesApi, 
  createInstanceApi, 
  startInstanceApi, 
  stopInstanceApi, 
  deleteInstanceApi,
  getImagesApi,
  getProviderCapabilitiesApi
} from '@/api/providers'
import { getAvailableSystemImages } from '@/api/public'
import { formatMemorySize, formatDiskSize } from '@/utils/unit-formatter'

const router = useRouter()

// 响应式数据
const loading = ref(false)
const creating = ref(false)
const instances = ref([])
const availableImages = ref([])
const selectedProvider = ref('docker')
const createDialogVisible = ref(false)
const actionLoading = ref({})
const createFormRef = ref()
const portsInput = ref('')
const envInput = ref('')
const supportedInstanceTypes = ref(['container'])

// 创建表单（移除名称字段，由后端自动生成）
const createForm = reactive({
  image: '',
  cpu: '',
  memory: '',
  disk: '',
  network: '',
  ports: [],
  env: {},
  metadata: {},
  instance_type: 'container' // 默认为容器
})

// 表单验证规则（移除名称验证）
const createRules = {
  image: [{ required: true, message: '请选择镜像', trigger: 'change' }]
}

// 获取状态颜色
const getStatusColor = (status) => {
  const colors = {
    running: 'success',
    stopped: 'info',
    paused: 'warning',
    error: 'danger'
  }
  return colors[status] || 'info'
}

// 获取状态文本
const getStatusText = (status) => {
  const texts = {
    running: '运行中',
    stopped: '已停止',
    paused: '已暂停',
    error: '错误'
  }
  return texts[status] || status
}

// 格式化日期
const formatDate = (dateString) => {
  if (!dateString) return ''
  const date = new Date(dateString)
  return date.toLocaleString('zh-CN')
}

// 加载实例列表
const loadInstances = async () => {
  if (!selectedProvider.value) return
  
  loading.value = true
  try {
    const response = await getInstancesApi(selectedProvider.value)
    instances.value = response.instances || []
  } catch (error) {
    ElMessage.error('加载实例列表失败: ' + error.message)
  } finally {
    loading.value = false
  }
}

// 加载镜像列表
const loadImages = async () => {
  if (!selectedProvider.value) return
  
  try {
    const params = {
      providerType: selectedProvider.value,
      instanceType: createForm.instance_type || 'container'
    }
    const response = await getAvailableSystemImages(params)
    // 将系统镜像转换为前端期望的格式
    availableImages.value = (response.data || []).map(image => ({
      id: image.id,
      name: image.name,
      tag: image.architecture || 'latest',
      imageUrl: image.url
    }))
  } catch (error) {
    ElMessage.error('加载镜像列表失败: ' + error.message)
  }
}

// 跳转到用户安全申请页面
const goToUserApply = () => {
  router.push('/user/apply')
}

// 显示创建对话框
const showCreateDialog = () => {
  createDialogVisible.value = true
  loadImages()
  loadProviderCapabilities()
}

// 加载Provider能力
const loadProviderCapabilities = async () => {
  if (!selectedProvider.value) return
  
  try {
    const response = await getProviderCapabilitiesApi(selectedProvider.value)
    supportedInstanceTypes.value = response.data.supported_instance_types || ['container']
    
    // 如果当前选择的类型不被支持，重置为第一个支持的类型
    if (!supportedInstanceTypes.value.includes(createForm.instance_type)) {
      createForm.instance_type = supportedInstanceTypes.value[0]
    }
  } catch (error) {
    console.error('加载Provider能力失败:', error)
    supportedInstanceTypes.value = ['container']
  }
}

// 解析端口映射
const parsePorts = () => {
  if (!portsInput.value) {
    createForm.ports = []
    return
  }
  
  createForm.ports = portsInput.value.split(',').map(port => port.trim()).filter(port => port)
}

// 解析环境变量
const parseEnv = () => {
  createForm.env = {}
  if (!envInput.value) return
  
  const lines = envInput.value.split('\n')
  lines.forEach(line => {
    const [key, ...valueParts] = line.split('=')
    if (key && valueParts.length > 0) {
      createForm.env[key.trim()] = valueParts.join('=').trim()
    }
  })
}

// 创建实例
const createInstance = async () => {
  if (!createFormRef.value) return
  
  try {
    await createFormRef.value.validate()
    creating.value = true
    
    await createInstanceApi(selectedProvider.value, createForm)
    
    ElMessage.success('实例创建成功！')
    createDialogVisible.value = false
    resetCreateForm()
    loadInstances()
    
  } catch (error) {
    ElMessage.error('创建实例失败: ' + error.message)
  } finally {
    creating.value = false
  }
}

// 启动实例
const startInstance = async (instance) => {
  actionLoading.value[instance.id] = true
  try {
    await startInstanceApi(selectedProvider.value, instance.id)
    ElMessage.success('实例启动成功！')
    loadInstances()
  } catch (error) {
    ElMessage.error('启动实例失败: ' + error.message)
  } finally {
    actionLoading.value[instance.id] = false
  }
}

// 停止实例
const stopInstance = async (instance) => {
  actionLoading.value[instance.id] = true
  try {
    await stopInstanceApi(selectedProvider.value, instance.id)
    ElMessage.success('实例停止成功！')
    loadInstances()
  } catch (error) {
    ElMessage.error('停止实例失败: ' + error.message)
  } finally {
    actionLoading.value[instance.id] = false
  }
}

// 删除实例
const deleteInstance = async (instance) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除实例 "${instance.name}" 吗？此操作不可恢复。`,
      '确认删除',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
      }
    )
    
    actionLoading.value[instance.id] = true
    await deleteInstanceApi(selectedProvider.value, instance.id)
    ElMessage.success('实例删除成功！')
    loadInstances()
    
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除实例失败: ' + error.message)
    }
  } finally {
    actionLoading.value[instance.id] = false
  }
}

// 重置创建表单（移除名称字段）
const resetCreateForm = () => {
  if (createFormRef.value) {
    createFormRef.value.resetFields()
  }
  Object.assign(createForm, {
    image: '',
    cpu: '',
    memory: '',
    disk: '',
    network: '',
    ports: [],
    env: {},
    metadata: {},
    instance_type: supportedInstanceTypes.value[0] || 'container'
  })
  portsInput.value = ''
  envInput.value = ''
}

// 页面加载时初始化
onMounted(() => {
  const urlParams = new URLSearchParams(window.location.search)
  const provider = urlParams.get('provider')
  if (provider) {
    selectedProvider.value = provider
  }
  
  loadInstances()
  loadProviderCapabilities()
})
</script>

<style scoped>
.instance-management {
  padding: 20px;
  max-width: 1400px;
  margin: 0 auto;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 30px;
  padding: 20px;
  background: white;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
}

.page-header h1 {
  color: #2c3e50;
  margin: 0;
}

.header-actions {
  display: flex;
  gap: 15px;
  align-items: center;
}

.instance-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(400px, 1fr));
  gap: 20px;
}

.instance-card {
  background: white;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
  transition: all 0.3s ease;
}

.instance-card:hover {
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
  transform: translateY(-2px);
}

.instance-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 15px;
}

.instance-info h3 {
  margin: 0 0 5px 0;
  color: #2c3e50;
}

.instance-details {
  margin-bottom: 20px;
}

.detail-row {
  display: flex;
  justify-content: space-between;
  margin-bottom: 8px;
  padding: 5px 0;
  border-bottom: 1px solid #f0f0f0;
}

.detail-row:last-child {
  border-bottom: none;
}

.label {
  font-weight: 500;
  color: #606266;
  min-width: 80px;
}

.value {
  color: #303133;
  word-break: break-all;
}

.instance-actions {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}

.instance-actions .el-button {
  flex: 1;
  min-width: 80px;
}

.empty-state {
  text-align: center;
  padding: 60px 20px;
  background: white;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
}

@media (max-width: 768px) {
  .page-header {
    flex-direction: column;
    gap: 15px;
  }
  
  .header-actions {
    width: 100%;
    justify-content: center;
  }
  
  .instance-grid {
    grid-template-columns: 1fr;
  }
  
  .instance-actions {
    flex-direction: column;
  }
  
  .instance-actions .el-button {
    width: 100%;
  }
}
</style>