<template>
  <div class="forgot-password-container">
    <div class="forgot-password-form">
      <div v-if="!emailSent">
        <h2>找回密码</h2>
        <p>请输入您的注册邮箱，我们将发送密码重置链接</p>

        <el-form
          ref="forgotFormRef"
          :model="forgotForm"
          :rules="forgotRules"
          label-width="0"
          size="large"
        >
          <el-form-item prop="email">
            <el-input
              v-model="forgotForm.email"
              placeholder="请输入邮箱"
              prefix-icon="Message"
            />
          </el-form-item>

          <el-form-item prop="captcha">
            <div class="captcha-container">
              <el-input
                v-model="forgotForm.captcha"
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

          <el-form-item>
            <el-button
              type="primary"
              :loading="loading"
              style="width: 100%;"
              @click="handleForgotPassword"
            >
              发送重置链接
            </el-button>
          </el-form-item>

          <div class="form-footer">
            <router-link to="/login">
              返回登录
            </router-link>
          </div>
        </el-form>
      </div>

      <div
        v-else
        class="success-message"
      >
        <el-result
          icon="success"
          title="邮件已发送"
          sub-title="请检查您的邮箱，点击邮件中的链接重置密码"
        >
          <template #extra>
            <el-button
              type="primary"
              @click="goToLogin"
            >
              返回登录
            </el-button>
          </template>
        </el-result>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { forgotPassword } from '@/api/auth'
import { getCaptcha } from '@/api/auth'

const router = useRouter()
const forgotFormRef = ref()
const loading = ref(false)
const emailSent = ref(false)
const captchaImage = ref('')
const captchaId = ref('')

const forgotForm = reactive({
  email: '',
  captcha: ''
})

const forgotRules = reactive({
  email: [
    { required: true, message: '请输入邮箱', trigger: 'blur' },
    { type: 'email', message: '请输入正确的邮箱格式', trigger: 'blur' }
  ],
  captcha: [
    { required: true, message: '请输入验证码', trigger: 'blur' }
  ]
})

const handleForgotPassword = async () => {
  if (!forgotFormRef.value) return

  await forgotFormRef.value.validate(async (valid) => {
    if (!valid) return

    loading.value = true
    try {
      const response = await forgotPassword({
        email: forgotForm.email,
        captcha: forgotForm.captcha,
        captchaId: captchaId.value
      })

      if (response.code === 0 || response.code === 200) {
        emailSent.value = true
      }
    } catch (error) {
      console.error('发送重置邮件失败', error)
      ElMessage.error('发送重置邮件失败，请稍后重试')
      refreshCaptcha()
    } finally {
      loading.value = false
    }
  })
}

const refreshCaptcha = async () => {
  try {
    const response = await getCaptcha()
    if (response.code === 0 || response.code === 200) {
      captchaImage.value = response.data.imageData
      captchaId.value = response.data.captchaId
      forgotForm.captcha = ''
    }
  } catch (error) {
    console.error('获取验证码失败', error)
  }
}

const goToLogin = () => {
  router.push('/login')
}

onMounted(() => {
  refreshCaptcha()
})
</script>

<style scoped>
.forgot-password-container {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  background-color: #f5f7fa;
}

.forgot-password-form {
  width: 400px;
  padding: 40px;
  background-color: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1);
}

.forgot-password-form h2 {
  font-size: 24px;
  color: #303133;
  margin-bottom: 10px;
  text-align: center;
}

.forgot-password-form p {
  font-size: 14px;
  color: #909399;
  margin-bottom: 30px;
  text-align: center;
}

.form-footer {
  text-align: center;
  margin-top: 20px;
}

.form-footer a {
  color: #409eff;
  text-decoration: none;
}

.success-message {
  text-align: center;
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

@media (max-width: 768px) {
  .forgot-password-form {
    width: 90%;
    padding: 20px;
  }
}
</style>