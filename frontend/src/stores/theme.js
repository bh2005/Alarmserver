import { ref } from 'vue'
import { defineStore } from 'pinia'

export const useThemeStore = defineStore('theme', () => {
  const isDark = ref(localStorage.getItem('alarmcore_theme') === 'dark')

  function apply() {
    document.documentElement.classList.toggle('dark', isDark.value)
  }

  function toggle() {
    isDark.value = !isDark.value
    localStorage.setItem('alarmcore_theme', isDark.value ? 'dark' : 'light')
    apply()
  }

  apply()

  return { isDark, toggle }
})
