import { createApp } from 'vue'
import App from './App.vue'
import { router } from './router'
import { consumeConnectRedirect } from './lib/api'
import { toast } from './lib/toast'
import './style.css'

// Back from the main site's full-page sign-in (WeChat / blocked popup): take the key out of the
// URL fragment before anything else reads the address.
consumeConnectRedirect()
  .then((signedIn) => signedIn && toast('登录成功', 'success'))
  .catch(() => {})

createApp(App).use(router).mount('#app')
