import { useEffect, useRef, useState, useCallback } from 'react'

/**
 * useWebSocket connects to a WS endpoint and invokes onMessage for each parsed
 * JSON frame. It auto-reconnects with a small backoff and exposes connection
 * status. onMessage is held in a ref so re-renders don't re-open the socket.
 */
export default function useWebSocket(url, onMessage) {
  const [status, setStatus] = useState('connecting')
  const wsRef = useRef(null)
  const handlerRef = useRef(onMessage)
  const retryRef = useRef(null)
  const attemptRef = useRef(0)
  const closedRef = useRef(false)

  handlerRef.current = onMessage

  const MAX_ATTEMPTS = 8

  const connect = useCallback(() => {
    if (!url) return
    let ws
    try {
      ws = new WebSocket(url)
    } catch {
      setStatus('error')
      return
    }
    wsRef.current = ws

    ws.onopen = () => {
      attemptRef.current = 0 // reset backoff on a successful connection
      setStatus('open')
    }
    ws.onmessage = (ev) => {
      let data = ev.data
      try {
        data = JSON.parse(ev.data)
      } catch {
        /* keep raw string if not JSON */
      }
      if (handlerRef.current) handlerRef.current(data)
    }
    ws.onerror = () => setStatus('error')
    ws.onclose = () => {
      setStatus('closed')
      if (closedRef.current) return
      if (attemptRef.current >= MAX_ATTEMPTS) {
        setStatus('failed')
        return
      }
      // Exponential backoff capped at 30s.
      const delay = Math.min(30000, 1000 * 2 ** attemptRef.current)
      attemptRef.current += 1
      retryRef.current = setTimeout(connect, delay)
    }
  }, [url])

  useEffect(() => {
    closedRef.current = false
    connect()
    return () => {
      closedRef.current = true
      if (retryRef.current) clearTimeout(retryRef.current)
      if (wsRef.current) wsRef.current.close()
    }
  }, [connect])

  return { status }
}
