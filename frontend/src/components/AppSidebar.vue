<template>
  <aside
    class="flex flex-col bg-ks-700 dark:bg-slate-900 h-screen flex-shrink-0 transition-all duration-200 overflow-hidden z-40"
    :class="sidebar.collapsed ? 'w-14' : 'w-56'"
  >
    <!-- Header -->
    <div class="flex items-center gap-2 h-14 px-2 border-b border-white/10 flex-shrink-0">
      <button
        @click="sidebar.toggle()"
        class="p-1.5 rounded text-white/60 hover:text-white hover:bg-white/10 transition-colors flex-shrink-0"
        :title="sidebar.collapsed ? 'Aufklappen' : 'Einklappen'"
      >
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16"/>
        </svg>
      </button>
      <a v-show="!sidebar.collapsed" href="/dashboard" class="flex items-center gap-2 overflow-hidden min-w-0">
        <div class="leading-tight min-w-0">
          <div class="text-white font-bold text-sm tracking-wide whitespace-nowrap">Alarm-Core</div>
          <div class="text-white/40 text-[9px] tracking-widest uppercase whitespace-nowrap">Manager</div>
        </div>
      </a>
    </div>

    <!-- Nav -->
    <nav class="flex-1 overflow-y-auto overflow-x-hidden py-2 space-y-0.5 px-1.5">
      <template v-for="link in mainLinks" :key="link.to">
        <NavLink v-if="!link.adminOnly || auth.isAdmin" :link="link" :collapsed="sidebar.collapsed" />
      </template>

      <!-- Separator before admin section -->
      <div v-if="auth.isAdmin" class="my-2 border-t border-white/10 mx-1"></div>

      <template v-if="auth.isAdmin" v-for="link in adminLinks" :key="link.to">
        <NavLink :link="link" :collapsed="sidebar.collapsed" />
      </template>
    </nav>

    <!-- Bottom -->
    <div class="border-t border-white/10 px-2 py-2 flex-shrink-0 space-y-1.5">
      <div class="flex items-center gap-0.5" :class="sidebar.collapsed ? 'flex-col' : ''">
        <button
          @click="$emit('openHelp')"
          class="p-1.5 rounded text-white/60 hover:text-white hover:bg-white/10 transition-colors text-sm font-bold w-7 h-7 flex items-center justify-center"
          title="Kontexthilfe (F1)"
        >?</button>
        <router-link
          to="/handbuch"
          class="p-1.5 rounded text-white/60 hover:text-white hover:bg-white/10 transition-colors flex-shrink-0"
          title="Handbuch"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253"/>
          </svg>
        </router-link>
        <button
          @click="theme.toggle()"
          class="p-1.5 rounded text-white/60 hover:text-white hover:bg-white/10 transition-colors"
          :title="theme.isDark ? 'Light Mode' : 'Dark Mode'"
        >
          <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
            <path v-if="theme.isDark" fill-rule="evenodd" clip-rule="evenodd" d="M10 2a1 1 0 011 1v1a1 1 0 11-2 0V3a1 1 0 011-1zm4 8a4 4 0 11-8 0 4 4 0 018 0zm-.464 4.95l.707.707a1 1 0 001.414-1.414l-.707-.707a1 1 0 00-1.414 1.414zm2.12-10.607a1 1 0 010 1.414l-.706.707a1 1 0 11-1.414-1.414l.707-.707a1 1 0 011.414 0zM17 11a1 1 0 100-2h-1a1 1 0 100 2h1zm-7 4a1 1 0 011 1v1a1 1 0 11-2 0v-1a1 1 0 011-1zM5.05 6.464A1 1 0 106.465 5.05l-.708-.707a1 1 0 00-1.414 1.414l.707.707zm1.414 8.486l-.707.707a1 1 0 01-1.414-1.414l.707-.707a1 1 0 011.414 1.414zM4 11a1 1 0 100-2H3a1 1 0 000 2h1z"/>
            <path v-else d="M17.293 13.293A8 8 0 016.707 2.707a8.001 8.001 0 1010.586 10.586z"/>
          </svg>
        </button>
      </div>

      <div v-if="!sidebar.collapsed" class="flex items-center gap-2 px-1">
        <div class="flex-1 min-w-0">
          <div class="text-white text-xs font-medium truncate">{{ auth.user?.username }}</div>
          <div class="text-white/40 text-[10px] uppercase tracking-wide">{{ auth.user?.role }}</div>
        </div>
        <button @click="logout" class="p-1.5 rounded text-white/50 hover:text-white hover:bg-white/10 transition-colors flex-shrink-0" title="Abmelden">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1"/>
          </svg>
        </button>
      </div>
      <div v-else class="flex justify-center">
        <button @click="logout" class="p-1.5 rounded text-white/50 hover:text-white hover:bg-white/10 transition-colors" title="Abmelden">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1"/>
          </svg>
        </button>
      </div>
    </div>
  </aside>
