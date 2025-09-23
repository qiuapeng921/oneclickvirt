import request from '@/utils/request'
import { createLongTimeoutRequest } from '@/utils/longTimeoutRequest'

// 创建实例操作专用请求实例（120秒超时）
const instanceOperationRequest = createLongTimeoutRequest(120000, {
  requestPrefix: 'provider_instance'
})

export function connectNodeApi(data) {
  return request({
    url: '/v1/providers/connect',
    method: 'post',
    data
  })
}

export function getSupportedProviders() {
  return request({
    url: '/v1/providers',
    method: 'get'
  })
}

export function getProviderStatusApi(type) {
  return request({
    url: `/v1/providers/${type}/status`,
    method: 'get'
  })
}

export function getProviderCapabilities(type) {
  return request({
    url: `/v1/providers/${type}/capabilities`,
    method: 'get'
  })
}

export const getInstancesApi = (type) => {
  return request({
    url: `/v1/providers/${type}/instances`,
    method: 'get'
  })
}

export const createInstanceApi = (type, data) => {
  return instanceOperationRequest({
    url: `/v1/providers/${type}/instances`,
    method: 'post',
    data
  })
}

export const getInstanceApi = (type, id) => {
  return request({
    url: `/v1/providers/${type}/instances/${id}`,
    method: 'get'
  })
}

export const startInstanceApi = (type, id) => {
  return request({
    url: `/v1/providers/${type}/instances/${id}/start`,
    method: 'post'
  })
}

export const stopInstanceApi = (type, id) => {
  return request({
    url: `/v1/providers/${type}/instances/${id}/stop`,
    method: 'post'
  })
}

export const deleteInstanceApi = (type, id) => {
  return instanceOperationRequest({
    url: `/v1/providers/${type}/instances/${id}`,
    method: 'delete'
  })
}

export const getImagesApi = (type) => {
  return request({
    url: `/v1/providers/${type}/images`,
    method: 'get'
  })
}

export const pullImageApi = (type, data) => {
  return request({
    url: `/v1/providers/${type}/images/pull`,
    method: 'post',
    data,
    timeout: 100000
  })
}

export const deleteImageApi = (type, id) => {
  return request({
    url: `/v1/providers/${type}/images/${id}`,
    method: 'delete'
  })
}