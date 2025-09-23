<template>
  <div class="invite-codes-container">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>邀请码管理</span>
          <div>
            <el-button
              type="success"
              @click="showCreateDialog = true"
            >
              创建自定义邀请码
            </el-button>
            <el-button
              type="primary"
              @click="showGenerateDialog = true"
            >
              批量生成邀请码
            </el-button>
          </div>
        </div>
      </template>
      
      <el-table
        v-loading="loading"
        :data="inviteCodes"
        style="width: 100%"
      >
        <el-table-column
          prop="id"
          label="ID"
          width="60"
        />
        <el-table-column
          prop="code"
          label="邀请码"
        />
        <el-table-column
          prop="max_uses"
          label="最大使用次数"
          width="120"
        >
          <template #default="scope">
            {{ scope.row.max_uses === 0 ? '无限制' : scope.row.max_uses }}
          </template>
        </el-table-column>
        <el-table-column
          prop="used_count"
          label="已使用次数"
          width="120"
        />
        <el-table-column
          prop="status"
          label="状态"
          width="100"
        >
          <template #default="scope">
            <el-tag :type="scope.row.status === 1 ? 'success' : 'info'">
              {{ scope.row.status === 1 ? '可用' : '已失效' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column
          prop="expires_at"
          label="过期时间"
          width="160"
        >
          <template #default="scope">
            {{ scope.row.expires_at ? new Date(scope.row.expires_at).toLocaleString() : '永不过期' }}
          </template>
        </el-table-column>
        <el-table-column
          prop="created_at"
          label="创建时间"
          width="160"
        >
          <template #default="scope">
            {{ new Date(scope.row.created_at).toLocaleString() }}
          </template>
        </el-table-column>
        <el-table-column
          label="操作"
          width="120"
        >
          <template #default="scope">
            <el-button
              size="small"
              type="danger"
              @click="deleteCode(scope.row.id)"
            >
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination-wrapper">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </el-card>

    <!-- 创建自定义邀请码对话框 -->
    <el-dialog 
      v-model="showCreateDialog" 
      title="创建自定义邀请码" 
      width="500px"
    >
      <el-form 
        ref="createFormRef" 
        :model="createForm" 
        :rules="createRules" 
        label-width="120px"
      >
        <el-form-item
          label="邀请码"
          prop="code"
        >
          <el-input 
            v-model="createForm.code" 
            placeholder="请输入自定义邀请码"
            maxlength="50"
            show-word-limit
          />
          <div class="form-tip">
            邀请码必须唯一，建议使用字母、数字、下划线或连字符
          </div>
        </el-form-item>
        <el-form-item
          label="最大使用次数"
          prop="maxUses"
        >
          <el-input-number
            v-model="createForm.maxUses"
            :min="0"
          />
          <div class="form-tip">
            设置为0表示无限制使用
          </div>
        </el-form-item>
        <el-form-item
          label="过期时间"
          prop="expiresAt"
        >
          <el-date-picker
            v-model="createForm.expiresAt"
            type="datetime"
            placeholder="选择过期时间"
            format="YYYY-MM-DD HH:mm:ss"
            value-format="YYYY-MM-DD HH:mm:ss"
            style="width: 100%"
          />
          <div class="form-tip">
            不设置表示永不过期
          </div>
        </el-form-item>
        <el-form-item
          label="描述"
          prop="description"
        >
          <el-input 
            v-model="createForm.description" 
            type="textarea" 
            :rows="3"
            placeholder="邀请码用途描述"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="cancelCreate">取消</el-button>
          <el-button
            type="primary"
            :loading="createLoading"
            @click="submitCreate"
          >创建</el-button>
        </span>
      </template>
    </el-dialog>

    <!-- 生成邀请码对话框 -->
    <el-dialog 
      v-model="showGenerateDialog" 
      title="生成邀请码" 
      width="500px"
    >
      <el-form 
        ref="generateFormRef" 
        :model="generateForm" 
        :rules="generateRules" 
        label-width="120px"
      >
        <el-form-item
          label="生成数量"
          prop="count"
        >
          <el-input-number
            v-model="generateForm.count"
            :min="1"
            :max="100"
          />
        </el-form-item>
        <el-form-item
          label="最大使用次数"
          prop="maxUses"
        >
          <el-input-number
            v-model="generateForm.maxUses"
            :min="0"
          />
          <div class="form-tip">
            设置为0表示无限制使用
          </div>
        </el-form-item>
        <el-form-item
          label="过期时间"
          prop="expiresAt"
        >
          <el-date-picker
            v-model="generateForm.expiresAt"
            type="datetime"
            placeholder="选择过期时间"
            format="YYYY-MM-DD HH:mm:ss"
            value-format="YYYY-MM-DD HH:mm:ss"
            style="width: 100%"
          />
          <div class="form-tip">
            不设置表示永不过期
          </div>
        </el-form-item>
        <el-form-item
          label="描述"
          prop="description"
        >
          <el-input 
            v-model="generateForm.description" 
            type="textarea" 
            :rows="3"
            placeholder="邀请码用途描述"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="cancelGenerate">取消</el-button>
          <el-button
            type="primary"
            :loading="generateLoading"
            @click="submitGenerate"
          >生成</el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getInviteCodes, createInviteCode, generateInviteCodes, deleteInviteCode } from '@/api/admin'

const inviteCodes = ref([])
const loading = ref(false)
const showCreateDialog = ref(false)
const showGenerateDialog = ref(false)
const createLoading = ref(false)
const generateLoading = ref(false)
const createFormRef = ref()
const generateFormRef = ref()

// 分页
const currentPage = ref(1)
const pageSize = ref(10)
const total = ref(0)

// 创建自定义邀请码表单
const createForm = reactive({
  code: '',
  maxUses: 1,
  expiresAt: '',
  description: ''
})

// 创建表单验证规则
const createRules = {
  code: [
    { required: true, message: '请输入邀请码', trigger: 'blur' },
    { min: 3, max: 50, message: '邀请码长度为3-50个字符', trigger: 'blur' },
    { pattern: /^[a-zA-Z0-9_-]+$/, message: '邀请码只能包含字母、数字、下划线或连字符', trigger: 'blur' }
  ],
  maxUses: [
    { required: true, message: '请输入最大使用次数', trigger: 'blur' }
  ]
}

// 生成邀请码表单
const generateForm = reactive({
  count: 1,
  maxUses: 1,
  expiresAt: '',
  description: ''
})

// 表单验证规则
const generateRules = {
  count: [
    { required: true, message: '请输入生成数量', trigger: 'blur' }
  ],
  maxUses: [
    { required: true, message: '请输入最大使用次数', trigger: 'blur' }
  ]
}

const loadInviteCodes = async () => {
  loading.value = true
  try {
    const response = await getInviteCodes({
      page: currentPage.value,
      pageSize: pageSize.value
    })
    inviteCodes.value = response.data.list || []
    total.value = response.data.total || 0
  } catch (error) {
    ElMessage.error('加载邀请码列表失败')
  } finally {
    loading.value = false
  }
}

const cancelCreate = () => {
  showCreateDialog.value = false
  createFormRef.value?.resetFields()
  Object.assign(createForm, {
    code: '',
    maxUses: 1,
    expiresAt: '',
    description: ''
  })
}

const submitCreate = async () => {
  try {
    await createFormRef.value.validate()
    createLoading.value = true

    const data = {
      code: createForm.code,
      count: 1, // 自定义邀请码一次只创建一个
      maxUses: createForm.maxUses,
      expiresAt: createForm.expiresAt || '',
      remark: createForm.description
    }

    await createInviteCode(data)
    ElMessage.success('自定义邀请码创建成功')
    cancelCreate()
    await loadInviteCodes()
  } catch (error) {
    if (error.response?.data?.msg) {
      ElMessage.error(error.response.data.msg)
    } else {
      ElMessage.error('邀请码创建失败')
    }
  } finally {
    createLoading.value = false
  }
}

const cancelGenerate = () => {
  showGenerateDialog.value = false
  generateFormRef.value?.resetFields()
  Object.assign(generateForm, {
    count: 1,
    maxUses: 1,
    expiresAt: '',
    description: ''
  })
}

const submitGenerate = async () => {
  try {
    await generateFormRef.value.validate()
    generateLoading.value = true

    const data = {
      count: generateForm.count,
      maxUses: generateForm.maxUses,
      expiresAt: generateForm.expiresAt || '',
      remark: generateForm.description
    }

    await generateInviteCodes(data)
    ElMessage.success('邀请码生成成功')
    cancelGenerate()
    await loadInviteCodes()
  } catch (error) {
    ElMessage.error('邀请码生成失败')
  } finally {
    generateLoading.value = false
  }
}

const deleteCode = async (id) => {
  try {
    await ElMessageBox.confirm(
      '确定删除该邀请码吗？此操作不可恢复。',
      '删除邀请码',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
      }
    )
    
    await deleteInviteCode(id)
    ElMessage.success('删除成功')
    await loadInviteCodes()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

const handleSizeChange = (newSize) => {
  pageSize.value = newSize
  currentPage.value = 1
  loadInviteCodes()
}

const handleCurrentChange = (newPage) => {
  currentPage.value = newPage
  loadInviteCodes()
}

onMounted(() => {
  loadInviteCodes()
})
</script>

<style scoped>
.invite-codes-container {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.pagination-wrapper {
  margin-top: 20px;
  display: flex;
  justify-content: center;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

.form-tip {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}
</style>