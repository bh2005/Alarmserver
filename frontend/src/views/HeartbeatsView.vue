<template>
  <div class="p-6 max-w-5xl space-y-4">
    <div class="flex items-center justify-between">
      <h1 class="text-xl font-bold text-slate-900 dark:text-white">Heartbeat-Überwachung</h1>
      <button @click="openForm(null)"
        class="px-3 py-1.5 text-sm bg-ks-600 hover:bg-ks-700 text-white rounded transition-colors">
        + Neu
      </button>
    </div>

    <p class="text-sm text-slate-500 dark:text-slate-400">
      Heartbeat-Monitore warten auf regelmäßige Pings von externen Systemen. Bleibt ein Ping aus,
      wird ein Alarm-Event erzeugt (Quelle + Stichwort gemäß Konfiguration) und über die passende Regel weitergeleitet.
      Ping-Endpunkt: <code class="bg-slate-100 dark:bg-slate-700 px-1 rounded text-xs">POST /api/heartbeat/{name}</code>
    </p>

    <!-- Status cards -->
    <div v-if="heartbeats.length" class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-3">
      <div v-for="h in heartbeats" :key="h.id"
        class="rounded-lg border p-4 text-center"
        :class="cardClass(h.status)">
        <div class="text-sm font-bold truncate" :title="h.name">{{ h.name }}</div>
        <div class="text-2xl mt-2">
          <span v-if="h.status === 'ok'">✓</span>
          <span v-else-if="h.status === 'dead'">✗</span>
          <span v-else-if="h.status === 'unknown'">?</span>
          <span v-else>–</span>
        </div>
        <div class="text-[10px] uppercase tracking-wider mt-1 opacity-75">{{ h.status }}</div>
        <div v-if="h.last_seen" class="text-[9px] mt-1 opacity-60">{{ fmtAgo(h.last_seen) }}</div>
        <div v-else class="text-[9px] mt-1 opacity-60">Noch kein Ping</div>
      </div>
    </div>

    <!-- Table -->
    <div class="rounded-lg border border-slate-200 dark:border-slate-700 overflow-hidden">
      <table class="w-full text-sm">
        <thead class="bg-slate-50 dark:bg-slate-800 text-xs text-slate-500 dark:text-slate-400 uppercase">
          <tr>
            <th class="px-4 py-2 text-left">Name</th>
            <th class="px-4 py-2 text-left">Interval / Grace</th>
            <th class="px-4 py-2 text-left">Quelle → Stichwort</th>
            <th class="px-4 py-2 text-left">Letzter Ping</th>
            <th class="px-4 py-2 text-left">Aktiv</th>
            <th class="px-4 py-2"></th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-100 dark:divide-slate-700/50">
          <tr v-if="!heartbeats.length">
            <td colspan="6" class="px-4 py-6 text-center text-slate-400 italic">Keine Heartbeat-Monitore konfiguriert</td>
          </tr>
          <tr v-for="h in heartbeats" :key="h.id"
            class="hover:bg-slate-50 dark:hover:bg-slate-800/50 transition-colors">
            <td class="px-4 py-2">
              <div class="font-medium">{{ h.name }}</div>
              <div v-if="h.description" class="text-xs text-slate-400">{{ h.description }}</div>
            </td>
            <td class="px-4 py-2 font-mono text-xs">{{ h.interval_sec }}s / {{ h.grace_sec }}s</td>
            <td class="px-4 py-2 text-xs text-slate-500 dark:text-slate-400">
              {{ h.source }} → {{ h.keyword }}
            </td>
            <td class="px-4 py-2 text-xs">
              <span v-if="h.last_seen">{{ fmtDt(h.last_seen) }}</span>
              <span v-else class="text-slate-400 italic">Noch nie</span>
            </td>
            <td class="px-4 py-2">
              <span :class="h.active ? 'text-green-600 dark:text-green-400' : 'text-slate-400'">
                {{ h.active ? 'Ja' : 'Nein' }}
              </span>
            </td>
            <td class="px-4 py-2 text-right">
              <button @click="openForm(h)" class="text-xs text-ks-600 dark:text-ks-400 hover:underline mr-3">Bearbeiten</button>
              <button @click="del(h.id)" class="text-xs text-red-500 hover:underline">Löschen</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Modal -->
    <div v-if="form" class="fixed inset-0 bg-black/40 flex items-center justify-center z-50 p-4">
      <div class="bg-white dark:bg-slate-800 rounded-xl shadow-xl w-full max-w-lg p-6 space-y-4">
        <h2 class="font-bold text-lg text-slate-900 dark:text-white">
          {{ form.id ? 'Heartbeat bearbeiten' : 'Neuer Heartbeat-Monitor' }}
        </h2>
        <div class="space-y-3">
          <label class="block">
            <span class="text-xs font-medium text-slate-600 dark:text-slate-400">Name * (eindeutig, wird in der Ping-URL verwendet)</span>
            <input v-model="form.name" class="mt-1 w-full input" :disabled="!!form.id" placeholder="mein-server-ping" />
          </label>
          <label class="block">
            <span class="text-xs font-medium text-slate-600 dark:text-slate-400">Beschreibung</span>
            <input v-model="form.description" class="mt-1 w-full input" placeholder="Optional" />
          </label>
          <div class="grid grid-cols-2 gap-3">
            <label class="block">
              <span class="text-xs font-medium text-slate-600 dark:text-slate-400">Interval (Sek.) *</span>
              <input v-model.number="form.interval_sec" type="number" min="30" class="mt-1 w-full input" />
            </label>
            <label class="block">
              <span class="text-xs font-medium text-slate-600 dark:text-slate-400">Toleranz (Sek.) *</span>
              <input v-model.number="form.grace_sec" type="number" min="0" class="mt-1 w-full input" />
            </label>
          </div>
          <div class="grid grid-cols-2 gap-3">
            <label class="block">
              <span class="text-xs font-medium text-slate-600 dark:text-slate-400">Quelle (für Alarm-Event) *</span>
              <input v-model="form.source" class="mt-1 w-full input" placeholder="heartbeat" />
            </label>
            <label class="block">
              <span class="text-xs font-medium text-slate-600 dark:text-slate-400">Stichwort * (muss Regel matchen)</span>
              <input v-model="form.keyword" class="mt-1 w-full input" placeholder="HEARTBEAT_FAIL" />
            </label>
          </div>
          <label class="flex items-center gap-2 cursor-pointer">
            <input type="checkbox" v-model="form.active" class="w-4 h-4 rounded" />
            <span class="text-sm text-slate-700 dark:text-slate-300">Aktiv</span>
          </label>
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

