<template>
  <div>
    <div class="flex items-center justify-between mb-4">
      <h1 class="text-xl font-bold">API Keys <span class="ml-2 text-sm font-normal text-gray-400 dark:text-slate-500">{{ keys.length }}</span></h1>
      <button class="btn-primary" @click="modalOpen = true">+ API Key</button>
    </div>

    <div class="card overflow-hidden p-0">
      <table class="w-full text-sm">
        <thead>
          <tr class="bg-gray-50 dark:bg-slate-700/60 border-b border-gray-200 dark:border-slate-700 text-left">
            <th class="th">Name</th>
            <th class="th">Erstellt</th>
            <th class="th text-center">Status</th>
            <th class="th w-24"></th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-100 dark:divide-slate-700/60">
          <tr v-if="keys.length === 0">
            <td colspan="4" class="td text-center text-gray-400 dark:text-slate-500 py-8">Keine API Keys vorhanden</td>
          </tr>
          <tr v-for="k in keys" :key="k.id" class="hover:bg-gray-50 dark:hover:bg-slate-700/30">
            <td class="td font-mono font-medium">{{ k.name }}</td>
            <td class="td text-gray-500 dark:text-slate-400">{{ fmtDate(k.created_at) }}</td>
            <td class="td text-center">
              <button
                class="badge cursor-pointer"
                :class="k.active
                  ? 'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300'
                  : 'bg-gray-100 text-gray-500 dark:bg-slate-700 dark:text-slate-400'"
                @click="toggleActive(k)"
              >{{ k.active ? 'aktiv' : 'deaktiviert' }}</button>
            </td>
            <td class="td text-right">
              <button class="text-red-600 hover:underline text-xs" @click="remove(k)">Löschen</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Create Modal -->
    <div v-if="modalOpen" class="fixed inset-0 bg-gray-900/40 flex items-center justify-center z-50 p-4" @click.self="modalOpen = false">
      <div class="bg-white dark:bg-slate-800 rounded-xl shadow-2xl p-5 w-full max-w-md">
        <h2 class="text-lg font-bold mb-4">Neuen API Key anlegen</h2>
        <form @submit.prevent="create" class="space-y-3">
          <div><label class="label">Name</label><input v-model="form.name" class="input" placeholder="z. B. monitoring-bot" required /></div>
          <div v-if="createError" class="text-red-600 text-sm bg-red-50 border border-red-200 rounded-lg px-3 py-2">{{ createError }}</div>
          <div class="flex justify-end gap-2 pt-2">
            <button type="button" class="btn-ghost" @click="modalOpen = false">Abbrechen</button>
            <button type="submit" class="btn-primary">Anlegen</button>
          </div>
        </form>
      </div>
    </div>

    <!-- Show Key Once Modal -->
    <div v-if="newKey" class="fixed inset-0 bg-gray-900/40 flex items-center justify-center z-50 p-4">
      <div class="bg-white dark:bg-slate-800 rounded-xl shadow-2xl p-5 w-full max-w-lg">
        <h2 class="text-lg font-bold mb-2">API Key wurde angelegt</h2>
        <p class="text-sm text-gray-600 dark:text-slate-400 mb-4">
          Dieser Key wird <strong>nur einmal</strong> angezeigt. Kopiere ihn jetzt — er kann nicht wiederhergestellt werden.
        </p>
        <div class="bg-gray-50 dark:bg-slate-900 border border-gray-200 dark:border-slate-700 rounded-lg p-3 mb-4 flex items-center gap-2">
          <code class="text-xs font-mono break-all flex-1 text-gray-800 dark:text-slate-200">{{ newKey }}</code>
          <button @click="copyKey" class="btn-ghost text-xs py-1 px-2 flex-shrink-0">{{ copied ? 'Kopiert!' : 'Kopieren' }}</button>
        </div>
        <div class="flex justify-end">
          <button class="btn-primary" @click="newKey = null; copied = false">Verstanden</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { apiClient } from '../api/index.js'

const keys = ref([])
const modalOpen = ref(false)
const newKey = ref(null)
const copied = ref(false)
const createError = ref('')
const form = reactive({ name: '' })

async function load() {
  const res = await apiClient.get('/api/apikeys')
  keys.value = res.data.api_keys
}

async function create() {
  createError.value = ''
  try {
    const res = await apiClient.post('/api/apikeys', { name: form.name })
    modalOpen.value = false
    form.name = ''
    newKey.value = res.data.key
    await load()
  } catch (e) {
    createError.value = e.response?.data || 'Anlegen fehlgeschlagen (Name bereits vergeben?)'
  }
}

async function toggleActive(k) {
  await apiClient.post(`/api/apikeys/${k.id}/active`, { active: !k.active })
  await load()
}

async function remove(k) {
  if (!confirm(`API Key "${k.name}" löschen?`)) return
  await apiClient.delete(`/api/apikeys/${k.id}`)
  await load()
}

async function copyKey() {
  await navigator.clipboard.writeText(newKey.value)
  copied.value = true
}

function fmtDate(iso) {
  return new Date(iso).toLocaleString('de-DE', { dateStyle: 'short', timeStyle: 'short' })
}

onMounted(load)
</script>
