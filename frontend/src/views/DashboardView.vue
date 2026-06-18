<template>
  <div class="p-6 space-y-6 max-w-7xl">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <h1 class="text-xl font-bold text-slate-900 dark:text-white">Dashboard</h1>
      <button @click="load" :disabled="loading"
        class="px-3 py-1.5 text-sm bg-slate-100 dark:bg-slate-700 text-slate-700 dark:text-slate-200 rounded hover:bg-slate-200 dark:hover:bg-slate-600 transition-colors">
        Aktualisieren
      </button>
    </div>

    <!-- Maintenance banner -->
    <div v-if="data.active_maintenance?.length" class="rounded-lg bg-yellow-50 dark:bg-yellow-900/30 border border-yellow-300 dark:border-yellow-700 p-4">
      <div class="font-semibold text-yellow-800 dark:text-yellow-300 mb-2">Aktive Wartungsfenster</div>
      <ul class="space-y-1">
        <li v-for="m in data.active_maintenance" :key="m.id" class="text-sm text-yellow-700 dark:text-yellow-400">
          <span class="font-medium">{{ m.label }}</span>
          <span class="ml-2 text-yellow-600 dark:text-yellow-500">
            {{ fmtDt(m.starts_at) }} – {{ fmtDt(m.ends_at) }}
          </span>
          <span v-if="m.source || m.keyword" class="ml-2 text-xs opacity-75">
            ({{ [m.source, m.keyword].filter(Boolean).join(' / ') }})
          </span>
        </li>
      </ul>
    </div>

    <!-- Heartbeat status row -->
    <div v-if="data.heartbeats?.length" class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-6 gap-3">
      <div v-for="h in data.heartbeats" :key="h.id"
        class="rounded-lg border p-3 text-center"
        :class="statusClass(h.status)">
        <div class="text-xs font-mono mb-1 truncate" :title="h.name">{{ h.name }}</div>
        <div class="text-lg font-bold">
          <span v-if="h.status === 'ok'">✓</span>
          <span v-else-if="h.status === 'dead'">✗</span>
          <span v-else-if="h.status === 'unknown'">?</span>
          <span v-else>–</span>
        </div>
        <div class="text-[10px] uppercase tracking-wide mt-0.5 opacity-75">{{ h.status }}</div>
      </div>
    </div>

    <!-- Open alarms -->
    <div>
      <h2 class="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2">
        Offene Alarme (letzte 24h)
        <span v-if="data.open_events?.length"
          class="ml-2 inline-flex items-center justify-center w-5 h-5 rounded-full bg-red-500 text-white text-[10px] font-bold">
          {{ data.open_events.length }}
        </span>
      </h2>
      <div v-if="!data.open_events?.length"
        class="text-sm text-slate-400 dark:text-slate-500 italic py-4 text-center border border-dashed rounded-lg border-slate-200 dark:border-slate-700">
        Keine offenen Alarme — alles quittiert oder kein Alarm in den letzten 24h.
      </div>
      <div v-else class="rounded-lg border border-slate-200 dark:border-slate-700 overflow-hidden">
        <table class="w-full text-sm">
          <thead class="bg-slate-50 dark:bg-slate-800 text-xs text-slate-500 dark:text-slate-400 uppercase">
            <tr>
              <th class="px-3 py-2 text-left">Zeit</th>
              <th class="px-3 py-2 text-left">Quelle</th>
              <th class="px-3 py-2 text-left">Stichwort</th>
              <th class="px-3 py-2 text-left">Gruppe</th>
              <th class="px-3 py-2 text-left">Kanäle</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100 dark:divide-slate-700/50">
            <tr v-for="e in data.open_events" :key="e.id"
              class="bg-red-50 dark:bg-red-900/10 hover:bg-red-100 dark:hover:bg-red-900/20">
              <td class="px-3 py-2 font-mono text-xs whitespace-nowrap">{{ fmtDt(e.received_at) }}</td>
              <td class="px-3 py-2">{{ e.source }}</td>
              <td class="px-3 py-2 font-medium">{{ e.keyword }}</td>
              <td class="px-3 py-2 text-slate-500 dark:text-slate-400">{{ e.target_group }}</td>
              <td class="px-3 py-2">
                <span v-for="ch in e.channels" :key="ch"
                  class="mr-1 inline-block px-1.5 py-0.5 rounded text-[10px] bg-slate-200 dark:bg-slate-700 text-slate-600 dark:text-slate-300">
                  {{ ch }}
                </span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Recent events -->
    <div>
      <h2 class="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2">Letzte 20 Ereignisse</h2>
      <div class="rounded-lg border border-slate-200 dark:border-slate-700 overflow-hidden">
        <table class="w-full text-sm">
          <thead class="bg-slate-50 dark:bg-slate-800 text-xs text-slate-500 dark:text-slate-400 uppercase">
            <tr>
              <th class="px-3 py-2 text-left">Zeit</th>
              <th class="px-3 py-2 text-left">Quelle</th>
              <th class="px-3 py-2 text-left">Stichwort</th>
              <th class="px-3 py-2 text-left">Status</th>
              <th class="px-3 py-2 text-left">Quittiert</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100 dark:divide-slate-700/50">
            <tr v-if="!data.recent_events?.length">
              <td colspan="5" class="px-3 py-4 text-center text-slate-400 italic">Noch keine Ereignisse</td>
            </tr>
            <tr v-for="e in data.recent_events" :key="e.id"
              class="hover:bg-slate-50 dark:hover:bg-slate-800/50 transition-colors">
              <td class="px-3 py-2 font-mono text-xs whitespace-nowrap">{{ fmtDt(e.received_at) }}</td>
              <td class="px-3 py-2 text-slate-600 dark:text-slate-400">{{ e.source }}</td>
              <td class="px-3 py-2 font-medium">{{ e.keyword }}</td>
              <td class="px-3 py-2">
                <span class="px-1.5 py-0.5 rounded text-[10px] font-medium"
                  :class="statusBadge(e.status)">
                  {{ e.status }}
                </span>
              </td>
              <td class="px-3 py-2">
                <span v-if="e.acknowledged" class="text-green-600 dark:text-green-400 text-xs">✓ Ja</span>
                <span v-else-if="e.status === 'DISPATCHED'" class="text-orange-500 text-xs">Ausstehend</span>
                <span v-else class="text-slate-400 text-xs">–</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'