</template>

<script setup>
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth.js'
import { useThemeStore } from '../stores/theme.js'
import { useSidebarStore } from '../stores/sidebar.js'
import NavLink from './SidebarNavLink.vue'

defineEmits(['openHelp'])

const auth    = useAuthStore()
const theme   = useThemeStore()
const sidebar = useSidebarStore()
const router  = useRouter()

const IC = {
  dashboard: 'M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6',
  shield:    'M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z',
  duty:      'M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z',
  users:     'M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z',
  clipboard: 'M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 8h6m-6 4h6',
  bell:      'M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9',
  cog:       'M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065zM15 12a3 3 0 11-6 0 3 3 0 016 0z',
  plug:      'M7 16V4m0 0L3 8m4-4l4 4M17 8V4m0 0l-4 4m4-4l4 4M3 20h18',
  key:       'M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z',
  wrench:    'M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z',
  heartbeat: 'M4.318 6.318a4.5 4.5 0 000 6.364L12 20.364l7.682-7.682a4.5 4.5 0 00-6.364-6.364L12 7.636l-1.318-1.318a4.5 4.5 0 00-6.364 0z',
  template:  'M4 5a1 1 0 011-1h14a1 1 0 011 1v2a1 1 0 01-1 1H5a1 1 0 01-1-1V5zM4 13a1 1 0 011-1h6a1 1 0 011 1v6a1 1 0 01-1 1H5a1 1 0 01-1-1v-6zM16 13a1 1 0 011-1h2a1 1 0 011 1v6a1 1 0 01-1 1h-2a1 1 0 01-1-1v-6z',
  chart:     'M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z',
  escalate:  'M13 7h8m0 0v8m0-8l-8 8-4-4-6 6',
  webhook:   'M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1',
}

// Always visible to logged-in users
const mainLinks = [
  { to: '/dashboard', label: 'Dashboard',   icon: IC.dashboard, exact: false, adminOnly: false },
  { to: '/me/groups', label: 'Mein Einsatz',icon: IC.duty,      exact: false, adminOnly: false },
  { to: '/events',    label: 'Event-Log',   icon: IC.clipboard, exact: false, adminOnly: false },
]

// Admin-only section
const adminLinks = [
  { to: '/',              label: 'Regeln',       icon: IC.shield,    exact: true  },
  { to: '/groups',        label: 'Gruppen',      icon: IC.users,     exact: false },
  { to: '/announcements', label: 'Ansagen',      icon: IC.bell,      exact: false },
  { to: '/maintenance',   label: 'Wartung',      icon: IC.wrench,    exact: false },
  { to: '/heartbeats',    label: 'Heartbeats',   icon: IC.heartbeat, exact: false },
  { to: '/templates',     label: 'Templates',    icon: IC.template,  exact: false },
  { to: '/stats',               label: 'Statistiken',   icon: IC.chart,    exact: false },
  { to: '/escalation-policies', label: 'Eskalation',    icon: IC.escalate, exact: false },
  { to: '/webhook-targets',     label: 'Webhooks',      icon: IC.webhook,  exact: false },
  { to: '/channels',            label: 'Kanäle',        icon: IC.plug,     exact: false },
  { to: '/apikeys',       label: 'API Keys',     icon: IC.key,       exact: false },
  { to: '/users',         label: 'Benutzer',     icon: IC.cog,       exact: false },
]

function logout() {
  auth.logout()
  router.push('/login')
}
</script>
