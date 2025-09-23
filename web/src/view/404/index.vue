<template>
  <div class="not-found-container">
    <div class="not-found-content">
      <div class="error-code">
        404
      </div>
      <h1>页面不存在</h1>
      <p>抱歉，您访问的页面不存在或已被删除。</p>
      <div class="actions">
        <el-button
          type="primary"
          @click="goHome"
        >
          返回首页
        </el-button>
        <el-button @click="goBack">
          返回上页
        </el-button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { useRouter } from 'vue-router'

const router = useRouter()

const goHome = () => {
  router.push('/')
}

const goBack = () => {
  router.go(-1)
}
</script>

<style scoped>
.not-found-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.not-found-content {
  text-align: center;
  color: white;
}

.error-code {
  font-size: 120px;
  font-weight: bold;
  margin-bottom: 20px;
  text-shadow: 2px 2px 4px rgba(0, 0, 0, 0.3);
}

h1 {
  font-size: 36px;
  margin: 0 0 20px 0;
  font-weight: 600;
}

p {
  font-size: 18px;
  margin: 0 0 40px 0;
  opacity: 0.9;
}

.actions {
  display: flex;
  gap: 20px;
  justify-content: center;
}
</style>