<template>
  <div>
    <div class="flex items-center justify-between mb-4">
      <h1 class="text-xl font-bold">Gruppen & Empfänger</h1>
      <div class="flex gap-2">
        <button v-if="tab === 'groups'" class="btn-primary" @click="openCreateGroup">+ Gruppe</button>
        <button v-else class="btn-primary" @click="openCreateRecipient">+ Empfänger</button>
      </div>
    </div>

    <div class="flex gap-1 mb-4">
      <button class="btn-ghost" :class="tab === 'groups' ? '!bg-ks-600 !text-white' : ''" @click="tab = 'groups'">Gruppen</button>
      <button class="btn-ghost" :class="tab === 'recipients' ? '!bg-ks-600 !text-white' : ''" @click="tab = 'recipients'">Empfänger</button>
    </div>

    <!-- Groups tab -->
    <div v-if="tab === 'groups'" class="card overflow-hidden p-0">
      <table class="w-full text-sm">
        <thead>
          <tr class="bg-gray-50 dark:bg-slate-700/60 border-b border-gray-200 dark:border-slate-700 text-left">
            <th class="th">Name</th>
            <th class="th">Beschreibung</th>
            <th class="th text-center">Mitglieder</th>
            <th class="th text-center">Self-Service</th>
            <th class="th w-40"></th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-100 dark:divide-slate-700/60">
          <tr v-if="groups.length === 0"><td colspan="4" class="py-10 text-center text-gray-400">Keine Gruppen</td></tr>
          <tr v-for="g in groups" :key="g.id" class="hover:bg-gray-50 dark:hover:bg-slate-700/30">
            <td class="td font-medium">{{ g.name }}</td>
            <td class="td text-gray-500">{{ g.description }}</td>
            <td class="td text-center">{{ g.member_count }}</td>
            <td class="td text-center">
              <span class="badge" :class="g.self_service ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300' : 'bg-gray-100 text-gray-400 dark:bg-slate-700 dark:text-slate-500'">
                {{ g.self_service ? 'Self-Service' : '–' }}
              </span>
            </td>
            <td class="td text-right">
              <button class="text-ks-600 dark:text-ks-300 hover:underline text-xs mr-2" @click="openMembers(g)">Mitglieder</button>
              <button class="text-ks-600 dark:text-ks-300 hover:underline text-xs mr-2" @click="openEditGroup(g)">Edit</button>
              <button class="text-red-600 hover:underline text-xs" @click="removeGroup(g)">Del</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Recipients tab -->
    <div v-else class="card overflow-hidden p-0">
      <table class="w-full text-sm">
        <thead>
          <tr class="bg-gray-50 dark:bg-slate-700/60 border-b border-gray-200 dark:border-slate-700 text-left">
            <th class="th">Name</th>
            <th class="th">Voice</th>
            <th class="th">Mobile</th>
            <th class="th">E-Mail</th>
            <th class="th text-center">Aktiv</th>
            <th class="th w-28"></th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-100 dark:divide-slate-700/60">
          <tr v-if="recipients.length === 0"><td colspan="6" class="py-10 text-center text-gray-400">Keine Empfänger</td></tr>
          <tr v-for="r in recipients" :key="r.id" class="hover:bg-gray-50 dark:hover:bg-slate-700/30">
            <td class="td font-medium">{{ r.last_name }}, {{ r.first_name }}</td>
            <td class="td font-mono text-xs">{{ r.voice }}</td>
            <td class="td font-mono text-xs">{{ r.mobile }}</td>
            <td class="td text-xs">{{ r.email }}</td>
            <td class="td text-center">
              <span class="badge" :class="r.active ? 'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300' : 'bg-gray-100 text-gray-500 dark:bg-slate-700 dark:text-slate-400'">{{ r.active ? 'aktiv' : 'inaktiv' }}</span>
            </td>
            <td class="td text-right">
              <button class="text-ks-600 dark:text-ks-300 hover:underline text-xs mr-2" @click="openEditRecipient(r)">Edit</button>
              <button class="text-red-600 hover:underline text-xs" @click="removeRecipient(r)">Del</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Group create/edit modal -->
    <div v-if="groupModalOpen" class="fixed inset-0 bg-gray-900/40 flex items-center justify-center z-50 p-4" @click.self="groupModalOpen = false">
      <div class="bg-white dark:bg-slate-800 rounded-xl shadow-2xl p-5 w-full max-w-md">
        <h2 class="text-lg font-bold mb-4">{{ groupForm.id ? 'Gruppe bearbeiten' : 'Neue Gruppe' }}</h2>
        <form @submit.prevent="saveGroup" class="space-y-3">
          <div><label class="label">Name</label><input v-model="groupForm.name" class="input" required /></div>
          <div><label class="label">Beschreibung</label><input v-model="groupForm.description" class="input" /></div>
          <div class="flex items-center gap-2">
            <input type="checkbox" id="ss" v-model="groupForm.self_service" />
            <label for="ss" class="text-sm">Self-Service (Mitglieder können sich selbst aktiv/inaktiv setzen)</label>
          </div>
          <div v-if="error" class="text-red-600 text-sm bg-red-50 border border-red-200 rounded-lg px-3 py-2">{{ error }}</div>
          <div class="flex justify-end gap-2 pt-2">
            <button type="button" class="btn-ghost" @click="groupModalOpen = false">Abbrechen</button>
            <button type="submit" class="btn-primary">Speichern</button>
          </div>
        </form>
      </div>
    </div>

    <!-- Members modal -->
    <div v-if="membersModalOpen" class="fixed inset-0 bg-gray-900/40 flex items-center justify-center z-50 p-4" @click.self="membersModalOpen = false">
      <div class="bg-white dark:bg-slate-800 rounded-xl shadow-2xl p-5 w-full max-w-md max-h-[80vh] overflow-y-auto">
        <h2 class="text-lg font-bold mb-1">Mitglieder: {{ activeGroup?.name }}</h2>
        <p class="text-xs text-gray-400 mb-4">Empfänger an-/abhaken</p>
        <div class="space-y-1.5">
          <label v-for="r in recipients" :key="r.id" class="flex items-center gap-2 text-sm py-1">
            <input type="checkbox" :checked="memberIds.has(r.id)" @change="toggleMember(r.id, $event.target.checked)" />
            {{ r.last_name }}, {{ r.first_name }}
          </label>
        </div>
        <div class="flex justify-end pt-4">
          <button class="btn-ghost" @click="membersModalOpen = false">Schließen</button>
        </div>
      </div>
    </div>

    <!-- Recipient create/edit modal -->
    <div v-if="recipientModalOpen" class="fixed inset-0 bg-gray-900/40 flex items-center justify-center z-50 p-4" @click.self="recipientModalOpen = false">
      <div class="bg-white dark:bg-slate-800 rounded-xl shadow-2xl p-5 w-full max-w-md">
        <h2 class="text-lg font-bold mb-4">{{ recipientForm.id ? 'Empfänger bearbeiten' : 'Neuer Empfänger' }}</h2>
        <form @submit.prevent="saveRecipient" class="space-y-3">
          <div class="grid grid-cols-2 gap-3">
            <div><label class="label">Vorname</label><input v-model="recipientForm.first_name" class="input" required /></div>
            <div><label class="label">Nachname</label><input v-model="recipientForm.last_name" class="input" required /></div>
          </div>
          <div><label class="label">Voice (Rufnummer)</label><input v-model="recipientForm.voice" class="input" /></div>
          <div><label class="label">Mobile (SMS)</label><input v-model="recipientForm.mobile" class="input" /></div>
          <div><label class="label">E-Mail</label><input v-model="recipientForm.email" type="email" class="input" /></div>
          <div class="flex items-center gap-2"><input v-model="recipientForm.active" type="checkbox" id="ract" /><label for="ract" class="text-sm">Aktiv</label></div>
          <div v-if="error" class="text-red-600 text-sm bg-red-50 border border-red-200 rounded-lg px-3 py-2">{{ error }}</div>
          <div class="flex justify-end gap-2 pt-2">
            <button type="button" class="btn-ghost" @click="recipientModalOpen = false">Abbrechen</button>
            <button type="submit" class="btn-primary">Speichern</button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { apiClient } from '../api/index.js'

