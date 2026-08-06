import { Component, type ErrorInfo, type ReactNode } from 'react'
import i18n from '../i18n'

type Props = {
  children: ReactNode
  fallback?: (error: Error, reset: () => void) => ReactNode
}

type State = {
  error: Error | null
}

export default class ErrorBoundary extends Component<Props, State> {
  state: State = { error: null }

  static getDerivedStateFromError(error: Error): State {
    return { error }
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error('[ErrorBoundary] Caught error:', error, info)
  }

  reset = () => {
    this.setState({ error: null })
  }

  render() {
    if (this.state.error) {
      if (this.props.fallback) {
        return this.props.fallback(this.state.error, this.reset)
      }
      return (
        <div style={{ padding: 24, color: '#ff6b6b', background: '#1a1a2e', minHeight: '100vh' }}>
          <h2>{i18n.t('errorBoundary.title')}</h2>
          <pre style={{ whiteSpace: 'pre-wrap', fontSize: 14 }}>{this.state.error.message}</pre>
          <pre className="mt-3" style={{ whiteSpace: 'pre-wrap', fontSize: 12, opacity: 0.7 }}>
            {this.state.error.stack}
          </pre>
          <button
            onClick={this.reset}
            className="mt-4"
            style={{ padding: '8px 16px', cursor: 'pointer' }}
          >
            {i18n.t('errorBoundary.retry')}
          </button>
        </div>
      )
    }
    return this.props.children
  }
}
