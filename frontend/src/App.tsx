import { useEffect, useState } from 'react'
import { fetchOptions, type Options } from './api'
import { ItemForm } from './ItemForm'
import { getInitData } from './telegram'
import { texts } from './texts'

type State =
  | { kind: 'loading' }
  | { kind: 'outsideTelegram' }
  | { kind: 'failed' }
  | { kind: 'ready'; options: Options }

export function App() {
  const [state, setState] = useState<State>(() =>
    getInitData() === '' ? { kind: 'outsideTelegram' } : { kind: 'loading' },
  )

  useEffect(() => {
    if (state.kind !== 'loading') {
      return
    }

    let cancelled = false
    fetchOptions()
      .then((options) => {
        if (!cancelled) {
          setState({ kind: 'ready', options })
        }
      })
      .catch(() => {
        if (!cancelled) {
          setState({ kind: 'failed' })
        }
      })
    return () => {
      cancelled = true
    }
  }, [state.kind])

  switch (state.kind) {
    case 'loading':
      return <p className="message">{texts.loading}</p>
    case 'outsideTelegram':
      return <p className="message">{texts.openFromBot}</p>
    case 'failed':
      return <p className="message">{texts.optionsFailed}</p>
    case 'ready':
      return <ItemForm options={state.options} />
  }
}
