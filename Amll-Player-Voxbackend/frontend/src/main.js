import { createApp } from 'vue'
import App from './App.vue'
import "./style/main.css"
import router from './router/router'
import "./components/ui/basic.css"
import { createPinia } from 'pinia'
import piniaPluginPersistedstate from 'pinia-plugin-persistedstate'
import Message from './components/ui/message'

const pinia = createPinia()
pinia.use(piniaPluginPersistedstate)
const app = createApp(App)
app.use(router)
app.use(Message)
app.use(pinia)
router.isReady().then(() => app.mount('#app'))
