<template>
  <div class="header-right">
    <n-popover trigger="click" placement="bottom-end" :width="180">
      <template #trigger>
        <n-tooltip trigger="hover">
          <template #trigger>
            <n-button quaternary circle size="small" class="action-btn" aria-label="布局密度">
              <template #icon>
                <n-icon :size="18"><LayoutGridIcon /></n-icon>
              </template>
            </n-button>
          </template>
          布局密度
        </n-tooltip>
      </template>
      <div class="density-panel">
        <div class="density-panel__title">布局密度</div>
        <n-radio-group :value="density" @update:value="(val: DensityMode) => emit('update:density', val)">
          <n-space vertical :size="4">
            <n-radio value="compact">紧凑</n-radio>
            <n-radio value="standard">标准</n-radio>
          </n-space>
        </n-radio-group>
      </div>
    </n-popover>

    <n-tooltip trigger="hover">
      <template #trigger>
        <n-button
          quaternary
          circle
          size="small"
          class="action-btn"
          :aria-label="isDark ? '切换到亮色模式' : '切换到暗色模式'"
          @click="emit('toggle-theme')"
        >
          <template #icon>
            <n-icon :size="18">
              <MoonIcon v-if="!isDark" />
              <SunIcon v-else />
            </n-icon>
          </template>
        </n-button>
      </template>
      {{ isDark ? '切换到亮色模式' : '切换到暗色模式' }}
    </n-tooltip>

    <n-dropdown :options="userMenuOptions" @select="(key: string) => emit('user-action', key)">
      <n-button text class="user-button" aria-label="用户菜单">
        <n-avatar :size="28" round>{{ username.charAt(0) }}</n-avatar>
        <span class="username">{{ username }}</span>
        <n-icon :size="16">
          <ChevronDownIcon />
        </n-icon>
      </n-button>
    </n-dropdown>
  </div>
</template>

<script setup lang="ts">
import { h } from 'vue'
import { type DropdownOption, NIcon } from '@rong/admin-ui/naive'
import {
  ChevronDown as ChevronDownIcon,
  LayoutGrid as LayoutGridIcon,
  LogOut,
  Moon as MoonIcon,
  Sun as SunIcon,
} from 'lucide-vue-next'

export type DensityMode = 'compact' | 'standard'

defineProps<{
  density: DensityMode
  isDark: boolean
  username: string
}>()

const emit = defineEmits<{
  'update:density': [value: DensityMode]
  'toggle-theme': []
  'user-action': [key: string]
}>()

const userMenuOptions: DropdownOption[] = [
  {
    label: '退出登录',
    key: 'logout',
    icon: () => h(NIcon, null, { default: () => h(LogOut) }),
  },
]
</script>

<style scoped>
.header-right {
  display: flex;
  align-items: center;
  gap: var(--ra-spacing-2, 8px);
}

.action-btn {
  color: var(--ra-color-text-secondary, #4a5068);
}

.user-button {
  display: flex;
  align-items: center;
  gap: var(--ra-spacing-2, 8px);
  padding: var(--ra-spacing-1, 4px) var(--ra-spacing-2, 8px);
  border-radius: var(--ra-radius-md, 12px);
  transition: background var(--ra-transition-base, 200ms cubic-bezier(0.4, 0, 0.2, 1));
}

.user-button:hover {
  background: var(--ra-color-bg-hover, #e6e9f2);
}

.username {
  font-size: var(--ra-font-size-sm, 13px);
  max-width: 100px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--ra-color-text-secondary, #4a5068);
}

.density-panel__title {
  font-size: var(--ra-font-size-sm, 13px);
  font-weight: var(--ra-font-weight-semibold, 600);
  color: var(--ra-color-text-primary, #1e2235);
  margin-bottom: var(--ra-spacing-2, 8px);
}
</style>
