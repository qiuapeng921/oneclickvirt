<template>
  <div class="app-wrapper">
    <!-- 顶部栏公告 -->
    <TopbarAnnouncement />
    
    <div
      v-if="device === 'mobile' && sidebar.opened"
      class="drawer-bg"
      @click="handleClickOutside"
    />
    <component
      :is="Sidebar"
      :key="userStore.userType"
      class="sidebar-container"
      :class="{ 'is-collapse': isCollapse }"
    />
    <div
      class="main-container"
      :class="{ 'main-container-collapsed': isCollapse }"
    >
      <div
        class="fixed-header"
        :class="{ 'fixed-header-collapsed': isCollapse }"
      >
        <navbar />
      </div>
      <app-main />
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, nextTick, provide } from 'vue'
import { Navbar, Sidebar, AppMain } from './components'
import { useUserStore } from '@/pinia/modules/user'
import TopbarAnnouncement from '@/components/TopbarAnnouncement.vue'

const userStore = useUserStore()
const device = ref('desktop')
const sidebar = ref({
  opened: true
})
const isCollapse = ref(false)

// 提供给子组件的方法
const toggleSidebarCollapse = (collapsed) => {
  isCollapse.value = collapsed
}

// 提供收缩状态给子组件
provide('toggleSidebarCollapse', toggleSidebarCollapse)

const sidebarWidth = computed(() => {
  return isCollapse.value ? 'var(--sidebar-width-collapsed)' : 'var(--sidebar-width)'
})

const handleClickOutside = () => {
  sidebar.value.opened = false
}

onMounted(() => {
  nextTick(() => {
    const sidebarEl = document.querySelector('.sidebar-container')
    if (!sidebarEl || sidebarEl.children.length === 0) {
      userStore.$patch({ userType: userStore.userType })
    }
  })
})
</script>

<style lang="scss" scoped>
.app-wrapper {
  position: relative;
  height: 100%;
  width: 100%;
  background-color: var(--bg-color-primary);

  &.mobile.openSidebar {
    position: fixed;
    top: 0;
  }
}

.drawer-bg {
  background: #000;
  opacity: 0.3;
  width: 100%;
  top: 0;
  height: 100%;
  position: absolute;
  z-index: 999;
}

.fixed-header {
  position: fixed;
  top: 0;
  right: 0;
  z-index: 9;
  width: calc(100% - var(--sidebar-width));
  transition: width 0.28s;
  background-color: var(--bg-color-secondary);
  box-shadow: var(--box-shadow-light);
  border-bottom: 1px solid var(--border-color);
  
  &.fixed-header-collapsed {
    width: calc(100% - var(--sidebar-width-collapsed));
  }
}

:deep(.sidebar-container.is-collapse) ~ .main-container .fixed-header {
  width: calc(100% - var(--sidebar-width-collapsed));
}

:deep(.sidebar-container) {
  transition: width 0.28s;
  width: var(--sidebar-width) !important;
  background-color: var(--bg-color-sidebar);
  height: 100%;
  position: fixed;
  font-size: 0px;
  top: 0;
  bottom: 0;
  left: 0;
  z-index: 1001;
  overflow: hidden;
  box-shadow: 2px 0 6px rgba(0, 0, 0, 0.1);
  display: block !important;
  visibility: visible !important;
  
  &.is-collapse {
    width: var(--sidebar-width-collapsed) !important;
  }
}

.main-container {
  min-height: 100%;
  transition: margin-left 0.28s;
  margin-left: var(--sidebar-width);
  position: relative;
  padding-top: 50px;
  display: flex;
  flex-direction: column;
  
  &.main-container-collapsed {
    margin-left: var(--sidebar-width-collapsed);
  }
}

/* 侧边栏收缩时调整主容器 */
:deep(.sidebar-container.is-collapse) ~ .main-container {
  margin-left: var(--sidebar-width-collapsed);
}
</style>