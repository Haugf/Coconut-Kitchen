import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import '@fontsource-variable/fraunces/full.css'
import '@fontsource-variable/instrument-sans'
import './styles.css'
import App from './App'
import Settings from './settings/Settings'

// mirror.local:8080/settings is the phone page; everything else is the mirror.
const isSettings = window.location.pathname.replace(/\/+$/, '') === '/settings'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    {isSettings ? <Settings /> : <App />}
  </StrictMode>,
)
