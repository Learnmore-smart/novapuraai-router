import { useEffect, useRef } from 'react'

declare global {
  interface Window {
    turnstile?: {
      render: (element: HTMLElement, options: Record<string, unknown>) => string
      remove: (widgetId: string) => void
      reset: (widgetId: string) => void
    }
  }
}

interface TurnstileProps {
  siteKey: string
  action: string
  onVerify: (token: string) => void
  onExpire?: () => void
  resetKey?: number
  className?: string
}

export function Turnstile(props: TurnstileProps) {
  const ref = useRef<HTMLDivElement | null>(null)
  const callbacksRef = useRef({
    onVerify: props.onVerify,
    onExpire: props.onExpire,
  })
  callbacksRef.current = {
    onVerify: props.onVerify,
    onExpire: props.onExpire,
  }

  useEffect(() => {
    let cancelled = false
    let widgetId: string | undefined
    const container = ref.current
    let script = document.querySelector<HTMLScriptElement>('#cf-turnstile')

    const render = () => {
      if (cancelled || !container || !window.turnstile) return
      container.replaceChildren()
      try {
        widgetId = window.turnstile.render(container, {
          sitekey: props.siteKey,
          action: props.action,
          callback: (token: string) => callbacksRef.current.onVerify(token),
          'error-callback': () => callbacksRef.current.onExpire?.(),
          'expired-callback': () => callbacksRef.current.onExpire?.(),
        })
      } catch {
        callbacksRef.current.onExpire?.()
      }
    }

    if (window.turnstile) {
      render()
    } else {
      if (!script) {
        script = document.createElement('script')
        script.id = 'cf-turnstile'
        script.src =
          'https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit'
        script.async = true
        script.defer = true
        document.head.appendChild(script)
      }
      script.addEventListener('load', render)
    }

    return () => {
      cancelled = true
      script?.removeEventListener('load', render)
      if (widgetId && window.turnstile) {
        window.turnstile.remove(widgetId)
      } else {
        container?.replaceChildren()
      }
    }
  }, [props.action, props.resetKey, props.siteKey])

  return <div ref={ref} className={props.className} />
}
