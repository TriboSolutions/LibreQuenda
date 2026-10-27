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
