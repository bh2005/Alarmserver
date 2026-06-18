<template>
  <div>
    <h1 class="text-xl font-bold mb-4">Kanäle</h1>

    <div v-if="restartNeeded" class="mb-4 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-700 rounded-lg px-4 py-3 text-sm text-yellow-800 dark:text-yellow-300">
      Konfiguration gespeichert — Backend-Neustart erforderlich, damit die Änderungen aktiv werden.
    </div>

    <div v-if="loading" class="text-gray-400 py-10 text-center">Laden…</div>
    <div v-else class="grid grid-cols-1 sm:grid-cols-2 gap-4">
      <div v-for="ch in channels" :key="ch.channel" class="card">
        <div class="flex items-center justify-between mb-3">
          <span class="font-semibold text-sm">{{ ch.label }}</span>
          <div class="flex items-center gap-2">
            <span class="badge" :class="ch.enabled
              ? 'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300'
              : 'bg-gray-100 text-gray-500 dark:bg-slate-700 dark:text-slate-400'">
              {{ ch.enabled ? 'aktiv' : 'inaktiv' }}
            </span>
            <button class="btn-ghost text-xs py-1 px-2" @click="openEdit(ch)">Konfigurieren</button>
          </div>
        </div>

        <div class="space-y-1">
          <div v-for="f in ch.fields" :key="f.key" class="flex text-xs gap-2">
            <span class="text-gray-400 dark:text-slate-500 w-36 flex-shrink-0">{{ f.label }}</span>
            <span class="font-mono break-all text-gray-700 dark:text-slate-300">
              {{ f.sensitive
                ? (ch.config[f.key] === '***' ? '●●●●●●' : '–')
                : (ch.config[f.key] || (f.default ? f.default : '–'))
              }}
            </span>
          </div>
        </div>
      </div>
    </div>

    <!-- Edit Modal -->
    <div v-if="editChannel" class="fixed inset-0 bg-gray-900/40 flex items-center justify-center z-50 p-4" @click.self="closeEdit">
      <div class="bg-white dark:bg-slate-800 rounded-xl shadow-2xl p-5 w-full max-w-lg max-h-[90vh] overflow-y-auto">
        <h2 class="text-lg font-bold mb-4">{{ editChannel.label }}</h2>
        <form @submit.prevent="save" class="space-y-3">
          <div class="flex items-center gap-2 pb-1">
            <input type="checkbox" id="ch-enabled" v-model="editForm.enabled" class="w-4 h-4 cursor-pointer" />
            <label for="ch-enabled" class="text-sm font-medium cursor-pointer">Kanal aktiviert</label>
          </div>

          <div v-for="f in editChannel.fields" :key="f.key">
            <label class="label">{{ f.label }}</label>
            <input
              v-model="editForm.config[f.key]"
              :type="f.sensitive ? 'password' : 'text'"
              class="input"
              :placeholder="f.sensitive && originalSensitive[f.key] ? 'Gesetzt — leer lassen zum Beibehalten' : (f.default || '')"
            />
          </div>

          <div v-if="saveError" class="text-red-600 text-sm bg-red-50 border border-red-200 rounded-lg px-3 py-2">{{ saveError }}</div>

          <div class="flex justify-end gap-2 pt-2">
            <button type="button" class="btn-ghost" @click="closeEdit">Abbrechen</button>
            <button type="submit" class="btn-primary" :disabled="saving">{{ saving ? 'Speichern…' : 'Speichern' }}</button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { apiClient } from '../api/index.js'

const channels = ref([])
const loading = ref(true)
const restartNeeded = ref(false)

const editChannel = ref(null)
const editForm = reactive({ enabled: false, config: {} })
const originalSensitive = reactive({})
const saving = ref(false)
const saveError = ref('')

async function load() {
  const res = await apiClient.get('/api/config/channels')
  channels.value = res.data.channels
  loading.value = false
}

function openEdit(ch) {
  editChannel.value = ch
  editForm.enabled = ch.enabled
  editForm.config = {}

  for (const f of ch.fields) {
    if (f.sensitive) {
      originalSensitive[f.key] = ch.config[f.key] === '***'
      editForm.config[f.key] = '' // always empty — leave blank = keep existing
    } else {
      editForm.config[f.key] = ch.config[f.key] || ''
    }
  }
  saveError.value = ''
}

function closeEdit() {
  editChannel.value = null
}

async function save() {
  saving.value = true
  saveError.value = ''

  const config = {}
  for (const f of editChannel.value.fields) {
    if (f.sensitive && editForm.config[f.key] === '' && originalSensitive[f.key]) {
      config[f.key] = '***' // tell backend: preserve existing
    } else {
      config[f.key] = editForm.config[f.key]
    }
  }

  try {
    await apiClient.put(`/api/config/channels/${editChannel.value.channel}`, {
      enabled: editForm.enabled,
      config,
    })
    restartNeeded.value = true
    closeEdit()
    await load()
  } catch (e) {
    saveError.value = e.response?.data || 'Speichern fehlgeschlagen'
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>
