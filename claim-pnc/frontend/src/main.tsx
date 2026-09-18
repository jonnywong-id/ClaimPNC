import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'

import { App } from './app/App'
import './styles.css'

const root = document.getElementById('akar')
if (!root) {
  throw new Error('elemen #akar tidak ditemukan di index.html')
}

createRoot(root).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
