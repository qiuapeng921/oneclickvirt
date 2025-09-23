import request from '@/utils/request'

export function getProviders() {
  return request({
    url: '/v1/virtualization/providers',
    method: 'get'
  })
}

export function addProvider(data) {
  return request({
    url: '/v1/virtualization/providers',
    method: 'post',
    data
  })
}

export function updateProvider(id, data) {
  return request({
    url: `/v1/virtualization/providers/${id}`,
    method: 'put',
    data
  })
}

export function deleteProvider(id) {
  return request({
    url: `/v1/virtualization/providers/${id}`,
    method: 'delete'
  })
}

export function testProvider(id) {
  return request({
    url: `/v1/virtualization/providers/${id}/test`,
    method: 'post'
  })
}

export function getInstances(params) {
  return request({
    url: '/v1/virtualization/instances',
    method: 'get',
    params
  })
}

export function getInstance(id) {
  return request({
    url: `/v1/virtualization/instances/${id}`,
    method: 'get'
  })
}

export function createInstance(data) {
  return request({
    url: '/v1/virtualization/instances',
    method: 'post',
    data
  })
}

export function updateInstance(id, data) {
  return request({
    url: `/v1/virtualization/instances/${id}`,
    method: 'put',
    data
  })
}

export function deleteInstance(id) {
  return request({
    url: `/v1/virtualization/instances/${id}`,
    method: 'delete'
  })
}

export function startInstance(id) {
  return request({
    url: `/v1/virtualization/instances/${id}/start`,
    method: 'post'
  })
}

export function stopInstance(id) {
  return request({
    url: `/v1/virtualization/instances/${id}/stop`,
    method: 'post'
  })
}

export function restartInstance(id) {
  return request({
    url: `/v1/virtualization/instances/${id}/restart`,
    method: 'post'
  })
}

export function getInstanceLogs(id, params) {
  return request({
    url: `/v1/virtualization/instances/${id}/logs`,
    method: 'get',
    params
  })
}

export function getInstanceStats(id) {
  return request({
    url: `/v1/virtualization/instances/${id}/stats`,
    method: 'get'
  })
}

export function getImages(params) {
  return request({
    url: '/v1/virtualization/images',
    method: 'get',
    params
  })
}

export function pullImage(data) {
  return request({
    url: '/v1/virtualization/images/pull',
    method: 'post',
    data
  })
}

export function deleteImage(id) {
  return request({
    url: `/v1/virtualization/images/${id}`,
    method: 'delete'
  })
}

export function getNetworks() {
  return request({
    url: '/v1/virtualization/networks',
    method: 'get'
  })
}

export function createNetwork(data) {
  return request({
    url: '/v1/virtualization/networks',
    method: 'post',
    data
  })
}

export function deleteNetwork(id) {
  return request({
    url: `/v1/virtualization/networks/${id}`,
    method: 'delete'
  })
}

export function getVolumes() {
  return request({
    url: '/v1/virtualization/volumes',
    method: 'get'
  })
}

export function createVolume(data) {
  return request({
    url: '/v1/virtualization/volumes',
    method: 'post',
    data
  })
}

export function deleteVolume(id) {
  return request({
    url: `/v1/virtualization/volumes/${id}`,
    method: 'delete'
  })
}