const heartbeats = ref([])
const form       = ref(null)
const error      = ref('')
const saving     = ref(false)

async function load() {
  const r = await fetch('/api/heartbeats')
  if (r.ok) heartbeats.value = (await r.json()).heartbeats || []
}

function fmtDt(s) {
  return new Date(s).toLocaleString('de-DE', { day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

function fmtAgo(s) {
  const diff = Math.round((Date.now() - new Date(s).getTime()) / 1000)
  if (diff < 60) return `vor ${diff}s`
  if (diff < 3600) return `vor ${Math.floor(diff/60)}min`
  return `vor ${Math.floor(diff/3600)}h`
}

function cardClass(status) {
  return {
    ok:       'border-green-300 dark:border-green-700 bg-green-50 dark:bg-green-900/20 text-green-800 dark:text-green-300',
    dead:     'border-red-300 dark:border-red-700 bg-red-50 dark:bg-red-900/20 text-red-800 dark:text-red-300',
    unknown:  'border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800 text-slate-600 dark:text-slate-400',
    disabled: 'border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800 text-slate-400 opacity-60',
  }[status] || 'border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800'
}

function openForm(h) {
  error.value = ''
  if (h) {
    form.value = { id: h.id, name: h.name, description: h.description || '', interval_sec: h.interval_sec, grace_sec: h.grace_sec, source: h.source, keyword: h.keyword, active: h.active }
  } else {
    form.value = { name: '', description: '', interval_sec: 300, grace_sec: 60, source: 'heartbeat', keyword: '', active: true }
  }
}

async function save() {
  error.value = ''
  if (!form.value.name || !form.value.keyword) { error.value = 'Name und Stichwort sind Pflichtfelder.'; return }
  saving.value = true
  try {
    const url = form.value.id ? `/api/heartbeats/${form.value.id}` : '/api/heartbeats'
    const method = form.value.id ? 'PUT' : 'POST'
    const r = await fetch(url, { method, headers: {'Content-Type': 'application/json'}, body: JSON.stringify(form.value) })
    if (!r.ok) { error.value = await r.text(); return }
    form.value = null
    await load()
  } finally {
    saving.value = false
  }
}

async function del(id) {
  if (!confirm('Heartbeat-Monitor wirklich löschen?')) return
  await fetch(`/api/heartbeats/${id}`, { method: 'DELETE' })
  await load()
}

onMounted(load)
</script>

<style scoped>
.input {
  @apply block w-full rounded-md border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-700 text-slate-900 dark:text-white px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ks-500 disabled:opacity-50;
}
</style>
