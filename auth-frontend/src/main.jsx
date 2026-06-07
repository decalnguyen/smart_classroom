import React from 'react'
import ReactDOM from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import { ColorModeProvider } from './context/ColorModeContext'
import App from './App'
import { AuthProvider } from './context/AuthContext'
import ErrorBoundary from './components/ErrorBoundary'

ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <ColorModeProvider>
      <ErrorBoundary>
        <BrowserRouter>
          <AuthProvider>
            <App />
          </AuthProvider>
        </BrowserRouter>
      </ErrorBoundary>
    </ColorModeProvider>
  </React.StrictMode>
)
