import { createApp } from 'vue'
import ElementPlus from 'element-plus'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'
import 'element-plus/dist/index.css'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import en from 'element-plus/es/locale/lang/en'

import App from './App.vue'
import router from './router'
import { createPinia } from 'pinia'
import i18n, { getCurrentLocale } from './i18n'

import './styles/variables.scss'
import './styles/index.scss'
import './styles/apple.scss'

import { useTheme } from './composables/useTheme'
useTheme()

const app = createApp(App)

for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
  app.component(key, component)
}

const elementLocale = getCurrentLocale() === 'zh-CN' ? zhCn : en

app.use(ElementPlus, { locale: elementLocale })
app.use(router)
app.use(createPinia())
app.use(i18n)

app.mount('#app')
