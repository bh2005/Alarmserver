<template>
  <div>
    <div class="flex items-center justify-between mb-4">
      <h1>Eskalationspolitiken</h1>
      <button class="btn-primary" @click="openCreate">+ Neue Politik</button>
    </div>

    <div v-if="loading" class="text-gray-400 py-10 text-center">Laden…</div>
    <div v-else-if="!policies.length" class="card text-sm text-gray-400 py-8 text-center">
      Noch keine Eskalationspolitiken angelegt.
    </div>

    <div v-else class="space-y-4">
      <div v-for="pol in policies" :key="pol.id" class="card">
        <!-- Header -->
        <div class="flex items-start justify-between gap-2 mb-3">
          <div>
            <div class="font-semibold">{{ pol.name }}</div>
            <div v-if="pol.description" class="text-xs text-gray-400 dark:text-slate-500 mt-0.5">{{ pol.description }}</div>
          </div>
          <div class="flex gap-2 flex-shrink-0">
            <button class="btn-ghost text-xs py-1 px-2" @click="openEdit(pol)">Bearbeiten</button>
            <button class="btn-ghost text-xs py-1 px-2 text-red-500 hover:text-red-600" @click="deletePol(pol)">Löschen</button>
          </div>
        </div>

        <!-- Steps -->
        <div class="space-y-2">
          <div v-for="step in pol.steps" :key="step.id"
            class="flex items-center gap-3 text-sm bg-gray-50 dark:bg-slate-700/50 rounded-lg px-3 py-2">
            <span class="w-6 h-6 rounded-full bg-ks-600 text-white text-xs font-bold flex items-center justify-center flex-shrink-0">
              {{ step.step }}
            </span>
            <div class="flex-1 min-w-0">
              <span class="font-medium">{{ step.group }}</span>
              <span class="text-gray-400 dark:text-slate-500 ml-2">nach {{ fmtDelay(step.delay_sec) }}</span>
              <span v-if="step.channels?.length" class="ml-2 text-xs text-ks-600 dark:text-ks-300">
                ({{ step.channels.join(', ') }})
              </span>
              <span v-else class="ml-2 text-xs text-gray-400">Kanäle aus Regel</span>
            </div>
            <button class="text-gray-400 hover:text-red-500 transition-colors p-1" @click="deleteStep(pol, step)" title="Schritt löschen">
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
              </svg>
            </button>
          </div>

          <!-- Add step inline -->
          <div class="border border-dashed border-gray-200 dark:border-slate-600 rounded-lg p-3 space-y-3">
            <div class="grid grid-cols-2 gap-3">
              <div>
                <label class="label">Gruppe</label>
                <select v-model="newStep[pol.id].groupId" class="input text-sm">
                  <option value="">— wählen —</option>
                  <option v-for="g in groups" :key="g.id" :value="g.id">{{ g.name }}</option>
                </select>
              </div>
              <div>
                <label class="label">Nach (Minuten)</label>
                <input v-model.number="newStep[pol.id].delayMin" type="number" min="1" class="input text-sm" placeholder="10" />
              </div>
            </div>
            <div>
              <label class="label">Kanäle (leer = aus Regel übernehmen)</label>
              <div class="flex flex-wrap gap-3 mt-1">
                <label v-for="ch in allChannels" :key="ch"
                  class="flex items-center gap-1.5 text-sm cursor-pointer">
                  <input type="checkbox" :value="ch" v-model="newStep[pol.id].channels" class="w-3.5 h-3.5" />
                  {{ ch }}
                </label>
              </div>
            </div>
            <div class="flex justify-end">
              <button class="btn-primary text-sm py-1.5 px-3" @click="addStep(pol)" :disabled="!newStep[pol.id].groupId">
                + Schritt hinzufügen
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Create/Edit Modal -->
    <div v-if="modal" class="fixed inset-0 bg-gray-900/40 flex items-center justify-center z-50 p-4" @click.self="closeModal">
      <div class="bg-white dark:bg-slate-800 rounded-xl shadow-2xl p-5 w-full max-w-md">
        <h2 class="text-lg font-bold mb-4">{{ modal.id ? 'Politik bearbeiten' : 'Neue Eskalationspolitik' }}</h2>
        <form @submit.prevent="saveModal" class="space-y-3">
          <div>
            <label class="label">Name</label>
            <input v-model="modal.name" class="input" required placeholder="z.B. Standard-Eskalation" />
          </div>
          <div>
            <label class="label">Beschreibung (optional)</label>
            <input v-model="modal.description" class="input" placeholder="Kurze Erklärung" />
          </div>
          <div v-if="modalError" class="text-red-600 dark:text-red-400 text-sm bg-red-50 dark:bg-red-950/50 border border-red-200 dark:border-red-800 rounded-lg px-3 py-2">{{ modalError }}</div>
          <div class="flex justify-end gap-2 pt-1">
            <button type="button" class="btn-ghost" @click="closeModal">Abbrechen</button>
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

