import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { apiClient, setAuthToken } from '../api/index.js'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('alarmcore_token') || '')
  const user = ref(JSON.parse(localStorage.getItem('alarmcore_user') || 'null'))

  if (token.value) setAuthToken(token.value)

  const isLoggedIn = computed(() => !!token.value)
  const isAdmin = computed(() => user.value?.role === 'admin')

  async function login(username, password) {
    const res = await apiClient.post('/auth/login', { username, password })
    token.value = res.data.token
    localStorage.setItem('alarmcore_token', token.value)
    setAuthToken(token.value)
    await fetchMe()
  }

  async function fetchMe() {
    const res = await apiClient.get('/auth/me')
    user.value = res.data
    localStorage.setItem('alarmcore_user', JSON.stringify(user.value))
  }

  function logout() {
    token.value = ''
    user.value = null
    localStorage.removeItem('alarmcore_token')
    localStorage.removeItem('alarmcore_user')
    setAuthToken(null)
  }

  return { token, user, isLoggedIn, isAdmin, login, fetchMe, logout }
})
