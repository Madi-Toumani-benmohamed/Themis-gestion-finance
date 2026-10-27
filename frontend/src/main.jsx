import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import { ToastProvider } from './Toast'
import { MotionProvider } from './motion'
import '@fontsource-variable/inter'
import '@fontsource/fraunces/400.css'
import '@fontsource/ibm-plex-mono/400.css'
import '@fontsource/ibm-plex-mono/500.css'
import './styles.css'
import './refonte.css'
import './cockpit.css'
import './motion/motion.css'

ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <MotionProvider><ToastProvider>
      <App />
    </ToastProvider></MotionProvider>
  </React.StrictMode>,
)
