<template>
  <div>
    <div class="flex items-center justify-between mb-4">
      <h1 class="text-xl font-bold">Ansagen <span class="ml-2 text-sm font-normal text-gray-400 dark:text-slate-500">{{ announcements.length }}</span></h1>
    </div>

    <div class="card mb-4">
      <h2 class="text-sm font-semibold mb-2">Neue Ansage hochladen</h2>
      <form @submit.prevent="upload" class="flex items-end gap-3">
        <div class="flex-1">
          <label class="label">Name</label>
          <input v-model="uploadName" class="input" placeholder="z.B. evening-tone" pattern="[a-zA-Z0-9_-]+" required />
        </div>
        <div class="flex-1">
          <label class="label">WAV-Datei (8kHz mono empfohlen)</label>
          <input type="file" accept=".wav" @change="onFile" class="input" required />
        </div>
        <button type="submit" class="btn-primary" :disabled="uploading">{{ uploading ? 'Lädt…' : 'Hochladen' }}</button>
      </form>
      <div v-if="error" class="text-red-600 text-sm bg-red-50 border border-red-200 rounded-lg px-3 py-2 mt-3">{{ error }}</div>
    </div>

    <div class="card">
      <h2 class="text-sm font-semibold mb-3">Verfügbare Ansagen</h2>
      <div v-if="announcements.length === 0" class="text-sm text-gray-400 py-4 text-center">Keine Ansagen vorhanden.</div>
      <ul v-else class="divide-y divide-gray-100 dark:divide-slate-700/60">
        <li v-for="a in announcements" :key="a" class="py-2 font-mono text-sm flex items-center justify-between">
          {{ a }}
          <span class="text-xs text-gray-400">ASTERISK_ANNOUNCEMENT={{ a }}</span>
        </li>
      </ul>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { apiClient } from '../api/index.js'

const announcements = ref([])
const uploadName = ref('')
const file = ref(null)
const uploading = ref(false)
const error = ref('')

async function load() {
  const res = await apiClient.get('/api/announcements')
  announcements.value = res.data.announcements
}

function onFile(e) {
  file.value = e.target.files[0] ?? null
}

async function upload() {
  error.value = ''
  if (!file.value) return
  uploading.value = true
  try {
    const form = new FormData()
    form.append('name', uploadName.value)
    form.append('file', file.value)
    // Content-Type must be left for the browser to set (it needs to add the
    // multipart boundary) — overriding the api client's JSON default to
    // undefined here, rather than to a fixed multipart/form-data string
    // that would be missing that boundary.
    await apiClient.post('/api/announcements', form, { headers: { 'Content-Type': undefined } })
    uploadName.value = ''
    file.value = null
    await load()
  } catch (e) {
    error.value = typeof e.response?.data === 'string' ? e.response.data : 'Upload fehlgeschlagen'
  } finally {
    uploading.value = false
  }
}

onMounted(load)
</script>
