// API函数：获取用户活跃的资源预留
// 文件路径: web/src/api/user.js

import request from '@/utils/request'

/**
 * 获取用户的活跃资源预留
 * @returns {Promise} 预留列表
 */
export function getActiveReservations() {
  return request({
    url: '/v1/user/active-reservations',
    method: 'get'
  })
}

/**
 * 获取用户配额信息（包含预留资源计算）
 * @returns {Promise} 配额信息
 */
export function getUserQuotaInfo() {
  return request({
    url: '/v1/user/quota-info',
    method: 'get'
  })
}

/**
 * 创建实例（使用资源预留机制）
 * @param {Object} data 创建实例参数
 * @returns {Promise} 任务信息
 */
export function createInstance(data) {
  return request({
    url: '/v1/user/instances',
    method: 'post',
    data
  })
}

/**
 * 获取可用节点列表（包含预留资源计算）
 * @returns {Promise} 节点列表
 */
export function getAvailableProviders() {
  return request({
    url: '/v1/user/available-providers',
    method: 'get'
  })
}

/**
 * 取消任务（会自动释放预留资源）
 * @param {number} taskId 任务ID
 * @returns {Promise}
 */
export function cancelTask(taskId) {
  return request({
    url: `/v1/user/tasks/${taskId}/cancel`,
    method: 'post'
  })
}