const tab = ref('groups')
const groups = ref([])
const recipients = ref([])
const error = ref('')

const groupModalOpen = ref(false)
const groupForm = reactive({ id: null, name: '', description: '', self_service: false })

const membersModalOpen = ref(false)
const activeGroup = ref(null)
const memberIds = ref(new Set())

const recipientModalOpen = ref(false)
const recipientForm = reactive({ id: null, first_name: '', last_name: '', voice: '', mobile: '', email: '', active: true })

async function load() {
  const [g, r] = await Promise.all([apiClient.get('/api/groups'), apiClient.get('/api/recipients')])
  groups.value = g.data.groups
  recipients.value = r.data.recipients
}

function openCreateGroup() {
  Object.assign(groupForm, { id: null, name: '', description: '', self_service: false })
  error.value = ''
  groupModalOpen.value = true
}
function openEditGroup(g) {
  Object.assign(groupForm, { ...g })
  error.value = ''
  groupModalOpen.value = true
}
async function saveGroup() {
  error.value = ''
  try {
    if (groupForm.id) {
      await apiClient.put(`/api/groups/${groupForm.id}`, { name: groupForm.name, description: groupForm.description, self_service: groupForm.self_service })
    } else {
      await apiClient.post('/api/groups', { name: groupForm.name, description: groupForm.description })
    }
    groupModalOpen.value = false
    await load()
  } catch (e) {
    error.value = typeof e.response?.data === 'string' ? e.response.data : 'Speichern fehlgeschlagen'
  }
}
async function removeGroup(g) {
  if (!confirm(`Gruppe "${g.name}" löschen?`)) return
  await apiClient.delete(`/api/groups/${g.id}`)
  await load()
}

