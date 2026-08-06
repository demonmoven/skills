import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import ErrorBoundary from './components/ErrorBoundary'
import './i18n'
import './styles.css'
import { initTheme } from './theme'

initTheme()

// Dev-only fixture page: ?magereview-fixtures=1
// import.meta.env.DEV ensures this block is tree-shaken from production builds.
if (import.meta.env.DEV && new URLSearchParams(window.location.search).get('magereview-fixtures') === '1') {
  import('./dev/MageReviewFixtures').then(({ default: MageReviewFixtures }) => {
    ReactDOM.createRoot(document.getElementById('root') as HTMLElement).render(
      <React.StrictMode>
        <MageReviewFixtures />
      </React.StrictMode>,
    )
  })
} else {
  ReactDOM.createRoot(document.getElementById('root') as HTMLElement).render(
    <React.StrictMode>
      <ErrorBoundary>
        <App />
      </ErrorBoundary>
    </React.StrictMode>,
  )
}
