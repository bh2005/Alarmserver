<template>
  <div>
    <div class="flex items-center justify-between mb-4">
      <h1 class="text-xl font-bold">Regeln <span class="ml-2 text-sm font-normal text-gray-400 dark:text-slate-500">{{ rules.length }}</span></h1>
      <button class="btn-primary" @click="openCreate">+ Regel</button>
    </div>

    <div class="card overflow-hidden p-0">
      <table class="w-full text-sm">
        <thead>
          <tr class="bg-gray-50 dark:bg-slate-700/60 border-b border-gray-200 dark:border-slate-700 text-left">
            <th class="th">Quelle</th>
            <th class="th">Keyword</th>
            <th class="th">Zeitfenster</th>
            <th class="th">Zielgruppe</th>
            <th class="th">Kanäle</th>
            <th class="th text-center">Prio</th>
            <th class="th text-center">Aktiv</th>
            <th class="th text-center">Ack</th>
            <th class="th">Eskalation</th>
            <th class="th w-28"></th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-100 dark:divide-slate-700/60">
          <tr v-if="loading"><td colspan="9" class="py-10 text-center text-gray-400">Laden…</td></tr>
          <tr v-else-if="rules.length === 0"><td colspan="9" class="py-10 text-center text-gray-400">Keine Regeln</td></tr>
          <tr v-for="r in rules" :key="r.id" class="hover:bg-gray-50 dark:hover:bg-slate-700/30">
            <td class="td font-mono text-xs">{{ r.source }}</td>
            <td class="td font-mono text-xs">{{ r.keyword }}</td>
            <td class="td text-xs text-gray-500">
              <template v-if="r.start_hour !== null || r.end_hour !== null">{{ r.start_hour ?? '–' }}–{{ r.end_hour ?? '–' }} Uhr</template>
              <template v-else>immer</template>
              <span v-if="r.weekdays_only" class="ml-1">(Mo–Fr)</span>
            </td>
            <td class="td">
              <span v-if="r.schedule?.length" class="text-blue-600 dark:text-blue-400 text-xs">Zeitplan ({{ r.schedule.length }})</span>
              <span v-else>{{ r.target_group }}</span>
            </td>
            <td class="td"><span v-for="c in r.channels" :key="c" class="badge bg-ks-50 text-ks-700 dark:bg-slate-700 dark:text-ks-300 mr-1">{{ c }}</span></td>
            <td class="td text-center">{{ r.priority }}</td>
            <td class="td text-center">
              <span class="badge" :class="r.active ? 'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300' : 'bg-gray-100 text-gray-500 dark:bg-slate-700 dark:text-slate-400'">
                {{ r.active ? 'aktiv' : 'inaktiv' }}
              </span>
            </td>
            <td class="td text-center">{{ r.require_ack ? '✓' : '–' }}</td>
            <td class="td text-xs text-gray-500">
              <template v-if="r.escalation_channels?.length">
                <span v-for="c in r.escalation_channels" :key="c" class="badge bg-orange-50 text-orange-700 dark:bg-orange-900/30 dark:text-orange-300 mr-1">{{ c }}</span>
                <span class="ml-1">({{ r.escalation_delay_seconds }}s)</span>
              </template>
              <template v-else>–</template>
            </td>
            <td class="td text-right">
              <button class="text-ks-600 dark:text-ks-300 hover:underline text-xs mr-2" @click="openEdit(r)">Edit</button>
              <button class="text-red-600 hover:underline text-xs" @click="remove(r)">Del</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Modal -->
    <div v-if="modalOpen" class="fixed inset-0 bg-gray-900/40 flex items-center justify-center z-50 p-4" @click.self="modalOpen = false">
      <div class="bg-white dark:bg-slate-800 rounded-xl shadow-2xl p-5 w-full max-w-lg max-h-[90vh] overflow-y-auto">
        <h2 class="text-lg font-bold mb-4">{{ form.id ? 'Regel bearbeiten' : 'Neue Regel' }}</h2>
        <form @submit.prevent="save" class="space-y-3">
          <div class="grid grid-cols-2 gap-3">
            <div><label class="label">Quelle</label><input v-model="form.source" class="input" placeholder="WEBHOOK" required /></div>
            <div><label class="label">Keyword</label><input v-model="form.keyword" class="input" placeholder="F2" required /></div>
          </div>
          <div class="grid grid-cols-3 gap-3">
            <div><label class="label">Start-Stunde</label><input v-model.number="startHour" type="number" min="0" max="23" class="input" placeholder="leer = egal" /></div>
            <div><label class="label">End-Stunde</label><input v-model.number="endHour" type="number" min="0" max="23" class="input" placeholder="leer = egal" /></div>
            <div class="flex items-center gap-2 pt-6"><input v-model="form.weekdays_only" type="checkbox" id="wd" /><label for="wd" class="text-sm">Nur Mo–Fr</label></div>
          </div>
          <div>
            <label class="label">Zielgruppe</label>
            <select v-model.number="form.target_group_id" class="input" required>
              <option v-for="g in groups" :key="g.id" :value="g.id">{{ g.name }}</option>
            </select>
          </div>
          <div>
            <label class="label">Kanäle</label>
            <div class="flex gap-3 flex-wrap">
              <label v-for="c in allChannels" :key="c" class="flex items-center gap-1.5 text-sm">
                <input type="checkbox" :value="c" v-model="form.channels" />{{ c }}
              </label>
            </div>
          </div>

          <!-- Zeitgesteuerter Gruppenplan -->
          <div class="border-t border-gray-100 dark:border-slate-700 pt-3">
            <div class="flex items-center gap-2 mb-2">
              <input type="checkbox" id="sched-toggle" v-model="scheduleEnabled" />
              <label for="sched-toggle" class="text-sm font-medium">Zeitgesteuerter Gruppenplan</label>
            </div>
            <div v-if="scheduleEnabled" class="space-y-2 pl-1">
              <p class="text-xs text-gray-400">Zielgruppe aus Standardregel wird als Fallback verwendet wenn kein Eintrag passt.</p>
              <div v-for="(entry, idx) in scheduleEntries" :key="idx" class="border border-gray-200 dark:border-slate-600 rounded-lg p-3 space-y-2">
                <div class="flex flex-wrap gap-1">
                  <label v-for="(d, di) in dayNames" :key="di" class="flex items-center gap-1 text-xs border border-gray-200 dark:border-slate-600 rounded px-2 py-0.5 cursor-pointer"
                    :class="entry.days.includes(di) ? 'bg-ks-600 text-white border-ks-600' : 'bg-white dark:bg-slate-800'">
                    <input type="checkbox" :value="di" v-model="entry.days" class="sr-only" />{{ d }}
                  </label>
                </div>
                <div class="flex items-center gap-2 flex-wrap">
                  <div><label class="label text-xs">Von</label><input v-model="entry.from" type="time" class="input py-1 text-xs w-28" /></div>
                  <div><label class="label text-xs">Bis</label><input v-model="entry.to" type="time" class="input py-1 text-xs w-28" /></div>
                  <div class="flex-1 min-w-36">
                    <label class="label text-xs">Gruppe</label>
                    <select v-model="entry.target_group" class="input py-1 text-xs">
                      <option v-for="g in groups" :key="g.id" :value="g.name">{{ g.name }}</option>
                    </select>
                  </div>
                  <button type="button" class="mt-4 text-red-500 hover:text-red-700 text-xs" @click="scheduleEntries.splice(idx, 1)">Entfernen</button>
                </div>
              </div>
              <button type="button" class="btn-ghost text-xs" @click="addScheduleEntry">+ Zeitfenster</button>
            </div>
          </div>

          <div class="border-t border-gray-100 dark:border-slate-700 pt-3">
            <div class="flex items-center gap-2 mb-2">
              <input type="checkbox" id="esc-toggle" v-model="escalationEnabled" />
              <label for="esc-toggle" class="text-sm font-medium">Eskalation bei Nicht-Antwort</label>
            </div>
            <div v-if="escalationEnabled" class="space-y-2 pl-4">
              <div>
                <label class="label">Eskalationskanäle</label>
                <div class="flex gap-3 flex-wrap">
                  <label v-for="c in allChannels" :key="'esc-'+c" class="flex items-center gap-1.5 text-sm">
                    <input type="checkbox" :value="c" v-model="form.escalation_channels" />{{ c }}
                  </label>
                </div>
              </div>
              <div>
                <label class="label">Wartezeit vor Eskalation (Sekunden)</label>
                <input v-model.number="form.escalation_delay_seconds" type="number" min="0" class="input w-32" />
              </div>
            </div>
          </div>

          <!-- Wiederhol-Benachrichtigung -->
          <div class="border-t border-gray-100 dark:border-slate-700 pt-3">
            <div class="flex items-center gap-2 mb-2">
              <input type="checkbox" id="rep-toggle" v-model="repeatEnabled" />
              <label for="rep-toggle" class="text-sm font-medium">Wiederhol-Benachrichtigung (erfordert Ack)</label>
            </div>
            <div v-if="repeatEnabled" class="space-y-2 pl-4">
              <p class="text-xs text-gray-400">Wird erneut alarmiert wenn nach dem Interval kein Ack eingegangen ist.</p>
              <div class="grid grid-cols-2 gap-3">
                <div>
                  <label class="label">Interval (Sek.)</label>
                  <input v-model.number="form.repeat_interval_sec" type="number" min="30" class="input w-28" placeholder="300" />
                </div>
                <div>
                  <label class="label">Max. Wiederholungen</label>
                  <input v-model.number="form.repeat_max" type="number" min="1" max="20" class="input w-28" placeholder="3" />
                </div>
              </div>
            </div>
          </div>

          <div>
            <label class="label">Eskalationspolitik (optional)</label>
            <select v-model="form.escalation_policy_id" class="input">
              <option :value="null">— keine —</option>
              <option v-for="p in escalationPolicies" :key="p.id" :value="p.id">{{ p.name }}</option>
            </select>
          </div>

          <div class="grid grid-cols-3 gap-3">
            <div><label class="label">Priorität</label><input v-model.number="form.priority" type="number" class="input" /></div>
            <div class="flex items-center gap-2 pt-6"><input v-model="form.active" type="checkbox" id="ac" /><label for="ac" class="text-sm">Aktiv</label></div>
            <div class="flex items-center gap-2 pt-6"><input v-model="form.require_ack" type="checkbox" id="ra" /><label for="ra" class="text-sm">Ack erwarten</label></div>
          </div>
          <div v-if="error" class="text-red-600 text-sm bg-red-50 border border-red-200 rounded-lg px-3 py-2">{{ error }}</div>
          <div class="flex justify-end gap-2 pt-2">
            <button type="button" class="btn-ghost" @click="modalOpen = false">Abbrechen</button>
            <button type="submit" class="btn-primary">Speichern</button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref, watch } from 'vue'