const data = ref({ recent_events: [], open_events: [], active_maintenance: [], heartbeats: [] })
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    const r = await fetch('/api/dashboard')
    if (r.ok) data.value = await r.json()
  } finally {
    loading.value = false
  }
}

function fmtDt(s) {
  if (!s) return '–'
  return new Date(s).toLocaleString('de-DE', { day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

function statusClass(s) {
  return {
    ok:       'bg-green-50 dark:bg-green-900/20 border-green-300 dark:border-green-700 text-green-800 dark:text-green-300',
    dead:     'bg-red-50 dark:bg-red-900/20 border-red-300 dark:border-red-700 text-red-800 dark:text-red-300',
    unknown:  'bg-slate-50 dark:bg-slate-800 border-slate-200 dark:border-slate-700 text-slate-600 dark:text-slate-400',
    disabled: 'bg-slate-50 dark:bg-slate-800 border-slate-200 dark:border-slate-700 text-slate-400 dark:text-slate-600 opacity-60',
  }[s] || 'bg-slate-50 dark:bg-slate-800 border-slate-200 dark:border-slate-700'
}

function statusBadge(s) {
  return {
    DISPATCHED:  'bg-blue-100 dark:bg-blue-900/40 text-blue-700 dark:text-blue-300',
    NO_MATCH:    'bg-slate-100 dark:bg-slate-700 text-slate-500 dark:text-slate-400',
    ERROR:       'bg-red-100 dark:bg-red-900/40 text-red-700 dark:text-red-300',
    MAINTENANCE: 'bg-yellow-100 dark:bg-yellow-900/40 text-yellow-700 dark:text-yellow-300',
  }[s] || 'bg-slate-100 text-slate-500'
}

onMounted(load)
</script>
