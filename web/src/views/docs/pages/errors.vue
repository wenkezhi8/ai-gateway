<template>
  <section class="docs-route-page">
    <header class="route-head">
      <p class="route-badge">Error Codes</p>
      <h1>错误码</h1>
      <p>常见 HTTP 状态码、业务错误码与错误响应格式。</p>
    </header>

    <LegacyDocsContent initial-tab="errors" :hide-header="true" :hide-tabs="true" />
  </section>
</template>

<script setup lang="ts">
import LegacyDocsContent from '../legacy-content.vue'
</script>

<style scoped>
.docs-route-page {
  border: 1px solid rgba(98, 136, 164, 0.24);
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.86);
  padding: 0.95rem;
}

.route-head {
  margin-bottom: 0.8rem;
}

.route-head h1 {
  margin: 0.35rem 0 0;
  font-size: 1.55rem;
}

.route-head p {
  margin: 0.45rem 0 0;
  color: #4f667b;
}

.route-badge {
  margin: 0;
  display: inline-flex;
  align-items: center;
  padding: 0.22rem 0.58rem;
  border-radius: 999px;
  font-size: 0.78rem;
  color: #0f766e;
  background: rgba(14, 165, 160, 0.12);
}
</style>
