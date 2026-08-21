<template>
  <div class="error-page">
    <RResultState
      status="404"
      title="页面不存在"
      description="抱歉，您访问的页面不存在或已被移除"
      data-testid="error-404"
    >
      <template #actions>
        <n-space>
          <n-button type="primary" data-testid="go-home" aria-label="返回首页" @click="goHome">
            <template #icon>
              <RIcon name="home" :size="16" />
            </template>
            返回首页
          </n-button>
          <n-button data-testid="go-back" aria-label="返回上页" @click="goBack">
            <template #icon>
              <RIcon name="arrow-left" :size="16" />
            </template>
            返回上页
          </n-button>
        </n-space>
      </template>
    </RResultState>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import { RResultState, RIcon } from '@rong/admin-ui'

const router = useRouter()

function goHome(): void {
  router.push('/')
}

function goBack(): void {
  if (window.history.length > 1) {
    router.back()
  } else {
    router.push('/')
  }
}
</script>

<style scoped>
.error-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--ra-color-bg-page, #f5f7fa);
}
</style>