import { apiClient } from '../api/index.js'

const rules = ref([])
const groups = ref([])
const escalationPolicies = ref([])
const loading = ref(true)
const modalOpen = ref(false)
const error = ref('')

const staticChannels = ['voice', 'voice-smseagle', 'sms', 'email', 'teams']
const allChannels = ref([...staticChannels])
const dayNames = ['Mo', 'Di', 'Mi', 'Do', 'Fr', 'Sa', 'So']

const form = reactive({
  id: null, source: '', keyword: '', weekdays_only: false,
  target_group_id: null, channels: [], priority: 10, active: true, require_ack: false,
  escalation_channels: [], escalation_delay_seconds: 120,
  repeat_interval_sec: 300, repeat_max: 3,
  escalation_policy_id: null,
})
const startHour = ref(null)
const endHour = ref(null)
const escalationEnabled = ref(false)
const scheduleEnabled   = ref(false)
const scheduleEntries   = ref([])
const repeatEnabled     = ref(false)

watch(escalationEnabled, (val) => { if (!val) form.escalation_channels = [] })
watch(scheduleEnabled,   (val) => { if (!val) scheduleEntries.value = [] })
watch(repeatEnabled,     (val) => { if (!val) { form.repeat_interval_sec = 0; form.repeat_max = 3 } })

