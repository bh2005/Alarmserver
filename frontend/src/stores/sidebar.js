import { ref } from 'vue'
import { defineStore } from 'pinia'

export const useSidebarStore = defineStore('sidebar', () => {
  const collapsed = ref(localStorage.getItem('alarmcore_sidebar') === 'collapsed')

  function toggle() {
    collapsed.value = !collapsed.value
    localStorage.setItem('alarmcore_sidebar', collapsed.value ? 'collapsed' : 'expanded')
  }

  return { collapsed, toggle }
})
