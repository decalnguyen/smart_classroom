import { createContext, useContext, useState, useCallback, useEffect, useRef } from 'react'
import useWebSocket from '../hooks/useWebSocket'
import { WS_BASE_URL, notificationApi } from '../api/client'
import { useAuth } from './AuthContext'

const RealtimeContext = createContext(null)

// Generate a short alarm beep via the Web Audio API (no audio asset needed).
function playAlarmBeep() {
  try {
    const Ctx = window.AudioContext || window.webkitAudioContext
    if (!Ctx) return
    const ctx = new Ctx()
    const now = ctx.currentTime
    for (let i = 0; i < 3; i++) {
      const osc = ctx.createOscillator()
      const gain = ctx.createGain()
      osc.type = 'square'
      osc.frequency.value = 880
      gain.gain.value = 0.0001
      osc.connect(gain)
      gain.connect(ctx.destination)
      const t = now + i * 0.35
      gain.gain.setValueAtTime(0.0001, t)
      gain.gain.exponentialRampToValueAtTime(0.25, t + 0.02)
      gain.gain.exponentialRampToValueAtTime(0.0001, t + 0.3)
      osc.start(t)
      osc.stop(t + 0.32)
    }
    setTimeout(() => ctx.close(), 1500)
  } catch {
    /* audio not available */
  }
}

export function RealtimeProvider({ children }) {
  const { isAuthenticated, user } = useAuth()
  const [notifications, setNotifications] = useState([])
  const [activeAlert, setActiveAlert] = useState(null)
  const [latestNotif, setLatestNotif] = useState(null)
  const seen = useRef(new Set())

  const refresh = useCallback(async () => {
    if (!isAuthenticated) return
    try {
      const { data } = await notificationApi.list()
      const list = Array.isArray(data) ? data : []
      setNotifications(list)
      list.forEach((n) => n.id && seen.current.add(n.id))
    } catch {
      /* ignore */
    }
  }, [isAuthenticated])

  useEffect(() => {
    refresh()
  }, [refresh])

  const handleMessage = useCallback((msg) => {
    if (!msg || typeof msg !== 'object') return
    // The WS server broadcasts to all clients (it has no identity), so filter
    // here: a targeted notification (account_id != ALL) is only for its recipient.
    const myId = user?.account_id
    if (msg.account_id && msg.account_id !== 'ALL' && msg.account_id !== myId) return
    if (msg.id && seen.current.has(msg.id)) return
    if (msg.id) seen.current.add(msg.id)
    setNotifications((prev) => [msg, ...prev].slice(0, 200))
    if (msg.title === 'alert') {
      setActiveAlert(msg)
      playAlarmBeep()
    } else {
      // Show a global toast for all other notifications (leave requests, etc.)
      setLatestNotif(msg)
    }
  }, [user?.account_id])

  // Only open the notification socket when authenticated.
  const wsUrl = isAuthenticated ? `${WS_BASE_URL}/ws/notifications` : null
  const { status } = useWebSocket(wsUrl, handleMessage)

  const unreadCount = notifications.filter((n) => !n.is_read).length

  const dismissAlert = useCallback(() => setActiveAlert(null), [])
  const dismissNotif = useCallback(() => setLatestNotif(null), [])

  const markRead = useCallback(async (id) => {
    setNotifications((prev) => prev.map((n) => (n.id === id ? { ...n, is_read: true } : n)))
    try {
      await notificationApi.markRead(id, { is_read: true })
    } catch {
      /* ignore */
    }
  }, [])

  const value = {
    notifications,
    unreadCount,
    activeAlert,
    dismissAlert,
    latestNotif,
    dismissNotif,
    markRead,
    refresh,
    wsStatus: status,
  }

  return <RealtimeContext.Provider value={value}>{children}</RealtimeContext.Provider>
}

export function useRealtime() {
  const ctx = useContext(RealtimeContext)
  if (!ctx) throw new Error('useRealtime must be used within RealtimeProvider')
  return ctx
}
