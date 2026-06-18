<template>
  <div>
    <div class="flex items-center justify-between mb-4">
      <h1 class="text-xl font-bold">Event-Log <span class="ml-2 text-sm font-normal text-gray-400 dark:text-slate-500">{{ events.length }}</span></h1>
      <button class="btn-ghost" @click="load" :disabled="loading">{{ loading ? 'Lädt…' : 'Aktualisieren' }}</button>
    </div>

    <div class="card overflow-hidden p-0">
      <table class="w-full text-sm">
        <thead>
          <tr class="bg-gray-50 dark:bg-slate-700/60 border-b border-gray-200 dark:border-slate-700 text-left">
            <th class="th">Zeit</th>
            <th class="th">Quelle</th>
            <th class="th">Keyword</th>
            <th class="th">Zielgruppe</th>
            <th class="th">Kanäle</th>
            <th class="th">Status</th>
            <th class="th w-24"></th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-100 dark:divide-slate-700/60">
          <tr v-if="loading"><td colspan="7" class="py-10 text-center text-gray-400">Laden…</td></tr>
          <tr v-else-if="events.length === 0"><td colspan="7" class="py-10 text-center text-gray-400">Keine Events</td></tr>
          <tr v-for="e in events" :key="e.id" class="hover:bg-gray-50 dark:hover:bg-slate-700/30">
            <td class="td text-xs font-mono whitespace-nowrap">{{ fmtTs(e.received_at) }}</td>
            <td class="td font-mono text-xs">{{ e.source }}</td>
            <td class="td font-mono text-xs">{{ e.keyword }}</td>
            <td class="td">{{ e.target_group || '–' }}</td>
            <td class="td"><span v-for="c in e.channels || []" :key="c" class="badge bg-ks-50 text-ks-700 dark:bg-slate-700 dark:text-ks-300 mr-1">{{ c }}</span></td>
            <td class="td">
              <span class="badge" :class="statusClass(e.status)">{{ e.status }}</span>
              <div v-if="e.error_detail" class="text-xs text-red-500 mt-0.5">{{ e.error_detail }}</div>
            </td>
            <td class="td text-right">
              <button class="text-ks-600 dark:text-ks-300 hover:underline text-xs" @click="openAcks(e)">Acks</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-if="acksModalOpen" class="fixed inset-0 bg-gray-900/40 flex items-center justify-center z-50 p-4" @click.self="acksModalOpen = false">
      <div class="bg-white dark:bg-slate-800 rounded-xl shadow-2xl p-5 w-full max-w-md max-h-[80vh] overflow-y-auto">
        <h2 class="text-lg font-bold mb-1">Quittierungen — Event #{{ activeEvent?.id }}</h2>
        <p class="text-xs text-gray-400 mb-4">{{ activeEvent?.source }} / {{ activeEvent?.keyword }}</p>
        <div v-if="acks.length === 0" class="text-sm text-gray-400 py-6 text-center">Noch keine Quittierungen.</div>
        <table v-else class="w-full text-sm">
          <thead>
            <tr class="text-left text-xs text-gray-400 uppercase">
              <th class="py-1">Empfänger-ID</th>
              <th class="py-1">Antwort</th>
              <th class="py-1">Kanal</th>
              <th class="py-1">Zeit</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="a in acks" :key="a.recipient_id" class="border-t border-gray-100 dark:border-slate-700">
              <td class="py-1.5">{{ a.recipient_id }}</td>
              <td class="py-1.5"><span class="badge" :class="ackClass(a.response)">{{ a.response }}</span></td>
              <td class="py-1.5">{{ a.channel }}</td>
              <td class="py-1.5 text-xs font-mono">{{ fmtTs(a.acknowledged_at) }}</td>
            </tr>
          </tbody>
        </table>
        <div class="flex justify-end pt-4">
          <button class="btn-ghost" @click="acksModalOpen = false">Schließen</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { apiClient } from '../api/index.js'

const events = ref([])
const loading = ref(true)
const acksModalOpen = ref(false)
const activeEvent = ref(null)
const acks = ref([])

async function load() {
  loading.value = true
  const res = await apiClient.get('/api/events/log', { params: { limit: 200 } })
  events.value = res.data.events
  loading.value = false
}

async function openAcks(e) {
  activeEvent.value = e
  const res = await apiClient.get('/api/acknowledgements', { params: { event_id: e.id } })
  acks.value = res.data.acknowledgements
  acksModalOpen.value = true
}

function fmtTs(ts) {
  if (!ts) return '—'
  return new Date(ts).toLocaleString('de-DE')
}

function statusClass(status) {
  return {
    DISPATCHED: 'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300',
    NO_MATCH:   'bg-gray-100 text-gray-500 dark:bg-slate-700 dark:text-slate-400',
    ERROR:      'bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-300',
  }[status] ?? 'bg-gray-100 text-gray-500'
}

function ackClass(response) {
  return {
    COMING:     'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300',
    NOT_COMING: 'bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-300',
    LATER:      'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/40 dark:text-yellow-300',
  }[response] ?? 'bg-gray-100 text-gray-500'
}

onMounted(load)
</script>
