<template>
  <div class="admin-frame" :class="`density-${density}`">
    <SidebarMenu
      :collapsed="collapsed"
      :active-menu="activeMenu"
      @update:collapsed="collapsed = $event"
      @select="handleMenuSelect"
    />

    <main class="admin-main">
      <div class="sticky-top">
        <header class="layout-header">
          <div class="header-left">
            <n-breadcrumb>
              <n-breadcrumb-item v-for="item in breadcrumbs" :key="item.path">
                {{ item.title }}
              </n-breadcrumb-item>
            </n-breadcrumb>
          </div>
          <div class="header-center">
            <n-input
              placeholder="搜索功能、菜单…"
              size="small"
              class="header-search"
              :style="{ width: '240px' }"
            >
              <template #prefix>
                <n-icon :size="16"><SearchIcon /></n-icon>
              </template>
            </n-input>
          </div>
          <TopbarActions
            :density="density"
            :is-dark="isDark"
            :username="userStore.username"
            @update:density="setDensity"
            @toggle-theme="toggleTheme"
            @user-action="handleUserMenuSelect"
          />
        </header>

        <div class="tabs-card">
          <RTabsView
            :tabs="openTabs"
            :active-key="activeTabKey"
            @select="handleTabSelect"
            @close="handleTabClose"
            @close-other="handleTabCloseOther"
            @close-all="handleTabCloseAll"
          />
        </div>
      </div>

      <router-view />
    </main>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useMessage, NIcon } from '@rong/admin-ui/naive'
import { Search as SearchIcon } from 'lucide-vue-next'
import { RTabsView } from '@rong/admin-ui'
import type { TabItem } from '@rong/admin-ui'
import { useUserStore } from '@/store/modules/user'
import { getThemeProvider } from '@/app'
import { storage } from '@/utils/storage'
import SidebarMenu from './components/SidebarMenu.vue'
import TopbarActions from './components/TopbarActions.vue'
import type { DensityMode } from './components/TopbarActions.vue'

const router = useRouter()
const route = useRoute()
const message = useMessage()
const userStore = useUserStore()

const collapsed = ref(false)

const DENSITY_KEY = 'ra-layout-density'

function loadDensity(): DensityMode {
  const saved = storage.get(DENSITY_KEY)
  if (saved === 'compact' || saved === 'standard') return saved
  return 'compact'
}

const density = ref<DensityMode>(loadDensity())

function setDensity(val: DensityMode) {
  density.value = val
  storage.set(DENSITY_KEY, val)
}

const isDark = ref(false)
let themeObserver: MutationObserver | null = null

function syncThemeMode(): void {
  const provider = getThemeProvider()
  const htmlDark = document.documentElement.classList.contains('ra-dark')
  isDark.value = htmlDark || provider.currentMode === 'dark'
}

function toggleTheme() {
  const provider = getThemeProvider()
  const nextMode = provider.currentMode === 'dark' ? 'light' : 'dark'
  provider.setMode(nextMode)
  document.documentElement.classList.toggle('ra-dark', nextMode === 'dark')
  syncThemeMode()
}

onMounted(() => {
  syncThemeMode()
  themeObserver = new MutationObserver(syncThemeMode)
  themeObserver.observe(document.documentElement, {
    attributes: true,
    attributeFilter: ['class'],
  })
})

onUnmounted(() => {
  themeObserver?.disconnect()
  themeObserver = null
})

const AFFIX_TAB: TabItem = { key: '/dashboard', label: '首页', path: '/dashboard', affix: true }
const openTabs = ref<TabItem[]>([AFFIX_TAB])

function buildTabRouteKey(): string {
  return route.fullPath
}

function resolveRouteTabId(): string {
  const raw = route.query.tabId ?? route.params.id
  if (Array.isArray(raw)) {
    return raw[0] != null ? String(raw[0]) : ''
  }
  return raw != null ? String(raw) : ''
}

function buildTabLabel(baseTitle: string): string {
  const tabId = resolveRouteTabId().trim()
  if (!tabId) return baseTitle
  return `${baseTitle} #${tabId}`
}

const activeTabKey = computed(() => buildTabRouteKey())

function findOrCreateTab(routeKey: string, title: string, path: string): void {
  if (!routeKey || route.name === 'Layout') return
  const exists = openTabs.value.find((t) => t.key === routeKey)
  if (exists) {
    if (exists.label !== title) exists.label = title
    if (exists.path !== path) exists.path = path
    return
  }
  openTabs.value.push({ key: routeKey, label: title, path })
}

