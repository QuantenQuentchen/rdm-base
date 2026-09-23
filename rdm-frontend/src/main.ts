import { createApp } from 'vue'
import App from './App.vue'

// The only stylesheet the app needs. If `npm create vue` left an
// `src/assets/main.css` behind, delete it — don't import it here.
import './styles/theme.css'

createApp(App).mount('#app')
