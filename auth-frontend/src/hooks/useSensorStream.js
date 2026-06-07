import { useCallback, useRef, useState } from 'react'
import useWebSocket from './useWebSocket'
import { WS_BASE_URL } from '../api/client'

// Normalise a raw device_type into one of our known metrics.
function metricOf(deviceType = '') {
  const t = deviceType.toLowerCase()
  if (t.includes('smoke') || t.includes('mq2') || t.includes('gas')) return 'smoke'
  if (t.includes('temp')) return 'temperature'
  if (t.includes('hum')) return 'humidity'
  if (t.includes('light') || t.includes('lux')) return 'light'
  return t
}

/**
 * useSensorStream subscribes to the realtime sensor WS and maintains:
 *  - latest: { [metric]: { value, device_id, timestamp } }
 *  - points: rolling array of merged snapshots for charting
 *  - devices: { [device_id]: lastReading }
 */
export default function useSensorStream(maxPoints = 40) {
  const [latest, setLatest] = useState({})
  const [points, setPoints] = useState([])
  const [devices, setDevices] = useState({})
  const snapshot = useRef({})

  const onMessage = useCallback(
    (msg) => {
      if (!msg || typeof msg !== 'object' || !msg.device_type) return
      const metric = metricOf(msg.device_type)
      const value = Number(msg.value)
      const ts = msg.timestamp ? new Date(msg.timestamp) : new Date()

      snapshot.current = { ...snapshot.current, [metric]: value }

      setLatest((prev) => ({
        ...prev,
        [metric]: { value, device_id: msg.device_id, timestamp: msg.timestamp },
      }))
      setDevices((prev) => ({ ...prev, [msg.device_id]: { ...msg, metric } }))
      setPoints((prev) => {
        const next = [
          ...prev,
          {
            time: ts.toLocaleTimeString('vi-VN', { hour12: false }),
            ...snapshot.current,
          },
        ]
        return next.slice(-maxPoints)
      })
    },
    [maxPoints]
  )

  const { status } = useWebSocket(`${WS_BASE_URL}/ws/sensor`, onMessage)

  return { latest, points, devices, status }
}