async function openMembers(g) {
  activeGroup.value = g
  const res = await apiClient.get(`/api/groups/${g.id}/members`)
  memberIds.value = new Set(res.data.members.map((m) => m.recipient_id))
  membersModalOpen.value = true
}
async function toggleMember(recipientId, checked) {
  if (checked) {
    await apiClient.post(`/api/groups/${activeGroup.value.id}/members`, { recipient_id: recipientId })
    memberIds.value.add(recipientId)
  } else {
    await apiClient.delete(`/api/groups/${activeGroup.value.id}/members/${recipientId}`)
    memberIds.value.delete(recipientId)
  }
  await load()
}

function openCreateRecipient() {
  Object.assign(recipientForm, { id: null, first_name: '', last_name: '', voice: '', mobile: '', email: '', active: true })
  error.value = ''
  recipientModalOpen.value = true
}
function openEditRecipient(r) {
  Object.assign(recipientForm, { ...r })
  error.value = ''
  recipientModalOpen.value = true
}
async function saveRecipient() {
  error.value = ''
  try {
    if (recipientForm.id) {
      await apiClient.put(`/api/recipients/${recipientForm.id}`, recipientForm)
    } else {
      await apiClient.post('/api/recipients', recipientForm)
    }
    recipientModalOpen.value = false
    await load()
  } catch (e) {
    error.value = typeof e.response?.data === 'string' ? e.response.data : 'Speichern fehlgeschlagen'
  }
}
async function removeRecipient(r) {
  if (!confirm(`Empfänger "${r.first_name} ${r.last_name}" löschen?`)) return
  await apiClient.delete(`/api/recipients/${r.id}`)
  await load()
}

onMounted(load)
</script>
