<template>
  <div>
    <div class="flex items-center justify-between mb-4">
      <div>
        <h1>Webhook-Ziele</h1>
        <p class="text-sm text-gray-400 dark:text-slate-500 mt-0.5">
          Jedes Ziel erscheint als Kanal (= Name) und kann in Regeln verwendet werden. Backend-Neustart nach Änderungen erforderlich.
        </p>
      </div>
      <button class="btn-primary" @click="openCreate">+ Neues Ziel</button>
    </div>

    <div v-if="restartNeeded" class="mb-4 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-700 rounded-lg px-4 py-3 text-sm text-yellow-800 dark:text-yellow-300">
      Webhook-Ziel geändert — Backend-Neustart erforderlich, damit der Kanal aktiv wird.
    </div>

    <div v-if="loading" class="text-gray-400 py-10 text-center">Laden…</div>
    <div v-else-if="!targets.length" class="card text-sm text-gray-400 py-8 text-center">
      Noch keine Webhook-Ziele angelegt.
    </div>

    <div v-else class="space-y-3">
      <div v-for="t in targets" :key="t.id"
        class="card flex items-start gap-3">
        <div class="flex-1 min-w-0">
          <div class="flex items-center gap-2 mb-1">
            <span class="font-semibold text-sm">{{ t.label }}</span>
            <code class="text-xs bg-gray-100 dark:bg-slate-700 text-ks-600 dark:text-ks-300 px-1.5 py-0.5 rounded">{{ t.name }}</code>
            <span class="badge text-xs"
              :class="t.active
                ? 'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300'
                : 'bg-gray-100 text-gray-500 dark:bg-slate-700 dark:text-slate-400'">
              {{ t.active ? 'aktiv' : 'inaktiv' }}
            </span>
          </div>
          <div class="text-xs text-gray-500 dark:text-slate-400 font-mono truncate">
            <span class="text-gray-400 mr-1">{{ t.method }}</span>{{ t.url }}
          </div>
          <div v-if="t.body_template" class="text-xs text-gray-400 dark:text-slate-500 mt-1 font-mono truncate">
            Body: {{ t.body_template }}
          </div>
        </div>
        <div class="flex gap-2 flex-shrink-0">
          <button class="btn-ghost text-xs py-1 px-2" @click="openEdit(t)">Bearbeiten</button>
          <button class="btn-ghost text-xs py-1 px-2 text-red-500" @click="deleteTarget(t)">Löschen</button>
        </div>
      </div>
    </div>

    <!-- Modal -->
    <div v-if="form" class="fixed inset-0 bg-gray-900/40 flex items-center justify-center z-50 p-4" @click.self="closeForm">
      <div class="bg-white dark:bg-slate-800 rounded-xl shadow-2xl p-5 w-full max-w-lg max-h-[90vh] overflow-y-auto">
        <h2 class="text-lg font-bold mb-4">{{ form.id ? 'Webhook-Ziel bearbeiten' : 'Neues Webhook-Ziel' }}</h2>
        <form @submit.prevent="save" class="space-y-3">
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="label">Name (Kanalname in Regeln)</label>
              <input v-model="form.name" class="input" required placeholder="slack" pattern="[a-z0-9_-]+" />
              <p class="text-xs text-gray-400 mt-0.5">Kleinbuchstaben, Zahlen, _ oder -</p>
            </div>
            <div>
              <label class="label">Anzeigename</label>
              <input v-model="form.label" class="input" required placeholder="Slack #alarms" />
            </div>
          </div>
          <div>
            <label class="label">URL</label>
            <input v-model="form.url" class="input" required placeholder="https://hooks.slack.com/..." />
          </div>
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="label">HTTP-Methode</label>
              <select v-model="form.method" class="input">
                <option>POST</option>
                <option>PUT</option>
                <option>PATCH</option>
              </select>
            </div>
            <div>
              <label class="label">Timeout (Sekunden)</label>
              <input v-model.number="form.timeout_sec" type="number" min="1" max="60" class="input" />
            </div>
          </div>
          <div>
            <label class="label">Body-Template (optional)</label>
            <textarea v-model="form.body_template" class="input font-mono text-xs" rows="3"
              placeholder='{"text":"{message}"}&#10;Platzhalter: {message} {keyword} {source} {group} {time}'></textarea>
            <p class="text-xs text-gray-400 mt-0.5">Leer lassen für Standard-JSON. Platzhalter: <code class="text-ks-600 dark:text-ks-300">{message}</code></p>
          </div>
          <div>
            <label class="label">Zusatz-Header (JSON, optional)</label>
            <input v-model="form.headersRaw" class="input font-mono text-xs" placeholder='{"Authorization":"Bearer token123"}' />
          </div>
          <div class="flex items-center gap-2">
            <input type="checkbox" id="wh-active" v-model="form.active" class="w-4 h-4" />
            <label for="wh-active" class="text-sm cursor-pointer">Aktiv</label>
          </div>
          <div v-if="saveError" class="text-red-600 dark:text-red-400 text-sm bg-red-50 dark:bg-red-950/50 border border-red-200 dark:border-red-800 rounded-lg px-3 py-2">{{ saveError }}</div>
          <div class="flex justify-end gap-2 pt-1">
            <button type="button" class="btn-ghost" @click="closeForm">Abbrechen</button>
            <button type="submit" class="btn-primary" :disabled="saving">{{ saving ? 'Speichern…' : 'Speichern' }}</button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { apiClient } from '../api/index.js'

const targets      = ref([])
const loading      = ref(true)
const restartNeeded = ref(false)
const form         = ref(null)
const saveError    = ref('')
const saving       = ref(false)

async function load() {
  const res = await apiClient.get('/api/webhook-targets')
  targets.value = res.data
  loading.value = false
}

function openCreate() {
  form.value = { id: null, name: '', label: '', url: '', method: 'POST',
    timeout_sec: 10, body_template: '', headersRaw: '', active: true }
  saveError.value = ''
}

function openEdit(t) {
  form.value = {
    id: t.id, name: t.name, label: t.label, url: t.url,
    method: t.method, timeout_sec: t.timeout_sec,
    body_template: t.body_template || '',
    headersRaw: Object.keys(t.headers || {}).length ? JSON.stringify(t.headers) : '',
    active: t.active,
  }
  saveError.value = ''
}

function closeForm() { form.value = null }

async function save() {
  saving.value = true
  saveError.value = ''
  let headers = {}
  if (form.value.headersRaw) {
    try { headers = JSON.parse(form.value.headersRaw) } catch {
      saveError.value = 'Header-JSON ungültig'
      saving.value = false
      return
    }
  }
  const payload = {
    name: form.value.name, label: form.value.label, url: form.value.url,
    method: form.value.method, timeout_sec: form.value.timeout_sec,
    body_template: form.value.body_template, headers, active: form.value.active,
  }
  try {
    if (form.value.id) {
      await apiClient.put(`/api/webhook-targets/${form.value.id}`, payload)
    } else {
      await apiClient.post('/api/webhook-targets', payload)
    }
    restartNeeded.value = true
    closeForm()
    await load()
  } catch (e) {
    saveError.value = e.response?.data || 'Fehler beim Speichern'
  } finally {
    saving.value = false
  }
}

async function deleteTarget(t) {
  if (!confirm(`Webhook-Ziel "${t.label}" löschen?`)) return
  await apiClient.delete(`/api/webhook-targets/${t.id}`)
  restartNeeded.value = true
  await load()
}

onMounted(load)
</script>
