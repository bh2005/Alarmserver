<template>
  <Teleport to="body">
    <Transition name="help-slide">
      <div v-if="open" class="fixed top-0 right-0 bottom-0 z-[65] flex">
        <div class="fixed inset-0 bg-gray-900/40" @click="$emit('close')" />

        <div class="relative ml-auto w-full max-w-sm bg-white dark:bg-slate-800 shadow-2xl ring-1 ring-gray-200 dark:ring-slate-700 flex flex-col h-full">

          <!-- Header -->
          <div class="flex items-center justify-between px-4 py-3.5 border-b border-gray-200 dark:border-slate-700 bg-gray-50 dark:bg-slate-900/50 flex-shrink-0">
            <div class="flex items-center gap-2">
              <span class="w-6 h-6 rounded-full bg-ks-600 flex items-center justify-center text-white text-xs font-bold flex-shrink-0">?</span>
              <div>
                <div class="text-xs text-gray-400 dark:text-slate-500 uppercase tracking-wider">Hilfe</div>
                <div class="text-sm font-semibold text-gray-800 dark:text-gray-100">{{ pageTitle }}</div>
              </div>
            </div>
            <button
              @click="$emit('close')"
              class="text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 p-1.5 rounded hover:bg-gray-100 dark:hover:bg-slate-700 transition-colors"
              title="Hilfe schließen (F1)"
            >
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
              </svg>
            </button>
          </div>

          <!-- Content -->
          <div class="flex-1 overflow-y-auto p-4">
            <div class="help-content" v-html="rendered" />
          </div>

          <!-- Footer -->
          <div class="px-4 py-2.5 border-t border-gray-100 dark:border-slate-700 flex-shrink-0 flex items-center justify-between">
            <span class="text-xs text-gray-400 dark:text-slate-500">Alarm-Core Manager — F1 zum Schließen</span>
            <router-link
              to="/handbuch"
              @click="$emit('close')"
              class="text-xs text-ks-600 dark:text-ks-400 hover:underline flex items-center gap-1"
            >
              <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253"/>
              </svg>
              Vollständiges Handbuch
            </router-link>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { getHelp } from '../content/helpTexts.js'

defineProps({ open: Boolean })
defineEmits(['close'])

const route = useRoute()
const rendered = ref('')

const pageTitles = {
  '/dashboard':    'Dashboard',
  '/':             'Regeln',
  '/groups':       'Gruppen & Empfänger',
  '/events':       'Event-Log',
  '/announcements':'Ansagen',
  '/users':        'Benutzerverwaltung',
  '/channels':     'Kanäle',
  '/apikeys':      'API Keys',
  '/maintenance':  'Wartungsfenster',
  '/heartbeats':   'Heartbeat-Überwachung',
  '/templates':    'Nachrichten-Templates',
  '/stats':        'Statistiken',
  '/me/groups':    'Mein Einsatz',
  '/handbuch':     'Handbuch',
}

const pageTitle = computed(() => pageTitles[route.path] ?? route.path)

watch(
  () => route.path,
  async (path) => {
    const { marked } = await import('marked')
    rendered.value = await marked.parse(getHelp(path))
  },
  { immediate: true },
)
</script>

<style>
.help-content h2 {
  font-size: 1rem;
  font-weight: 700;
  color: #173b7a;
  margin-top: 1.25rem;
  margin-bottom: 0.5rem;
  padding-bottom: 0.25rem;
  border-bottom: 1px solid #e5e7eb;
}
.help-content h3 { font-size: 0.875rem; font-weight: 600; color: #374151; margin-top: 1rem; margin-bottom: 0.375rem; }
.help-content p  { font-size: 0.8125rem; color: #4b5563; margin-bottom: 0.625rem; line-height: 1.6; }
.help-content ul { margin-left: 1rem; margin-bottom: 0.75rem; list-style-type: disc; }
.help-content li { font-size: 0.8125rem; color: #4b5563; margin-bottom: 0.25rem; line-height: 1.5; }
.help-content strong { font-weight: 600; color: #1f2937; }
.help-content code { background: #f3f4f6; color: #173b7a; padding: 0.1em 0.35em; border-radius: 0.25rem; font-size: 0.75rem; font-family: ui-monospace, Menlo, monospace; }
.help-content pre { background: #f3f4f6; border-radius: 0.5rem; padding: 0.75rem; overflow-x: auto; margin-bottom: 0.75rem; }
.help-content pre code { background: transparent; padding: 0; color: #1f2937; font-size: 0.75rem; }
.help-content table { width: 100%; border-collapse: collapse; margin-bottom: 0.75rem; font-size: 0.8125rem; }
.help-content th { background: #f9fafb; padding: 0.375rem 0.5rem; text-align: left; font-weight: 600; color: #374151; border-bottom: 2px solid #e5e7eb; }
.help-content td { padding: 0.375rem 0.5rem; color: #4b5563; border-bottom: 1px solid #f3f4f6; }
.help-content a { color: #173b7a; text-decoration: underline; }

.dark .help-content h2 { color: #93c5fd; border-bottom-color: #334155; }
.dark .help-content h3 { color: #cbd5e1; }
.dark .help-content p  { color: #94a3b8; }
.dark .help-content li { color: #94a3b8; }
.dark .help-content strong { color: #e2e8f0; }
.dark .help-content code { background: #1e293b; color: #93c5fd; }
.dark .help-content pre  { background: #0f172a; }
.dark .help-content pre code { color: #e2e8f0; }
.dark .help-content th { background: #1e293b; color: #cbd5e1; border-bottom-color: #334155; }
.dark .help-content td { color: #94a3b8; border-bottom-color: #1e293b; }
.dark .help-content a { color: #93c5fd; }
</style>

<style scoped>
.help-slide-enter-from { transform: translateX(100%); opacity: 0; }
.help-slide-enter-active { transition: transform 0.25s ease-out, opacity 0.2s ease-out; }
.help-slide-leave-to { transform: translateX(100%); opacity: 0; }
.help-slide-leave-active { transition: transform 0.2s ease-in, opacity 0.15s ease-in; }
</style>