function addScheduleEntry() {
  scheduleEntries.value.push({ days: [0, 1, 2, 3, 4], from: '08:00', to: '17:00', target_group: groups.value[0]?.name ?? '' })
}

async function load() {
  loading.value = true
  const [r, g, ep, wh] = await Promise.all([
    apiClient.get('/api/rules'),
    apiClient.get('/api/groups'),
    apiClient.get('/api/escalation-policies'),
    apiClient.get('/api/webhook-targets'),
  ])
  rules.value = r.data.rules
  groups.value = g.data.groups
  escalationPolicies.value = ep.data ?? []
  const webhookNames = (wh.data ?? []).filter(t => t.active).map(t => t.name)
  allChannels.value = [...staticChannels, ...webhookNames]
  loading.value = false
}

function openCreate() {
  Object.assign(form, { id: null, source: '', keyword: '', weekdays_only: false, target_group_id: groups.value[0]?.id ?? null, channels: [], priority: 10, active: true, require_ack: false, escalation_channels: [], escalation_delay_seconds: 120, repeat_interval_sec: 300, repeat_max: 3, escalation_policy_id: null })
  startHour.value = null
  endHour.value = null
  escalationEnabled.value = false
  scheduleEnabled.value = false
  scheduleEntries.value = []
  repeatEnabled.value = false
  error.value = ''
  modalOpen.value = true
}

