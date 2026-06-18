<template>
  <div class="p-6 max-w-4xl space-y-4">
    <div class="flex items-center justify-between">
      <h1 class="text-xl font-bold text-slate-900 dark:text-white">Wartungsfenster</h1>
      <button @click="openForm(null)"
        class="px-3 py-1.5 text-sm bg-ks-600 hover:bg-ks-700 text-white rounded transition-colors">
        + Neu
      </button>
    </div>

    <p class="text-sm text-slate-500 dark:text-slate-400">
      Während eines aktiven Wartungsfensters werden passende Alarme unterdrückt und als
      <code class="bg-slate-100 dark:bg-slate-700 px-1 rounded text-xs">MAINTENANCE</code> im Log gespeichert.
      Quelle und Stichwort sind optional — leer bedeutet "gilt für alle".
    </p>

    <!-- Table -->
    <div class="rounded-lg border border-slate-200 dark:border-slate-700 overflow-hidden">
      <table class="w-full text-sm">
        <thead class="bg-slate-50 dark:bg-slate-800 text-xs text-slate-500 dark:text-slate-400 uppercase">
          <tr>
            <th class="px-4 py-2 text-left">Bezeichnung</th>
            <th class="px-4 py-2 text-left">Start</th>
            <th class="px-4 py-2 text-left">Ende</th>
            <th class="px-4 py-2 text-left">Quelle</th>
            <th class="px-4 py-2 text-left">Stichwort</th>
            <th class="px-4 py-2 text-left">Status</th>
            <th class="px-4 py-2"></th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-100 dark:divide-slate-700/50">
          <tr v-if="!windows.length">
            <td colspan="7" class="px-4 py-6 text-center text-slate-400 italic">Keine Wartungsfenster angelegt</td>
          </tr>
          <tr v-for="w in windows" :key="w.id"
            class="hover:bg-slate-50 dark:hover:bg-slate-800/50 transition-colors">
            <td class="px-4 py-2 font-medium">{{ w.label }}</td>
            <td class="px-4 py-2 font-mono text-xs">{{ fmtDt(w.starts_at) }}</td>
            <td class="px-4 py-2 font-mono text-xs">{{ fmtDt(w.ends_at) }}</td>
            <td class="px-4 py-2 text-slate-500 dark:text-slate-400">{{ w.source || '–' }}</td>
            <td class="px-4 py-2 text-slate-500 dark:text-slate-400">{{ w.keyword || '–' }}</td>
            <td class="px-4 py-2">
              <span class="px-2 py-0.5 rounded-full text-[10px] font-medium"
                :class="windowStatus(w) === 'Aktiv'
                  ? 'bg-green-100 dark:bg-green-900/40 text-green-700 dark:text-green-300'
                  : windowStatus(w) === 'Ausstehend'
                    ? 'bg-blue-100 dark:bg-blue-900/40 text-blue-600 dark:text-blue-300'
                    : 'bg-slate-100 dark:bg-slate-700 text-slate-500 dark:text-slate-400'">
                {{ windowStatus(w) }}
              </span>
            </td>
            <td class="px-4 py-2 text-right">
              <button @click="openForm(w)" class="text-xs text-ks-600 dark:text-ks-400 hover:underline mr-3">Bearbeiten</button>
              <button @click="del(w.id)" class="text-xs text-red-500 hover:underline">Löschen</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Modal -->
    <div v-if="form" class="fixed inset-0 bg-black/40 flex items-center justify-center z-50 p-4">
      <div class="bg-white dark:bg-slate-800 rounded-xl shadow-xl w-full max-w-lg p-6 space-y-4">
        <h2 class="font-bold text-lg text-slate-900 dark:text-white">
          {{ form.id ? 'Wartungsfenster bearbeiten' : 'Neues Wartungsfenster' }}
        </h2>
        <div class="space-y-3">
          <label class="block">
            <span class="text-xs font-medium text-slate-600 dark:text-slate-400">Bezeichnung *</span>
            <input v-model="form.label" class="mt-1 w-full input" placeholder="z.B. Netzwartung Rechenzentrum" />
          </label>
          <div class="grid grid-cols-2 gap-3">
            <label class="block">
              <span class="text-xs font-medium text-slate-600 dark:text-slate-400">Start *</span>
              <input v-model="form.starts_at" type="datetime-local" class="mt-1 w-full input" />
            </label>
            <label class="block">
              <span class="text-xs font-medium text-slate-600 dark:text-slate-400">Ende *</span>
              <input v-model="form.ends_at" type="datetime-local" class="mt-1 w-full input" />
            </label>
          </div>
          <div class="grid grid-cols-2 gap-3">
            <label class="block">
              <span class="text-xs font-medium text-slate-600 dark:text-slate-400">Quelle (leer = alle)</span>
              <input v-model="form.source" class="mt-1 w-full input" placeholder="WEBHOOK" />
            </label>
            <label class="block">
              <span class="text-xs font-medium text-slate-600 dark:text-slate-400">Stichwort (leer = alle)</span>
              <input v-model="form.keyword" class="mt-1 w-full input" placeholder="PING" />
            </label>
          </div>
        </div>
        <p v-if="error" class="text-sm text-red-500">{{ error }}</p>
        <div class="flex gap-2 justify-end pt-2">
          <button @click="form = null; error = ''" class="px-4 py-2 text-sm text-slate-600 dark:text-slate-300 hover:underline">Abbrechen</button>
          <button @click="save" :disabled="saving"
            class="px-4 py-2 text-sm bg-ks-600 hover:bg-ks-700 text-white rounded transition-colors disabled:opacity-50">
            {{ saving ? 'Speichern...' : 'Speichern' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'

const windows = ref([])
const form    = ref(null)
const error   = ref('')
const saving  = ref(false)

async function load() {
  const r = await fetch('/api/maintenance')
  if (r.ok) windows.value = (await r.json()).windows || []
}

function fmtDt(s) {
  if (!s) return '–'
  return new Date(s).toLocaleString('de-DE', { day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit' })
}

function toLocalDTInput(s) {
  if (!s) return ''
  const d = new Date(s)
  d.setMinutes(d.getMinutes() - d.getTimezoneOffset())
  return d.toISOString().slice(0, 16)
}

function windowStatus(w) {
  const now = Date.now()
  const s   = new Date(w.starts_at).getTime()
  const e   = new Date(w.ends_at).getTime()
  if (now >= s && now < e) return 'Aktiv'
  if (now < s) return 'Ausstehend'
  return 'Abgelaufen'
}

function openForm(w) {
  error.value = ''
  if (w) {
    form.value = { id: w.id, label: w.label, starts_at: toLocalDTInput(w.starts_at), ends_at: toLocalDTInput(w.ends_at), source: w.source || '', keyword: w.keyword || '' }
  } else {
    const now = new Date(); now.setMinutes(now.getMinutes() - now.getTimezoneOffset())
    const later = new Date(now.getTime() + 3600000); later.setMinutes(later.getMinutes())
    form.value = { label: '', starts_at: now.toISOString().slice(0,16), ends_at: later.toISOString().slice(0,16), source: '', keyword: '' }
  }
}

async function save() {
  error.value = ''
  if (!form.value.label || !form.value.starts_at || !form.value.ends_at) {
    error.value = 'Bezeichnung, Start und Ende sind Pflichtfelder.'; return
  }
  const body = {
    label:     form.value.label,
    starts_at: new Date(form.value.starts_at).toISOString(),
    ends_at:   new Date(form.value.ends_at).toISOString(),
    source:    form.value.source || null,
    keyword:   form.value.keyword || null,
  }
  saving.value = true
  try {
    const url = form.value.id ? `/api/maintenance/${form.value.id}` : '/api/maintenance'
    const method = form.value.id ? 'PUT' : 'POST'
    const r = await fetch(url, { method, headers: {'Content-Type': 'application/json'}, body: JSON.stringify(body) })
    if (!r.ok) { error.value = await r.text(); return }
    form.value = null
    await load()
  } finally {
    saving.value = false
  }
}

async function del(id) {
  if (!confirm('Wartungsfenster wirklich löschen?')) return
  await fetch(`/api/maintenance/${id}`, { method: 'DELETE' })
  await load()
}

onMounted(load)
</script>

<style scoped>
.input {
  @apply block w-full rounded-md border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-700 text-slate-900 dark:text-white px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ks-500;
}
</style>
