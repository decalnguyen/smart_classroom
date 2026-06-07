import { useState, useEffect } from 'react'
import {
  Box, Card, CardContent, Typography, Chip, Skeleton, Alert,
  Table, TableContainer, TableHead, TableBody, TableRow, TableCell, Paper,
} from '@mui/material'
import CheckCircleIcon from '@mui/icons-material/CheckCircle'
import AccessTimeIcon from '@mui/icons-material/AccessTime'
import EventNoteIcon from '@mui/icons-material/EventNote'
import HowToRegIcon from '@mui/icons-material/HowToReg'
import PageHeader from '../components/PageHeader'
import StatCard from '../components/StatCard'
import EmptyState from '../components/EmptyState'
import { meApi, apiError } from '../api/client'

function statusChip(s) {
  if (s === 'present') return <Chip size="small" color="success" label="Có mặt" />
  if (s === 'late') return <Chip size="small" color="warning" label="Đi muộn" />
  return <Chip size="small" color="error" label="Vắng" />
}

export default function MyAttendance() {
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
          <Box sx={{ display: 'grid', gap: 2, gridTemplateColumns: { xs: '1fr', sm: 'repeat(3,1fr)' }, mb: 3 }}>
            <StatCard icon={<CheckCircleIcon />} value={s.present} label="Buổi có mặt" color="#16a34a" />
            <StatCard icon={<AccessTimeIcon />} value={s.late} label="Đi muộn" color="#ea580c" />
            <StatCard icon={<EventNoteIcon />} value={s.total} label="Tổng lượt điểm danh" color="#2563eb" />
          </Box>

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