watch(
  () => buildTabRouteKey(),
  (routeKey) => {
    const baseTitle = (route.meta?.title as string) || (route.name as string)
    const title = buildTabLabel(baseTitle)
    findOrCreateTab(routeKey, title, routeKey)
  },
  { immediate: true },
)

function handleTabSelect(key: string) {
  const tab = openTabs.value.find((t) => t.key === key)
  if (tab) router.push(tab.path)
}

function handleTabClose(key: string) {
  const idx = openTabs.value.findIndex((t) => t.key === key)
  if (idx === -1) return
  const tab = openTabs.value[idx]
  if (tab.affix) return
  openTabs.value.splice(idx, 1)
  if (activeTabKey.value === key) {
    const next = openTabs.value[Math.min(idx, openTabs.value.length - 1)]
    if (next) router.push(next.path)
  }
}

function handleTabCloseOther(key: string) {
  openTabs.value = openTabs.value.filter((t) => t.affix || t.key === key)
  const current = openTabs.value.find((t) => t.key === key)
  if (current && activeTabKey.value !== key) router.push(current.path)
}

function handleTabCloseAll() {
  openTabs.value = openTabs.value.filter((t) => t.affix)
  const first = openTabs.value[0]
  if (first) router.push(first.path)
}

const activeMenu = computed(() => route.name as string)

const breadcrumbs = computed(() => {
  const items: Array<{ path: string; title: string }> = []
  route.matched.forEach((record) => {
    if (record.meta?.title) {
      items.push({ path: record.path, title: record.meta.title as string })
    }
  })
  return items
})

function handleMenuSelect(key: string) {
  router.push({ name: key })
}

function handleUserMenuSelect(key: string) {
  if (key === 'logout') {
    userStore.logout()
    message.success('已退出登录')
    router.push('/login')
  }
}
</script>

<style scoped>
.density-compact {
  --layout-frame-padding: 6px;
  --layout-frame-gap: 6px;
  --layout-sider-width: 220px;
  --layout-header-height: 48px;
  --layout-card-radius: var(--ra-radius-lg, 12px);
  --layout-page-gap: 16px;
  --ra-card-padding-x: var(--ra-spacing-5, 20px);
}

.density-standard {
  --layout-frame-padding: 12px;
  --layout-frame-gap: 10px;
  --layout-sider-width: 240px;
  --layout-header-height: 56px;
  --layout-card-radius: var(--ra-radius-lg, 12px);
  --layout-page-gap: 20px;
  --ra-card-padding-x: var(--ra-spacing-6, 24px);
}

.admin-frame {
  display: flex;
  height: 100vh;
  padding: var(--layout-frame-padding);
  gap: var(--layout-frame-gap);
  background: var(--ra-color-bg-page, #f4f6fa);
  box-sizing: border-box;
}

.admin-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  overflow-y: auto;
  scrollbar-width: thin;
  scrollbar-color: var(--ra-color-border-default, #e2e6f0) transparent;
  padding-bottom: var(--layout-frame-gap);
}

.sticky-top {
  position: sticky;
  top: 0;
  z-index: 10;
  display: flex;
  flex-direction: column;
  gap: var(--layout-frame-gap);
  flex-shrink: 0;
  background: var(--ra-color-bg-page, #f4f6fa);
  padding-bottom: var(--layout-frame-gap);
}

.layout-header {
  height: var(--layout-header-height);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 var(--ra-card-padding-x);
  flex-shrink: 0;
  background: var(--ra-color-bg-surface, #fff);
  border: 1px solid var(--ra-color-border-light, #eef0f6);
  border-radius: var(--layout-card-radius);
  box-shadow: var(--ra-shadow-card, 0 1px 3px 0 rgb(0 0 0 / 0.04), 0 1px 2px -1px rgb(0 0 0 / 0.03));
}

.header-left {
  display: flex;
  align-items: center;
}

.header-center {
  flex: 1;
  display: flex;
  justify-content: center;
}

.header-search {
  border-radius: var(--ra-radius-full, 9999px);
}

.tabs-card {
  flex-shrink: 0;
  border-radius: var(--layout-card-radius);
  overflow: hidden;
  background: var(--ra-color-bg-surface, #fff);
  border: 1px solid var(--ra-color-border-light, #eef0f6);
  box-shadow: var(--ra-shadow-card, 0 1px 3px 0 rgb(0 0 0 / 0.04), 0 1px 2px -1px rgb(0 0 0 / 0.03));
}

.tabs-card :deep(.r-tabs-view) {
  border-bottom: none;
}

.admin-main :deep(.ra-page) {
  gap: var(--layout-page-gap);
}
</style>
