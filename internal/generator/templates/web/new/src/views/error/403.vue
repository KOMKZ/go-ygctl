<template>
  <div class="forbidden-page" data-testid="forbidden-page">
    <div class="forbidden-card">
      <h1>403</h1>
      <h2>无权限访问</h2>
      <p>当前账号没有访问该页面的权限，请联系管理员分配权限后重试。</p>
      <router-link to="/dashboard" class="forbidden-link">返回首页</router-link>
    </div>
  </div>
</template>

<style scoped>
.forbidden-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  background: radial-gradient(circle at 20% 20%, #f4f7ff 0, #eef3ff 45%, #f8fafc 100%);
}

.forbidden-card {
  width: min(520px, 100%);
  padding: 32px;
  border-radius: 20px;
  border: 1px solid var(--ra-color-border-light, #e6ebf5);
  background: var(--ra-color-bg-surface, #fff);
  box-shadow: var(--ra-shadow-card, 0 20px 40px rgb(15 23 42 / 8%));
  text-align: center;
}

.forbidden-card h1 {
  margin: 0;
  font-size: 56px;
  line-height: 1;
  color: var(--ra-color-danger, #e54d4d);
}

.forbidden-card h2 {
  margin: 12px 0 8px;
  font-size: 22px;
  color: var(--ra-color-text-primary, #1e2235);
}

.forbidden-card p {
  margin: 0 0 20px;
  color: var(--ra-color-text-secondary, #4e5568);
}

.forbidden-link {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 10px 16px;
  border-radius: 10px;
  background: var(--ra-color-brand-primary, #6b93f5);
  color: #fff;
  text-decoration: none;
  transition: transform 0.15s ease;
}

.forbidden-link:hover {
  transform: translateY(-1px);
}
</style>
