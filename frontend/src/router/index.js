import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth.js'
import HandbuchView     from '../views/HandbuchView.vue'
import LoginView        from '../views/LoginView.vue'
import DashboardView    from '../views/DashboardView.vue'
import RulesView        from '../views/RulesView.vue'
import GroupsView       from '../views/GroupsView.vue'
import EventLogView     from '../views/EventLogView.vue'
import AnnouncementsView from '../views/AnnouncementsView.vue'
import UsersView        from '../views/UsersView.vue'
import ChannelsView     from '../views/ChannelsView.vue'
import ApiKeysView      from '../views/ApiKeysView.vue'
import MeGroupsView     from '../views/MeGroupsView.vue'
import MaintenanceView         from '../views/MaintenanceView.vue'
import HeartbeatsView          from '../views/HeartbeatsView.vue'
import TemplatesView           from '../views/TemplatesView.vue'
import StatsView               from '../views/StatsView.vue'
import EscalationPoliciesView  from '../views/EscalationPoliciesView.vue'
import WebhookTargetsView      from '../views/WebhookTargetsView.vue'

const routes = [
  { path: '/login',        name: 'Login',        component: LoginView,        meta: { public: true } },
  { path: '/dashboard',    name: 'Dashboard',    component: DashboardView,    meta: { requiresAuth: true } },
  { path: '/',             name: 'Rules',        component: RulesView,        meta: { requiresAuth: true, requiresAdmin: true } },
  { path: '/groups',       name: 'Groups',       component: GroupsView,       meta: { requiresAuth: true, requiresAdmin: true } },
  { path: '/events',       name: 'EventLog',     component: EventLogView,     meta: { requiresAuth: true } },
  { path: '/announcements',name: 'Announcements',component: AnnouncementsView,meta: { requiresAuth: true, requiresAdmin: true } },
  { path: '/users',        name: 'Users',        component: UsersView,        meta: { requiresAuth: true, requiresAdmin: true } },
  { path: '/channels',     name: 'Channels',     component: ChannelsView,     meta: { requiresAuth: true, requiresAdmin: true } },
  { path: '/apikeys',      name: 'ApiKeys',      component: ApiKeysView,      meta: { requiresAuth: true, requiresAdmin: true } },
  { path: '/maintenance',  name: 'Maintenance',  component: MaintenanceView,  meta: { requiresAuth: true, requiresAdmin: true } },
  { path: '/heartbeats',   name: 'Heartbeats',   component: HeartbeatsView,   meta: { requiresAuth: true, requiresAdmin: true } },
  { path: '/templates',    name: 'Templates',    component: TemplatesView,    meta: { requiresAuth: true, requiresAdmin: true } },
  { path: '/stats',               name: 'Stats',              component: StatsView,              meta: { requiresAuth: true, requiresAdmin: true } },
  { path: '/escalation-policies',name: 'EscalationPolicies', component: EscalationPoliciesView, meta: { requiresAuth: true, requiresAdmin: true } },
  { path: '/webhook-targets',    name: 'WebhookTargets',     component: WebhookTargetsView,     meta: { requiresAuth: true, requiresAdmin: true } },
  { path: '/me/groups',          name: 'MeGroups',           component: MeGroupsView,           meta: { requiresAuth: true } },
  { path: '/handbuch',     name: 'Handbuch',     component: HandbuchView,     meta: { requiresAuth: true } },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach((to) => {
  const auth = useAuthStore()
  if (to.meta.requiresAuth && !auth.isLoggedIn) return { name: 'Login' }
  if (to.meta.requiresAdmin && !auth.isAdmin) return { name: 'EventLog' }
  if (to.name === 'Login' && auth.isLoggedIn) return { name: 'Dashboard' }
  return true
})

export default router
