<template>
  <router-view v-if="route.meta.public" />

  <div v-else class="flex h-screen overflow-hidden bg-slate-50 dark:bg-slate-900">
    <AppSidebar @open-help="helpOpen = true" />

    <main class="flex-1 overflow-y-auto px-6 py-5">
      <router-view />
    </main>

    <HelpPanel :open="helpOpen" @close="helpOpen = false" />
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import AppSidebar from './components/AppSidebar.vue'
import HelpPanel from './components/HelpPanel.vue'
import { useAuthStore } from './stores/auth.js'
import { useThemeStore } from './stores/theme.js'

const route = useRoute()
const auth = useAuthStore()
useThemeStore()

const helpOpen = ref(false)

function onKeydown(e) {
  if (e.key === 'F1') { e.preventDefault(); helpOpen.value = !helpOpen.value }
}

onMounted(async () => {
  if (auth.isLoggedIn) {
    await auth.fetchMe().catch((e) => {
      if (e.response?.status === 401) auth.logout()
    })
  }
  window.addEventListener('keydown', onKeydown)
})
onUnmounted(() => window.removeEventListener('keydown', onKeydown))
</script>
