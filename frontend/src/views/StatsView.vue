<template>
  <div class="p-6 max-w-5xl space-y-6">
    <div class="flex items-center justify-between">
      <h1 class="text-xl font-bold text-slate-900 dark:text-white">Statistiken</h1>
      <span class="text-xs text-slate-400">Letzte 30 Tage</span>
    </div>

    <div v-if="loading" class="text-center text-slate-400 py-12">Lädt...</div>
    <template v-else>
      <!-- Summary cards -->
      <div class="grid grid-cols-2 sm:grid-cols-4 gap-4">
        <div class="rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 p-4 text-center">
          <div class="text-3xl font-bold text-slate-900 dark:text-white">{{ stats.total_events ?? 0 }}</div>
          <div class="text-xs text-slate-500 mt-1">Alarme gesamt</div>
        </div>
        <div class="rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 p-4 text-center">
          <div class="text-3xl font-bold text-green-600 dark:text-green-400">{{ stats.acked_events ?? 0 }}</div>
          <div class="text-xs text-slate-500 mt-1">Quittiert</div>
        </div>
        <div class="rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 p-4 text-center">
          <div class="text-3xl font-bold text-orange-500">
            {{ stats.total_events ? (stats.total_events - stats.acked_events) : 0 }}
          </div>
          <div class="text-xs text-slate-500 mt-1">Nicht quittiert</div>
        </div>
        <div class="rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 p-4 text-center">
          <div class="text-3xl font-bold text-ks-600 dark:text-ks-400">
            {{ stats.total_events ? Math.round(stats.acked_events / stats.total_events * 100) : 0 }}%
          </div>
          <div class="text-xs text-slate-500 mt-1">Quittierungsrate</div>
        </div>
      </div>

      <!-- Events per day bar chart -->
      <div class="rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 p-4">
        <h2 class="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-3">Ereignisse pro Tag</h2>
        <div v-if="!stats.events_per_day?.length" class="text-slate-400 text-sm text-center py-4">Keine Daten</div>
        <div v-else class="overflow-x-auto">
          <div class="flex items-end gap-1 min-w-max pb-6 relative" style="height: 120px;">
            <div v-for="d in stats.events_per_day" :key="d.date"
              class="flex flex-col items-center group relative"
              style="width: 28px;">
              <div class="absolute bottom-6 left-1/2 -translate-x-1/2 bg-slate-900 dark:bg-white text-white dark:text-slate-900 text-[10px] px-1.5 py-0.5 rounded opacity-0 group-hover:opacity-100 transition-opacity whitespace-nowrap z-10 pointer-events-none">
                {{ d.date }}: {{ d.count }}
              </div>
              <div class="w-full bg-ks-500 dark:bg-ks-400 rounded-t transition-all"
                :style="{ height: maxDay ? Math.max(4, d.count / maxDay * 80) + 'px' : '4px' }">
              </div>
              <div class="text-[8px] text-slate-400 mt-1 transform -rotate-45 origin-left whitespace-nowrap">
                {{ d.date.slice(5) }}
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Two-column tables -->
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <!-- Top keywords -->
        <div class="rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 p-4">
          <h2 class="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-3">Top Stichworte</h2>
          <div v-if="!stats.top_keywords?.length" class="text-slate-400 text-sm">Keine Daten</div>
          <div v-else class="space-y-2">
            <div v-for="k in stats.top_keywords" :key="k.name" class="flex items-center gap-2">
              <div class="text-sm font-mono font-medium text-slate-800 dark:text-slate-200 w-24 truncate" :title="k.name">{{ k.name }}</div>
              <div class="flex-1 bg-slate-100 dark:bg-slate-700 rounded-full h-3 overflow-hidden">
                <div class="h-full bg-ks-500 dark:bg-ks-400 rounded-full transition-all"
                  :style="{ width: stats.top_keywords[0]?.count ? (k.count / stats.top_keywords[0].count * 100) + '%' : '0%' }">
                </div>
              </div>
              <div class="text-xs font-bold text-slate-600 dark:text-slate-400 w-8 text-right">{{ k.count }}</div>
            </div>
          </div>
        </div>

        <!-- Top sources -->
        <div class="rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 p-4">
          <h2 class="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-3">Top Quellen</h2>
          <div v-if="!stats.top_sources?.length" class="text-slate-400 text-sm">Keine Daten</div>
          <div v-else class="space-y-2">
            <div v-for="s in stats.top_sources" :key="s.name" class="flex items-center gap-2">
              <div class="text-sm font-mono font-medium text-slate-800 dark:text-slate-200 w-24 truncate" :title="s.name">{{ s.name }}</div>
              <div class="flex-1 bg-slate-100 dark:bg-slate-700 rounded-full h-3 overflow-hidden">
                <div class="h-full bg-blue-500 dark:bg-blue-400 rounded-full transition-all"
                  :style="{ width: stats.top_sources[0]?.count ? (s.count / stats.top_sources[0].count * 100) + '%' : '0%' }">
                </div>
              </div>
              <div class="text-xs font-bold text-slate-600 dark:text-slate-400 w-8 text-right">{{ s.count }}</div>
            </div>
          </div>
        </div>

        <!-- Channel usage -->
        <div class="rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 p-4 sm:col-span-2">
          <h2 class="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-3">Kanalnutzung</h2>
          <div v-if="!stats.channel_usage?.length" class="text-slate-400 text-sm">Keine Daten</div>
          <div v-else class="flex flex-wrap gap-3">
            <div v-for="c in stats.channel_usage" :key="c.name"
              class="flex items-center gap-2 bg-slate-50 dark:bg-slate-700 rounded-lg px-3 py-2">
              <span class="text-sm font-medium text-slate-700 dark:text-slate-200">{{ c.name }}</span>
              <span class="text-xs font-bold bg-ks-100 dark:bg-ks-900/40 text-ks-700 dark:text-ks-300 px-2 py-0.5 rounded-full">{{ c.count }}x</span>
            </div>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'

const stats   = ref({})
const loading = ref(true)

const maxDay = computed(() => {
  const days = stats.value.events_per_day || []
  return days.reduce((m, d) => Math.max(m, d.count), 0)
})

async function load() {
  loading.value = true
  try {
    const r = await fetch('/api/stats')
    if (r.ok) stats.value = await r.json()
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>
