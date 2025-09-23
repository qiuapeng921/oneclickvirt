import { defineStore } from 'pinia'

export const useRouterStore = defineStore('router', {
  state: () => ({
    menu: [
      {
        path: '/dashboard',
        name: '仪表盘',
        icon: 'dashboard'
      }
    ]
  }),

  getters: {
    menuTree: (state) => state.menu
  },

  actions: {
    setMenu(menu) {
      this.menu = menu
    }
  }
})