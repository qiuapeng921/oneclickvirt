import { defineStore } from 'pinia'

export const useDictionaryStore = defineStore('dictionary', {
  state: () => ({
    dictionaries: {}
  }),

  getters: {
    getDictByType: (state) => (type) => {
      return state.dictionaries[type] || []
    }
  },

  actions: {
    setDictionary(type, data) {
      this.dictionaries[type] = data
    },

    addDictItem(type, item) {
      if (!this.dictionaries[type]) {
        this.dictionaries[type] = []
      }
      this.dictionaries[type].push(item)
    }
  }
})