import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import { useThemeStore } from './stores/themeStore'
import './style.css'

const app = createApp(App)
const pinia = createPinia()

app.use(pinia)
app.use(router)

// Initialize theme from localStorage before mounting
const themeStore = useThemeStore(pinia)
themeStore.initTheme()

app.mount('#app')
