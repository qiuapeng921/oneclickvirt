<template>
  <div class="register-container">
    <!-- 注册被禁用的提示 -->
    <div
      v-if="!registrationEnabled"
      class="registration-disabled"
    >
      <el-card>
        <div class="disabled-content">
          <el-icon
            size="60"
            color="#e6a23c"
          >
            <Warning />
          </el-icon>
          <h2>注册功能暂时关闭</h2>
          <p>系统管理员已暂时关闭用户注册功能，请稍后再试。</p>
          <el-button
            type="primary"
            @click="router.push('/login')"
          >
            返回登录
          </el-button>
        </div>
      </el-card>
    </div>

    <!-- 正常注册表单 -->
    <div
      v-else
      class="register-form"
    >
      <div class="register-header">
        <h2>用户注册</h2>
        <p>创建您的账号</p>
      </div>

      <el-form 
        ref="registerFormRef"
        :model="registerForm"
        :rules="registerRules"
        label-width="80px"
        size="large"
      >
        <el-form-item
          label="用户名"
          prop="username"
        >
          <el-input 
            v-model="registerForm.username"
            placeholder="请输入用户名"
          />
        </el-form-item>

        <el-form-item
          label="密码"
          prop="password"
        >
          <el-input 
            v-model="registerForm.password"
            type="password"
            placeholder="请输入密码"
            show-password
          />
          <div class="password-hint">
            <el-text
              size="small"
              type="info"
            >
              密码需要至少8位，包含大写字母、小写字母、数字和特殊字符
            </el-text>
          </div>
        </el-form-item>

        <el-form-item
          label="确认密码"
          prop="confirmPassword"
        >
          <el-input 
            v-model="registerForm.confirmPassword"
            type="password"
            placeholder="请再次输入密码"
            show-password
          />
        </el-form-item>

        <el-form-item
          label="验证码"
          prop="captcha"
        >
          <div class="captcha-container">
            <el-input 
              v-model="registerForm.captcha"
              placeholder="请输入验证码"
              style="width: 60%"
            />
            <div
              class="captcha-image"
              @click="refreshCaptcha"
            >
              <img
                v-if="captchaImage"
                :src="captchaImage"
                alt="验证码"
              >
              <div
                v-else
                class="captcha-loading"
              >
                加载中...
              </div>
            </div>
          </div>
        </el-form-item>

        <el-form-item
          v-if="showInviteCode"
          label="邀请码"
          prop="inviteCode"
        >
          <el-input 
            v-model="registerForm.inviteCode"
            :placeholder="inviteCodeRequired ? '请输入邀请码（必填）' : '请输入邀请码（可选）'"
          />
        </el-form-item>

        <el-form-item>
          <el-button 
            type="primary" 
            :loading="loading" 
            style="width: 100%;"
            @click="handleRegister"
          >
            注册
          </el-button>
        </el-form-item>

        <div class="form-footer">
          <p>
            已有账户？<router-link to="/login">
              立即登录
            </router-link>
          </p>
        </div>
      </el-form>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { getCaptcha, register } from '@/api/auth'
import { getRegisterConfig } from '@/api/config'
import { useErrorHandler } from '@/composables/useErrorHandler'
import { Warning } from '@element-plus/icons-vue'

const router = useRouter()
const { executeAsync, handleSubmit } = useErrorHandler()
const registerFormRef = ref()
const loading = ref(false)
const showInviteCode = ref(false)
const inviteCodeRequired = ref(false)
const captchaImage = ref('')
const registrationEnabled = ref(true)

const registerForm = reactive({
  username: '',
  password: '',
  confirmPassword: '',
  captcha: '',
  captchaId: '',
  inviteCode: '',
  registerType: 'normal'
})

const validateConfirmPassword = (rule, value, callback) => {
  if (value !== registerForm.password) {
    callback(new Error('两次输入的密码不一致'))
  } else {
    callback()
  }
}

const validateInviteCode = (rule, value, callback) => {
  if (inviteCodeRequired.value && (!value || value.trim() === '')) {
    callback(new Error('请输入邀请码'))
  } else {
    callback()
  }
}

const registerRules = reactive({
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 3, max: 20, message: '用户名长度在 3 到 20 个字符', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 8, message: '密码长度至少为 8 个字符', trigger: 'blur' }
  ],
  confirmPassword: [
    { required: true, message: '请再次输入密码', trigger: 'blur' },
    { validator: validateConfirmPassword, trigger: 'blur' }
  ],
  captcha: [
    { required: true, message: '请输入验证码', trigger: 'blur' }
  ],
  inviteCode: [
    { validator: validateInviteCode, trigger: 'blur' }
  ]
})

