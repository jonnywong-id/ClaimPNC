import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'

import { App } from './app/App'
import './gaya.css'

const akar = document.getElementById('akar')
if (!akar) {
  throw new Error('elemen #akar tidak ditemukan di index.html')
}

createRoot(akar).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