const policies  = ref([])
const groups    = ref([])
const allChannels = ref([])
const loading   = ref(true)
const modal     = ref(null)
const modalError = ref('')
const saving    = ref(false)
const newStep   = reactive({})

function ensureNewStep(polId) {
  if (!newStep[polId]) newStep[polId] = { groupId: '', delayMin: 10, channels: [] }
}

async function load() {
  loading.value = true
  const [polRes, grpRes, whRes] = await Promise.all([
    apiClient.get('/api/escalation-policies'),
    apiClient.get('/api/groups'),
    apiClient.get('/api/webhook-targets'),
  ])
  policies.value  = polRes.data ?? []
  groups.value    = grpRes.data?.groups ?? grpRes.data ?? []
  const staticChannels = ['voice', 'voice-smseagle', 'sms', 'email', 'teams']
  const webhookNames   = (whRes.data ?? []).filter(t => t.active).map(t => t.name)
  allChannels.value    = [...staticChannels, ...webhookNames]
  policies.value.forEach(p => ensureNewStep(p.id))
  loading.value = false
}

function fmtDelay(sec) {
  if (sec < 60) return `${sec}s`
  const m = Math.round(sec / 60)
  return m === 1 ? '1 Minute' : `${m} Minuten`
}

function openCreate() {
  modal.value = { id: null, name: '', description: '' }
  modalError.value = ''
}
function openEdit(pol) {
  modal.value = { id: pol.id, name: pol.name, description: pol.description || '' }
  modalError.value = ''
}
function closeModal() { modal.value = null }

async function saveModal() {
  saving.value = true
  modalError.value = ''
  try {
    if (modal.value.id) {
      await apiClient.put(`/api/escalation-policies/${modal.value.id}`, {
        name: modal.value.name, description: modal.value.description,
      })
    } else {
      await apiClient.post('/api/escalation-policies', {
        name: modal.value.name, description: modal.value.description,
      })
    }
    closeModal()
    await load()
  } catch (e) {
    modalError.value = e.response?.data || 'Fehler beim Speichern'
  } finally {
    saving.value = false
  }
}

async function deletePol(pol) {
  if (!confirm(`Eskalationspolitik "${pol.name}" löschen?`)) return
  await apiClient.delete(`/api/escalation-policies/${pol.id}`)
  await load()
}

async function addStep(pol) {
  const s = newStep[pol.id]
  if (!s.groupId) return
  const nextStep = (pol.steps?.length ?? 0) + 1
  await apiClient.post(`/api/escalation-policies/${pol.id}/steps`, {
    step: nextStep,
    group_id: s.groupId,
    delay_sec: (s.delayMin || 10) * 60,
    channels: s.channels,
  })
  newStep[pol.id] = { groupId: '', delayMin: 10, channels: [] }
  await load()
}

async function deleteStep(pol, step) {
  await apiClient.delete(`/api/escalation-policies/${pol.id}/steps/${step.id}`)
  await load()
}

onMounted(load)
</script>