const refreshCaptcha = async () => {
  await executeAsync(async () => {
    const response = await getCaptcha()
    captchaImage.value = response.data.imageData
    registerForm.captchaId = response.data.captchaId
    registerForm.captcha = ''
  }, {
    showError: false, // 静默处理验证码错误
    showLoading: false
  })
}

const handleRegister = async () => {
  if (!registerFormRef.value) return

  await registerFormRef.value.validate(async (valid) => {
    if (!valid) return

    loading.value = true
    try {
      const result = await handleSubmit(async () => {
        return await register({
          username: registerForm.username,
          password: registerForm.password,
          captcha: registerForm.captcha,
          captchaId: registerForm.captchaId,
          inviteCode: showInviteCode.value ? registerForm.inviteCode : undefined,
          registerType: registerForm.registerType
        })
      }, {
        successMessage: '注册成功，正在跳转到仪表盘...',
        showLoading: false // 使用组件自己的loading
      })

      if (result.success && result.data) {
        // 注册成功，直接设置用户登录状态
        const responseData = result.data.data // 修复：正确获取嵌套的data数据
        
        // 导入用户store
        const { useUserStore } = await import('@/pinia/modules/user')
        const userStore = useUserStore()
        
        // 设置用户登录状态
        userStore.setToken(responseData.token)
        userStore.setUser(responseData.user)
        
        // 跳转到用户仪表盘
        router.push('/user/dashboard')
      } else {
        refreshCaptcha() // 注册失败刷新验证码
      }
    } finally {
      loading.value = false
    }
  })
}

const checkRegistrationEnabled = async () => {
  const result = await executeAsync(async () => {
    const response = await getRegisterConfig()
    const config = response.data
    
    // 新逻辑：如果启用公开注册，或者启用邀请码系统但不强制要求邀请码
    const enablePublicRegistration = config.auth?.enablePublicRegistration ?? false
    const inviteCodeEnabled = config.inviteCode?.enabled ?? false
    
    // 如果启用公开注册，或者启用了邀请码系统，则允许注册
    const canRegister = enablePublicRegistration || inviteCodeEnabled
    
    // 显示邀请码输入框的条件：启用了邀请码系统
    showInviteCode.value = inviteCodeEnabled
    
    // 邀请码必填的条件：启用邀请码系统且未启用公开注册
    inviteCodeRequired.value = inviteCodeEnabled && !enablePublicRegistration
    
    return canRegister
  }, {
    showError: false, // 不显示错误消息
    showLoading: false
  })
  
  // 如果成功获取配置，使用返回的值；否则默认允许注册
  registrationEnabled.value = result.success ? result.data : true
}

onMounted(async () => {
  await checkRegistrationEnabled()
  if (registrationEnabled.value) {
    refreshCaptcha()
  }
})
</script>

<style scoped>
.register-container {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  background-color: #f5f7fa;
}

.register-form {
  width: 500px;
  padding: 40px;
  background-color: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1);
}

.registration-disabled {
  width: 500px;
}

.disabled-content {
  text-align: center;
  padding: 40px;
}

.disabled-content h2 {
  color: #e6a23c;
  margin: 20px 0;
  font-size: 24px;
}

.disabled-content p {
  color: #666;
  margin-bottom: 30px;
  font-size: 16px;
  line-height: 1.5;
}

.register-header {
  text-align: center;
  margin-bottom: 30px;
}

.register-header h2 {
  font-size: 24px;
  color: #303133;
  margin-bottom: 10px;
}

.register-header p {
  font-size: 14px;
  color: #909399;
}

.form-footer {
  text-align: center;
  margin-top: 20px;
}

.form-footer a {
  color: #409eff;
  text-decoration: none;
}

.captcha-container {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.captcha-image {
  width: 38%;
  height: 40px;
  border: 1px solid #dcdfe6;
  border-radius: 4px;
  overflow: hidden;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
}

.captcha-image img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.captcha-loading {
  font-size: 12px;
  color: #909399;
}

.password-hint {
  margin-top: 5px;
  font-size: 12px;
  line-height: 1.4;
}

@media (max-width: 768px) {
  .register-form {
    width: 90%;
    padding: 20px;
  }
}
</style>