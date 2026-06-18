<template>
  <div class="p-6 max-w-3xl space-y-4">
    <h1 class="text-xl font-bold text-slate-900 dark:text-white">Nachrichten-Templates</h1>

    <p class="text-sm text-slate-500 dark:text-slate-400">
      Templates ermöglichen kanalspezifische Nachrichtenformatierung.
      Verfügbare Platzhalter:
      <code class="bg-slate-100 dark:bg-slate-700 px-1 rounded text-xs">{text}</code>
      <code class="bg-slate-100 dark:bg-slate-700 px-1 rounded text-xs">{source}</code>
      <code class="bg-slate-100 dark:bg-slate-700 px-1 rounded text-xs">{keyword}</code>
      <code class="bg-slate-100 dark:bg-slate-700 px-1 rounded text-xs">{time}</code>
      <code class="bg-slate-100 dark:bg-slate-700 px-1 rounded text-xs">{group}</code>.
      Kein Template = Rohtext der Ereignismeldung.
    </p>

    <!-- Existing templates -->
    <div class="space-y-2">
      <div v-for="t in templates" :key="t.channel"
        class="rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 p-4">
        <div class="flex items-start justify-between gap-4">
          <div class="flex-1">
            <div class="font-medium text-slate-800 dark:text-slate-200 mb-1">
              Kanal: <span class="font-mono text-ks-600 dark:text-ks-400">{{ t.channel }}</span>
            </div>
            <div v-if="editing !== t.channel" class="text-sm text-slate-600 dark:text-slate-300 font-mono bg-slate-50 dark:bg-slate-700 px-3 py-2 rounded">
              {{ t.template }}
            </div>
            <textarea v-else v-model="editValue"
              rows="2"
              class="w-full mt-1 font-mono text-sm rounded border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-700 text-slate-900 dark:text-white px-3 py-2 focus:outline-none focus:ring-2 focus:ring-ks-500">
            </textarea>
            <div class="text-[10px] text-slate-400 mt-1">Aktualisiert: {{ fmtDt(t.updated_at) }}</div>
          </div>
          <div class="flex gap-2 flex-shrink-0">
            <template v-if="editing !== t.channel">
              <button @click="startEdit(t)" class="text-xs text-ks-600 dark:text-ks-400 hover:underline">Bearbeiten</button>
              <button @click="del(t.channel)" class="text-xs text-red-500 hover:underline">Löschen</button>
            </template>
            <template v-else>
              <button @click="saveEdit(t.channel)" :disabled="saving" class="text-xs text-green-600 dark:text-green-400 hover:underline">Speichern</button>
              <button @click="editing = null" class="text-xs text-slate-500 hover:underline">Abbrechen</button>
            </template>
          </div>
        </div>
      </div>

      <div v-if="!templates.length && !showNew" class="text-center text-slate-400 italic py-6 border border-dashed rounded-lg border-slate-200 dark:border-slate-700">
        Noch keine Templates — Alarme werden ohne Formatierung verschickt.
      </div>
    </div>

    <!-- New template form -->
    <div v-if="showNew" class="rounded-lg border border-ks-300 dark:border-ks-700 bg-ks-50 dark:bg-ks-900/20 p-4 space-y-3">
      <h3 class="font-medium text-slate-800 dark:text-slate-200">Neues Template</h3>
      <div class="grid grid-cols-3 gap-3">
        <label class="block">
          <span class="text-xs font-medium text-slate-600 dark:text-slate-400">Kanal *</span>
          <input v-model="newCh.channel" class="mt-1 w-full input" placeholder="sms" />
        </label>
        <label class="col-span-2 block">
          <span class="text-xs font-medium text-slate-600 dark:text-slate-400">Template *</span>
          <input v-model="newCh.template" class="mt-1 w-full input"
            placeholder="ALARM [{keyword}] von {source} um {time}: {text}" />
        </label>
      </div>
      <p v-if="newErr" class="text-sm text-red-500">{{ newErr }}</p>
      <div class="flex gap-2">
        <button @click="saveNew" :disabled="saving"
          class="px-3 py-1.5 text-sm bg-ks-600 hover:bg-ks-700 text-white rounded transition-colors disabled:opacity-50">
          {{ saving ? '...' : 'Speichern' }}
        </button>
        <button @click="showNew = false; newErr = ''" class="text-sm text-slate-500 hover:underline px-2">Abbrechen</button>
      </div>
    </div>

    <div v-if="!showNew">
      <button @click="showNew = true"
        class="px-3 py-1.5 text-sm bg-slate-100 dark:bg-slate-700 hover:bg-slate-200 dark:hover:bg-slate-600 text-slate-700 dark:text-slate-200 rounded transition-colors">
        + Template hinzufügen
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'

const templates = ref([])
const editing   = ref(null)
const editValue = ref('')
const saving    = ref(false)
const showNew   = ref(false)
const newCh     = ref({ channel: '', template: '' })
const newErr    = ref('')

async function load() {
  const r = await fetch('/api/templates')
  if (r.ok) templates.value = (await r.json()).templates || []
}

function fmtDt(s) {
  return new Date(s).toLocaleString('de-DE', { day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit' })
}

function startEdit(t) {
  editing.value = t.channel
  editValue.value = t.template
}

async function saveEdit(channel) {
  if (!editValue.value.trim()) return
  saving.value = true
  try {
    const r = await fetch(`/api/templates/${encodeURIComponent(channel)}`, {
      method: 'PUT', headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({ template: editValue.value })
    })
    if (r.ok) { editing.value = null; await load() }
  } finally { saving.value = false }
}

async function saveNew() {
  newErr.value = ''
  if (!newCh.value.channel || !newCh.value.template) { newErr.value = 'Kanal und Template sind Pflichtfelder.'; return }
  saving.value = true
  try {
    const r = await fetch(`/api/templates/${encodeURIComponent(newCh.value.channel)}`, {
      method: 'PUT', headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({ template: newCh.value.template })
    })
    if (!r.ok) { newErr.value = await r.text(); return }
    showNew.value = false; newCh.value = { channel: '', template: '' }; await load()
  } finally { saving.value = false }
}

async function del(channel) {
  if (!confirm(`Template für Kanal "${channel}" wirklich löschen?`)) return
  await fetch(`/api/templates/${encodeURIComponent(channel)}`, { method: 'DELETE' })
  await load()
}

onMounted(load)
</script>

<style scoped>
.input {
  @apply block w-full rounded-md border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-700 text-slate-900 dark:text-white px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ks-500;
}
</style>
