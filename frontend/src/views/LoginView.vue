<template>
  <div class="min-h-screen flex items-center justify-center bg-ks-700 dark:bg-slate-950 px-4 transition-colors">
    <div class="w-full max-w-sm">

      <!-- Logo / Title -->
      <div class="text-center mb-8">
        <div class="inline-flex items-center justify-center w-14 h-14 rounded-2xl bg-white/10 dark:bg-white/5 mb-4">
          <svg class="w-8 h-8 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5"
              d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"/>
          </svg>
        </div>
        <h1 class="text-2xl font-bold text-white">Alarm-Core</h1>
        <p class="text-white/50 text-sm mt-1">Anmeldung erforderlich</p>
      </div>

      <!-- Card -->
      <div class="bg-white dark:bg-slate-800 rounded-2xl shadow-2xl dark:shadow-black/40 ring-1 ring-black/5 dark:ring-white/5 p-6">
        <form @submit.prevent="submit" class="space-y-4">
          <div>
            <label class="label">Benutzername</label>
            <input
              v-model="form.username"
              class="input"
              placeholder="admin"
              autocomplete="username"
              required
            />
          </div>
          <div>
            <label class="label">Passwort</label>
            <input
              v-model="form.password"
              type="password"
              class="input"
              autocomplete="current-password"
              required
            />
          </div>

          <div v-if="error"
            class="text-red-600 dark:text-red-400 text-sm bg-red-50 dark:bg-red-950/50 border border-red-200 dark:border-red-800 rounded-lg px-3 py-2">
            {{ error }}
          </div>

          <button
            type="submit"
            class="btn-primary w-full mt-2 py-2.5"
            :disabled="loading"
          >
            <span v-if="loading" class="flex items-center justify-center gap-2">
              <svg class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"/>
              </svg>
              Anmeldung…
            </span>
            <span v-else>Anmelden</span>
          </button>
        </form>
      </div>

      <p class="text-center text-white/30 text-xs mt-6">
        Example AG · Internes Werkzeug
      </p>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth.js'

const auth   = useAuthStore()
const router = useRouter()
const form   = reactive({ username: '', password: '' })
const error  = ref('')
const loading = ref(false)

async function submit() {
  error.value = ''
  loading.value = true
  try {
    await auth.login(form.username, form.password)
    router.push('/dashboard')
  } catch (e) {
    error.value = typeof e.response?.data === 'string' ? e.response.data : 'Anmeldung fehlgeschlagen'
  } finally {
    loading.value = false
  }
}
</script>