function openEdit(r) {
  Object.assign(form, { ...r })
  form.escalation_channels   = r.escalation_channels ?? []
  form.escalation_delay_seconds = r.escalation_delay_seconds ?? 120
  form.repeat_interval_sec   = r.repeat_interval_sec ?? 0
  form.repeat_max            = r.repeat_max ?? 3
  form.escalation_policy_id  = r.escalation_policy_id ?? null
  startHour.value            = r.start_hour
  endHour.value              = r.end_hour
  escalationEnabled.value    = form.escalation_channels.length > 0
  scheduleEntries.value      = r.schedule ? r.schedule.map(s => ({ ...s, days: [...s.days] })) : []
  scheduleEnabled.value      = scheduleEntries.value.length > 0
  repeatEnabled.value        = form.repeat_interval_sec > 0
  error.value                = ''
  modalOpen.value            = true
}

async function save() {
  error.value = ''
  const payload = {
    ...form,
    start_hour: startHour.value,
    end_hour: endHour.value,
    schedule: scheduleEnabled.value ? scheduleEntries.value : [],
    repeat_interval_sec: repeatEnabled.value ? (form.repeat_interval_sec || 300) : 0,
    repeat_max: repeatEnabled.value ? (form.repeat_max || 3) : 0,
  }
  try {
    if (form.id) {
      await apiClient.put(`/api/rules/${form.id}`, payload)
    } else {
      await apiClient.post('/api/rules', payload)
    }
    modalOpen.value = false
    await load()
  } catch (e) {
    error.value = typeof e.response?.data === 'string' ? e.response.data : 'Speichern fehlgeschlagen'
  }
}

async function remove(r) {
  if (!confirm(`Regel "${r.source}/${r.keyword}" löschen?`)) return
  await apiClient.delete(`/api/rules/${r.id}`)
  await load()
}

onMounted(load)
</script>
