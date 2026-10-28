import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router'
import { AuthWithRouter } from '@/components/AuthWithRouter'
import { MeProvider } from '@/components/MeProvider'
import { TooltipProvider } from '@/components/ui/tooltip'
import './globals.css'
import './i18n' // initialise translations before the first render
import App from './App'
import { loadConfig } from './config.ts'

const root = createRoot(document.getElementById('root')!)

// A page opened before a new deployment asks for lazy chunks that no longer
// exist: load the new build once instead of failing (the flag stops a loop).
window.addEventListener('vite:preloadError', (event) => {
  try {
    if (sessionStorage.getItem('lq:reloaded-for-build')) return
    sessionStorage.setItem('lq:reloaded-for-build', '1')
  } catch {
    return
  }
  event.preventDefault()
  window.location.reload()
})
window.addEventListener('load', () => {
  // a build that loaded fine may reload again after the next deployment
  window.setTimeout(() => {
    try {
      sessionStorage.removeItem('lq:reloaded-for-build')
    } catch {
      // nothing to clear
    }
  }, 10_000)
})

loadConfig()
  .then((config) => {
    root.render(
      <StrictMode>
        <BrowserRouter>
          <AuthWithRouter config={config}>
            <MeProvider>
              <TooltipProvider delayDuration={150}>
                <App />
              </TooltipProvider>
            </MeProvider>
          </AuthWithRouter>
        </BrowserRouter>
      </StrictMode>,
    )
  })
  .catch((error: Error) => {
    root.render(<p className="p-4 text-destructive">{error.message}</p>)
  })
