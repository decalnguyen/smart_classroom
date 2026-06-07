import { useCallback, useState } from 'react'
import useWebSocket from './useWebSocket'
import { WS_BASE_URL } from '../api/client'

/**
 * useAttendanceStream subscribes to the realtime attendance channel and keeps a
 * rolling list of recognized students (face-scan success events). Each event:
 * { student_id, mssv, student_name, classroom_id, class_id, subject,
 *   attendance_status, detection_time, date, device_id }.
 */
export default function useAttendanceStream(maxEvents = 50) {
  const [events, setEvents] = useState([])

  const onMessage = useCallback(
    (msg) => {
      if (!msg || typeof msg !== 'object' || !msg.student_id) return
      setEvents((prev) => [{ ...msg, _ts: Date.now() }, ...prev].slice(0, maxEvents))
    },
    [maxEvents]
  )

  const { status } = useWebSocket(`${WS_BASE_URL}/ws/attendance`, onMessage)
  return { events, status }
}
