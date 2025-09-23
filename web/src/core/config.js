export default {
  // API 基础地址
  baseURL: import.meta.env.VITE_API_BASE_URL || 'http://localhost:8888',
  
  // 应用信息
  app: {
    name: 'OneClickVirt',
    version: '1.0.0',
    description: '虚拟化管理平台'
  },
  
  // 分页配置
  pagination: {
    pageSize: 20,
    pageSizes: [10, 20, 50, 100]
  },
  
  // 上传配置
  upload: {
    maxSize: 10 * 1024 * 1024, // 10MB
    acceptTypes: ['.jpg', '.jpeg', '.png', '.gif', '.pdf', '.doc', '.docx']
  }
}