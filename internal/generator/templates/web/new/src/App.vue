<template>
  <n-config-provider :locale="zhCN" :date-locale="dateZhCN" :theme="naiveTheme" :theme-overrides="overrides">
    <n-loading-bar-provider>
      <n-dialog-provider>
        <n-notification-provider>
          <n-message-provider>
            <router-view />
          </n-message-provider>
        </n-notification-provider>
      </n-dialog-provider>
    </n-loading-bar-provider>
  </n-config-provider>
</template>

<script setup lang="ts">
  import { computed, onMounted, onUnmounted, ref } from 'vue'
  import { darkTheme, zhCN, dateZhCN } from '@rong/admin-ui/naive'
  import { getThemeProvider } from './app'

  const overrides = computed(() => getThemeProvider().naiveOverrides)
  const isDarkMode = ref(false)
  let observer: MutationObserver | null = null

  function syncDarkMode() {
    isDarkMode.value = document.documentElement.classList.contains('ra-dark')
  }

  onMounted(() => {
    syncDarkMode()
    observer = new MutationObserver(syncDarkMode)
    observer.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ['class'],
    })
  })

  onUnmounted(() => {
    observer?.disconnect()
    observer = null
  })

  const naiveTheme = computed(() => (isDarkMode.value ? darkTheme : null))
</script>

<style>
  * {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
  }

  html,
  body {
    width: 100%;
    height: 100%;
    font-family: var(
      --ra-font-family-body,
      -apple-system,
      BlinkMacSystemFont,
      'Segoe UI',
      Roboto,
      'Helvetica Neue',
      Arial,
      sans-serif
    );
    background-color: var(--ra-color-bg-page);
    color: var(--ra-color-text-primary);
  }

  #app {
    width: 100%;
    height: 100%;
  }

  @media print {
    html,
    body,
    #app {
      height: auto !important;
      overflow: visible !important;
      background: var(--ra-color-bg-surface) !important;
    }

    .admin-frame {
      display: block !important;
      height: auto !important;
      padding: 0 !important;
      gap: 0 !important;
      background: var(--ra-color-bg-surface) !important;
    }

    .admin-sider,
    .sticky-top,
    .n-message-container,
    .n-notification-container,
    .n-dialog-container {
      display: none !important;
    }

    .admin-main {
      display: block !important;
      height: auto !important;
      overflow: visible !important;
      padding: 0 !important;
    }
  }
</style>
