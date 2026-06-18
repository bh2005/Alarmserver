<template>
  <div>
    <div class="flex items-center justify-between mb-4">
      <h1 class="text-xl font-bold">Benutzer <span class="ml-2 text-sm font-normal text-gray-400 dark:text-slate-500">{{ users.length }}</span></h1>
      <button class="btn-primary" @click="modalOpen = true">+ Benutzer</button>
    </div>

    <div class="card overflow-hidden p-0">
      <table class="w-full text-sm">
        <thead>
          <tr class="bg-gray-50 dark:bg-slate-700/60 border-b border-gray-200 dark:border-slate-700 text-left">
            <th class="th">Benutzername</th>
            <th class="th">Rolle</th>
            <th class="th">Teilnehmer</th>
            <th class="th text-center">Aktiv</th>
            <th class="th w-32"></th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-100 dark:divide-slate-700/60">
          <tr v-for="u in users" :key="u.id" class="hover:bg-gray-50 dark:hover:bg-slate-700/30">
            <td class="td font-medium">{{ u.username }}</td>
            <td class="td"><span class="badge bg-ks-50 text-ks-700 dark:bg-slate-700 dark:text-ks-300">{{ u.role }}</span></td>
            <td class="td">
              <select class="input py-0.5 text-xs" :value="u.recipient_id ?? ''" @change="setRecipient(u, $event.target.value)">
                <option value="">– nicht verknüpft –</option>
                <option v-for="r in recipients" :key="r.id" :value="r.id">{{ r.last_name }}, {{ r.first_name }}</option>
              </select>
            </td>
            <td class="td text-center">
              <button class="badge cursor-pointer" :class="u.active ? 'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300' : 'bg-gray-100 text-gray-500 dark:bg-slate-700 dark:text-slate-400'" @click="toggleActive(u)">
                {{ u.active ? 'aktiv' : 'inaktiv' }}
              </button>
            </td>
            <td class="td text-right">
              <button class="text-red-600 hover:underline text-xs" @click="remove(u)">Del</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-if="modalOpen" class="fixed inset-0 bg-gray-900/40 flex items-center justify-center z-50 p-4" @click.self="modalOpen = false">
      <div class="bg-white dark:bg-slate-800 rounded-xl shadow-2xl p-5 w-full max-w-md">
        <h2 class="text-lg font-bold mb-4">Neuer Benutzer</h2>
        <form @submit.prevent="create" class="space-y-3">
          <div><label class="label">Benutzername</label><input v-model="form.username" class="input" required /></div>
          <div><label class="label">Passwort</label><input v-model="form.password" type="password" class="input" required /></div>
          <div>
            <label class="label">Rolle</label>
            <select v-model="form.role" class="input">
              <option value="viewer">viewer</option>
              <option value="admin">admin</option>
            </select>
          </div>
          <div v-if="error" class="text-red-600 text-sm bg-red-50 border border-red-200 rounded-lg px-3 py-2">{{ error }}</div>
          <div class="flex justify-end gap-2 pt-2">
            <button type="button" class="btn-ghost" @click="modalOpen = false">Abbrechen</button>
            <button type="submit" class="btn-primary">Anlegen</button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { apiClient } from '../api/index.js'

const users = ref([])
const recipients = ref([])
const modalOpen = ref(false)
const error = ref('')
const form = reactive({ username: '', password: '', role: 'viewer' })

async function load() {
  const [u, r] = await Promise.all([apiClient.get('/api/users'), apiClient.get('/api/recipients')])
  users.value = u.data.users
  recipients.value = r.data.recipients
}

async function create() {
  error.value = ''
  try {
    await apiClient.post('/api/users/create', form)
    modalOpen.value = false
    Object.assign(form, { username: '', password: '', role: 'viewer' })
    await load()
  } catch (e) {
    error.value = typeof e.response?.data === 'string' ? e.response.data : 'Anlegen fehlgeschlagen'
  }
}

async function toggleActive(u) {
  await apiClient.post(`/api/users/${u.id}/active`, { active: !u.active })
  await load()
}

async function setRecipient(u, val) {
  const recipientId = val === '' ? null : parseInt(val)
  await apiClient.post(`/api/users/${u.id}/recipient`, { recipient_id: recipientId })
  await load()
}

async function remove(u) {
  if (!confirm(`Benutzer "${u.username}" löschen?`)) return
  await apiClient.delete(`/api/users/${u.id}`)
  await load()
}

onMounted(load)
</script>
