<template>
  <el-dropdown @command="handleLocaleChange" trigger="click">
    <el-button type="primary" text>
      <el-icon><Promotion /></el-icon>
      {{ currentLocaleName }}
    </el-button>
    <template #dropdown>
      <el-dropdown-menu>
        <el-dropdown-item 
          v-for="locale in availableLocales" 
          :key="locale.code" 
          :command="locale.code"
          :class="{ 'is-active': locale.code === currentLocale }"
        >
          {{ locale.name }}
        </el-dropdown-item>
      </el-dropdown-menu>
    </template>
  </el-dropdown>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { setLocale, availableLocales, getCurrentLocale } from '@/i18n'
import { Promotion } from '@element-plus/icons-vue'

const currentLocale = computed(() => getCurrentLocale())

const currentLocaleName = computed(() => {
  const locale = availableLocales.find(l => l.code === currentLocale.value)
  return locale?.name || 'English'
})

const handleLocaleChange = (localeCode: string) => {
  setLocale(localeCode as 'zh-CN' | 'en-US')
  window.location.reload()
}
</script>

<style scoped>
.el-dropdown-menu__item.is-active {
  color: var(--el-color-primary);
  font-weight: 600;
}
</style>
