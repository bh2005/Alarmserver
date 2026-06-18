<template>
  <div>
    <h1 class="text-xl font-bold mb-1">Mein Einsatz</h1>
    <p class="text-sm text-gray-400 dark:text-slate-500 mb-4">Hier kannst du dich in Bereitschaftsgruppen aktiv oder inaktiv setzen.</p>

    <div v-if="loading" class="text-gray-400 py-10 text-center">Laden…</div>

    <div v-else-if="noRecipient" class="card text-sm text-yellow-700 dark:text-yellow-300 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-700">
      Dein Benutzerkonto ist noch nicht mit einem Teilnehmer verknüpft. Bitte einen Administrator, dein Konto unter <strong>Benutzer → Teilnehmer</strong> zuzuordnen.
    </div>

    <div v-else-if="groups.length === 0" class="card text-sm text-gray-400 dark:text-slate-500 text-center py-10">
      Du bist in keiner Self-Service-Gruppe Mitglied.
    </div>

    <div v-else class="grid grid-cols-1 sm:grid-cols-2 gap-4">
      <div v-for="g in groups" :key="g.id" class="card flex items-center justify-between">
        <div>
          <div class="font-semibold">{{ g.name }}</div>
          <div v-if="g.description" class="text-xs text-gray-400 dark:text-slate-500 mt-0.5">{{ g.description }}</div>
        </div>
        <button
          @click="toggle(g)"
          class="badge cursor-pointer text-sm px-4 py-1.5 transition-colors"
          :class="g.active
            ? 'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300 hover:bg-green-200'
            : 'bg-gray-100 text-gray-500 dark:bg-slate-700 dark:text-slate-400 hover:bg-gray-200'"
        >
          {{ g.active ? 'Aktiv' : 'Inaktiv' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { apiClient } from '../api/index.js'

const groups = ref([])
const loading = ref(true)
const noRecipient = ref(false)

async function load() {
  loading.value = true
  try {
    const res = await apiClient.get('/api/me/groups')
    groups.value = res.data.groups
  } catch (e) {
    if (e.response?.status === 403) noRecipient.value = true
  } finally {
    loading.value = false
  }
}

async function toggle(g) {
  await apiClient.post(`/api/me/groups/${g.id}/active`, { active: !g.active })
  await load()
}

onMounted(load)
</script>
