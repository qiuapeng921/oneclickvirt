import { createApp } from 'vue'
import { errorHandler } from '@/utils/errorHandler'
import { useErrorHandler } from '@/composables/useErrorHandler'

/**
 * 全局错误处理配置插件
 * 为整个应用提供统一的错误处理能力
 */
export const errorHandlerPlugin = {
  install(app) {
    // 全局属性注入
    app.config.globalProperties.$errorHandler = errorHandler
    app.config.globalProperties.$useErrorHandler = useErrorHandler

    // 全局提供
    app.provide('errorHandler', errorHandler)
    app.provide('useErrorHandler', useErrorHandler)

    // 全局未捕获错误处理
    app.config.errorHandler = (err, vm, info) => {
      console.error('Vue全局错误:', err, info)
      
      // 使用统一错误处理
      errorHandler.handleApiError(err, {
        showMessage: true,
        customMessage: '系统遇到未知错误，请刷新页面重试'
      })
    }

    // 全局未处理的Promise错误
    window.addEventListener('unhandledrejection', (event) => {
      console.error('未处理的Promise错误:', event.reason)
      
      // 阻止浏览器默认的错误处理
      event.preventDefault()
      
      // 使用统一错误处理
      errorHandler.handleApiError(event.reason, {
        showMessage: true,
        customMessage: '系统遇到网络错误，请检查网络连接'
      })
    })

    // 添加全局混入
    app.mixin({
      methods: {
        // 便捷的错误处理方法
        $handleError(error, options = {}) {
          return errorHandler.handleApiError(error, options)
        },
        
        // 便捷的确认对话框
        async $confirm(message, title = '确认操作', options = {}) {
          return await errorHandler.showConfirmDialog(message, title, options)
        },
        
        // 便捷的消息显示
        $showMessage(message, type = 'info') {
          errorHandler.showErrorMessage(message, type)
        }
      }
    })
  }
}

/**
 * 在main.js中使用示例：
 * 
 * import { createApp } from 'vue'
 * import App from './App.vue'
 * import { errorHandlerPlugin } from '@/config/errorHandler'
 * 
 * const app = createApp(App)
 * app.use(errorHandlerPlugin)
 * app.mount('#app')
 */

export default errorHandlerPlugin
