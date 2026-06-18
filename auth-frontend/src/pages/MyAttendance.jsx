import { useState, useEffect, useMemo } from 'react'
import {
  Box, Card, CardContent, Typography, Chip, Skeleton, Alert, Stack,
  Table, TableContainer, TableHead, TableBody, TableRow, TableCell, Paper,
} from '@mui/material'
import { useTheme } from '@mui/material/styles'
import {
  ResponsiveContainer, BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip as RTooltip, Legend,
} from 'recharts'
import CheckCircleIcon from '@mui/icons-material/CheckCircle'
import AccessTimeIcon from '@mui/icons-material/AccessTime'
import EventNoteIcon from '@mui/icons-material/EventNote'
import HowToRegIcon from '@mui/icons-material/HowToReg'
import BarChartIcon from '@mui/icons-material/BarChart'
import PageHeader from '../components/PageHeader'
import StatCard from '../components/StatCard'
import EmptyState from '../components/EmptyState'
import { meApi, apiError } from '../api/client'

function statusChip(s) {
  if (s === 'present') return <Chip size="small" color="success" label="Có mặt" />
  if (s === 'late') return <Chip size="small" color="warning" label="Đi muộn" />
  if (s === 'excused') return <Chip size="small" color="info" label="Có phép" />
  return <Chip size="small" color="error" label="Vắng" />
}

export default function MyAttendance() {
  const theme = useTheme()
  const [data, setData] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    let active = true
    ;(async () => {
      try {
        const res = await meApi.attendance()
        if (active) setData(res.data)
      } catch (err) {
        if (active) setError(apiError(err, 'Không tải được dữ liệu điểm danh.'))
      } finally {
        if (active) setLoading(false)
      }
    })()
    return () => { active = false }
  }, [])

  const s = data?.summary || { total: 0, present: 0, late: 0 }
  const records = data?.records || []

  // Per-subject attendance (worst rate on top) — answers "which subject am I failing?"
  const bySubject = useMemo(() => {
    const m = new Map()
    for (const r of records) {
      const k = r.subject || 'Khác'
      const e = m.get(k) || { subject: k, present: 0, late: 0, excused: 0, absent: 0, total: 0 }
      if (e[r.status] != null) e[r.status] += 1
      e.total += 1
      m.set(k, e)
    }
    return [...m.values()]
      .map((e) => ({ ...e, ratePct: e.total - e.excused > 0 ? Math.round((100 * (e.present + e.late)) / (e.total - e.excused)) : 0 }))
      .sort((a, b) => a.ratePct - b.ratePct)
      .slice(0, 10)
  }, [records])

  return (
    <Box>
      <PageHeader
        title="Điểm danh của tôi"
        subtitle={data?.student ? `${data.student.student_name} · MSSV ${data.student.mssv}` : 'Lịch sử điểm danh cá nhân'}
      />

      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

      {loading ? (
        <Box sx={{ display: 'grid', gap: 2, gridTemplateColumns: { xs: '1fr', sm: 'repeat(3,1fr)' }, mb: 3 }}>
          {Array.from({ length: 3 }).map((_, i) => <Skeleton key={i} variant="rounded" height={92} />)}
        </Box>
      ) : data && data.linked === false ? (
        <Card>
          <CardContent>
            <EmptyState
              icon={<HowToRegIcon />}
              title="Tài khoản chưa liên kết hồ sơ học sinh"
              description="Tài khoản của bạn chưa được gắn với một hồ sơ học sinh, nên chưa có dữ liệu điểm danh."
            />
          </CardContent>
        </Card>
      ) : (
        <>
          <Box sx={{ display: 'grid', gap: 2, gridTemplateColumns: { xs: '1fr 1fr', sm: 'repeat(4,1fr)' }, mb: 3 }}>
            <StatCard icon={<CheckCircleIcon />} value={(s.present || 0) + (s.late || 0)} label="Buổi có mặt" color="#16a34a" />
            <StatCard icon={<AccessTimeIcon />} value={s.late} label="Đi muộn" color="#ea580c" />
            <StatCard icon={<EventNoteIcon />} value={s.excused || 0} label="Có phép" color="#0891b2" />
            <StatCard icon={<EventNoteIcon />} value={s.total} label="Tổng lượt điểm danh" color="#2563eb" />
          </Box>

          {bySubject.length > 0 && (
            <Card sx={{ mb: 3 }}>
              <CardContent>
                <Stack direction="row" alignItems="center" spacing={1} mb={0.5}>
                  <BarChartIcon color="primary" />
                  <Typography variant="h6">Chuyên cần theo môn học</Typography>
                </Stack>
                <Typography variant="caption" color="text.secondary" display="block" mb={1}>
                  Môn có tỉ lệ tham gia thấp nhất xếp trên cùng — để biết cần tập trung môn nào
                </Typography>
                <Box sx={{ height: Math.max(200, bySubject.length * 38) }}>
                  <ResponsiveContainer width="100%" height="100%">
                    <BarChart layout="vertical" data={bySubject} margin={{ top: 4, right: 16, bottom: 0, left: 8 }}>
                      <CartesianGrid strokeDasharray="3 3" stroke={theme.palette.mode === 'dark' ? 'rgba(148,163,184,0.15)' : '#eef2f7'} />
                      <XAxis type="number" allowDecimals={false} tick={{ fontSize: 11, fill: theme.palette.text.secondary }} />
                      <YAxis type="category" dataKey="subject" width={140} tick={{ fontSize: 11, fill: theme.palette.text.secondary }} />
                      <RTooltip
                        formatter={(v, n) => [`${v} buổi`, n]}
                        contentStyle={{ background: theme.palette.background.paper, border: `1px solid ${theme.palette.divider}`, borderRadius: 8 }} />
                      <Legend />
                      <Bar dataKey="present" name="Có mặt" stackId="s" fill="#16a34a" />
                      <Bar dataKey="late" name="Đi muộn" stackId="s" fill="#ea580c" />
                      <Bar dataKey="excused" name="Có phép" stackId="s" fill="#0891b2" />
                      <Bar dataKey="absent" name="Vắng" stackId="s" fill="#dc2626" radius={[0, 4, 4, 0]} />
                    </BarChart>
                  </ResponsiveContainer>
                </Box>
              </CardContent>
            </Card>
          )}

          <Card>
            <CardContent>
              <Typography variant="h6" mb={2}>Lịch sử điểm danh</Typography>
              {records.length === 0 ? (
                <EmptyState dense icon={<EventNoteIcon />} title="Chưa có dữ liệu" description="Bạn chưa được điểm danh buổi nào." />
              ) : (
                <TableContainer component={Paper} variant="outlined" sx={{ maxHeight: 540 }}>
                  <Table size="small" stickyHeader>
                    <TableHead>
                      <TableRow>
                        <TableCell>Ngày</TableCell>
                        <TableCell>Giờ</TableCell>
                        <TableCell>Môn</TableCell>
                        <TableCell>Phòng</TableCell>
                        <TableCell>Trạng thái</TableCell>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {records.map((r, i) => (
                        <TableRow key={i} hover>
                          <TableCell>{r.date}</TableCell>
                          <TableCell>{r.detection_time}</TableCell>
                          <TableCell>{r.subject || '—'}</TableCell>
                          <TableCell>{r.classroom_name || '—'}</TableCell>
                          <TableCell>{statusChip(r.status)}</TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </TableContainer>
              )}
            </CardContent>
          </Card>
        </>
      )}
    </Box>
  )
}
