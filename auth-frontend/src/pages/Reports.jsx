import { useState, useEffect, useCallback } from 'react'
import {
  Box,
  Card,
  CardContent,
  Typography,
  Stack,
  TextField,
  Button,
  Skeleton,
  Alert,
  Table,
  TableContainer,
  TableHead,
  TableBody,
  TableRow,
  TableCell,
  Paper,
  Chip,
  LinearProgress,
} from '@mui/material'
import FileDownloadIcon from '@mui/icons-material/FileDownload'
import { useTheme } from '@mui/material/styles'
import CheckCircleIcon from '@mui/icons-material/CheckCircle'
import CancelIcon from '@mui/icons-material/Cancel'
import AccessTimeIcon from '@mui/icons-material/AccessTime'
import PercentIcon from '@mui/icons-material/Percent'
import GroupsIcon from '@mui/icons-material/Groups'
import AssessmentIcon from '@mui/icons-material/Assessment'
import {
  ResponsiveContainer, BarChart, Bar, PieChart, Pie, Cell,
  LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip as RTooltip, Legend,
} from 'recharts'
import PageHeader from '../components/PageHeader'
import StatCard from '../components/StatCard'
import EmptyState from '../components/EmptyState'
import { reportApi, apiError } from '../api/client'
import { useAuth } from '../context/AuthContext'

function todayStr(offsetDays = 0) {
  const d = new Date()
  d.setDate(d.getDate() + offsetDays)
  return d.toISOString().slice(0, 10)
}

export default function Reports() {
  const theme = useTheme()
  const { role } = useAuth()
  const [date, setDate] = useState(todayStr())
  const [from, setFrom] = useState(todayStr(-6))
  const [to, setTo] = useState(todayStr())
  const [data, setData] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const load = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const res = await reportApi.attendance({ date, from, to })
      setData(res.data)
    } catch (err) {
      setError(apiError(err, 'Không tải được báo cáo.'))
      setData(null)
    } finally {
      setLoading(false)
    }
  }, [date, from, to])

  useEffect(() => { load() }, [load])

  const [exporting, setExporting] = useState(false)
  const handleExport = async () => {
    setExporting(true)
    try {
      await reportApi.exportCsv({ date })
    } catch (err) {
      setError(apiError(err, 'Xuất CSV thất bại.'))
    } finally {
      setExporting(false)
    }
  }

  const grid = theme.palette.mode === 'dark' ? 'rgba(148,163,184,0.15)' : '#eef2f7'
  const axisTick = { fontSize: 11, fill: theme.palette.text.secondary }
  const totals = data?.totals || { present: 0, late: 0, absent: 0, enrolled: 0, rate: 0 }
  const ratePct = Math.round((totals.rate || 0) * 100)
  const byClassroom = data?.by_classroom || []
  const byDate = data?.by_date || []
  const pieData = [
    { name: 'Có mặt', value: totals.present },
    { name: 'Đi muộn', value: totals.late || 0 },
    { name: 'Vắng', value: totals.absent },
  ]
  const PIE_COLORS = ['#16a34a', '#ea580c', '#dc2626']

  const dateInputs = (
    <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1.5} alignItems="center">
      <TextField size="small" type="date" label="Ngày" value={date} onChange={(e) => setDate(e.target.value)} InputLabelProps={{ shrink: true }} />
      <TextField size="small" type="date" label="Từ" value={from} onChange={(e) => setFrom(e.target.value)} InputLabelProps={{ shrink: true }} />
      <TextField size="small" type="date" label="Đến" value={to} onChange={(e) => setTo(e.target.value)} InputLabelProps={{ shrink: true }} />
      <Button variant="outlined" startIcon={<FileDownloadIcon />} onClick={handleExport} disabled={exporting}>
        {exporting ? 'Đang xuất...' : 'Xuất CSV'}
      </Button>
    </Stack>
  )

  return (
    <Box>
      <PageHeader
        title="Báo cáo điểm danh"
        subtitle={role === 'teacher' ? 'Thống kê các lớp bạn được phân công' : 'Thống kê toàn bộ lớp học, phân tích theo từng lớp'}
        action={dateInputs}
      />

      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

      {/* KPI */}
      <Box sx={{ display: 'grid', gap: 2, gridTemplateColumns: { xs: '1fr 1fr', md: 'repeat(5, 1fr)' }, mb: 3 }}>
        {loading ? (
          Array.from({ length: 5 }).map((_, i) => <Skeleton key={i} variant="rounded" height={92} />)
        ) : (
          <>
            <StatCard icon={<CheckCircleIcon />} value={totals.present} label="Có mặt" color="#16a34a" />
            <StatCard icon={<AccessTimeIcon />} value={totals.late || 0} label="Đi muộn" color="#ea580c" />
            <StatCard icon={<CancelIcon />} value={totals.absent} label="Vắng" color="#dc2626" />
            <StatCard icon={<GroupsIcon />} value={totals.enrolled} label="Sĩ số (ngày)" color="#2563eb" />
            <StatCard icon={<PercentIcon />} value={`${ratePct}%`} label="Tỉ lệ tham gia" color="#7c3aed" />
          </>
        )}
      </Box>

      {!loading && byClassroom.length === 0 ? (
        <Card>
          <CardContent>
            <EmptyState
              icon={<AssessmentIcon />}
              title="Chưa có dữ liệu"
              description={role === 'teacher' ? 'Bạn chưa được phân công lớp nào, hoặc chưa có dữ liệu điểm danh cho ngày đã chọn.' : 'Không có dữ liệu điểm danh cho ngày đã chọn.'}
            />
          </CardContent>
        </Card>
      ) : (
        <>
          <Box sx={{ display: 'grid', gap: 2, gridTemplateColumns: { xs: '1fr', lg: '2fr 1fr' }, mb: 3 }}>
            {/* Per-classroom bar chart */}
            <Card>
              <CardContent>
                <Typography variant="h6" mb={1}>Có mặt / Vắng theo từng lớp ({data?.date})</Typography>
                <Box sx={{ height: 340 }}>
                  {loading ? <Skeleton variant="rounded" height={320} /> : (
                    <ResponsiveContainer width="100%" height="100%">
                      <BarChart data={byClassroom} margin={{ top: 8, right: 12, bottom: 0, left: -12 }}>
                        <CartesianGrid strokeDasharray="3 3" stroke={grid} />
                        <XAxis dataKey="classroom_name" tick={axisTick} />
                        <YAxis tick={axisTick} allowDecimals={false} />
                        <RTooltip contentStyle={{ background: theme.palette.background.paper, border: `1px solid ${theme.palette.divider}`, borderRadius: 8 }} />
                        <Legend />
                        <Bar dataKey="present" name="Có mặt" stackId="a" fill="#16a34a" />
                        <Bar dataKey="late" name="Đi muộn" stackId="a" fill="#ea580c" />
                        <Bar dataKey="absent" name="Vắng" stackId="a" fill="#dc2626" radius={[4, 4, 0, 0]} />
                      </BarChart>
                    </ResponsiveContainer>
                  )}
                </Box>
              </CardContent>
            </Card>

            {/* Pie */}
            <Card>
              <CardContent>
                <Typography variant="h6" mb={1}>Tỉ lệ tổng</Typography>
                <Box sx={{ height: 340 }}>
                  {loading ? <Skeleton variant="rounded" height={320} /> : (
                    <ResponsiveContainer width="100%" height="100%">
                      <PieChart>
                        <Pie data={pieData} dataKey="value" nameKey="name" cx="50%" cy="50%" innerRadius={60} outerRadius={100} paddingAngle={2}>
                          {pieData.map((_, i) => <Cell key={i} fill={PIE_COLORS[i]} />)}
                        </Pie>
                        <RTooltip contentStyle={{ background: theme.palette.background.paper, border: `1px solid ${theme.palette.divider}`, borderRadius: 8 }} />
                        <Legend />
                      </PieChart>
                    </ResponsiveContainer>
                  )}
                </Box>
              </CardContent>
            </Card>
          </Box>

          {/* Trend */}
          <Card sx={{ mb: 3 }}>
            <CardContent>
              <Typography variant="h6" mb={1}>Xu hướng điểm danh ({data?.from} → {data?.to})</Typography>
              <Box sx={{ height: 260 }}>
                {loading ? <Skeleton variant="rounded" height={240} /> : (
                  <ResponsiveContainer width="100%" height="100%">
                    <LineChart data={byDate} margin={{ top: 8, right: 16, bottom: 0, left: -12 }}>
                      <CartesianGrid strokeDasharray="3 3" stroke={grid} />
                      <XAxis dataKey="date" tick={axisTick} />
                      <YAxis tick={axisTick} allowDecimals={false} />
                      <RTooltip contentStyle={{ background: theme.palette.background.paper, border: `1px solid ${theme.palette.divider}`, borderRadius: 8 }} />
                      <Line type="monotone" dataKey="present" name="Lượt có mặt" stroke="#2563eb" strokeWidth={2} dot={{ r: 3 }} />
                    </LineChart>
                  </ResponsiveContainer>
                )}
              </Box>
            </CardContent>
          </Card>

          {/* Per-classroom table */}
          <Card>
            <CardContent>
              <Typography variant="h6" mb={2}>Chi tiết theo lớp</Typography>
              <TableContainer component={Paper} variant="outlined">
                <Table size="small">
                  <TableHead>
                    <TableRow>
                      <TableCell>Phòng</TableCell>
                      <TableCell>Môn</TableCell>
                      <TableCell align="right">Sĩ số</TableCell>
                      <TableCell align="right">Có mặt</TableCell>
                      <TableCell align="right">Đi muộn</TableCell>
                      <TableCell align="right">Vắng</TableCell>
                      <TableCell sx={{ minWidth: 160 }}>Tỉ lệ</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {byClassroom.map((r) => {
                      const pct = Math.round((r.rate || 0) * 100)
                      return (
                        <TableRow key={r.classroom_id} hover>
                          <TableCell>{r.classroom_name}</TableCell>
                          <TableCell>{r.subject}</TableCell>
                          <TableCell align="right">{r.enrolled}</TableCell>
                          <TableCell align="right"><Chip size="small" color="success" label={r.present} /></TableCell>
                          <TableCell align="right"><Chip size="small" color="warning" label={r.late || 0} /></TableCell>
                          <TableCell align="right"><Chip size="small" color="error" label={r.absent} /></TableCell>
                          <TableCell>
                            <Stack direction="row" alignItems="center" spacing={1}>
                              <LinearProgress variant="determinate" value={pct} color={pct >= 75 ? 'success' : pct >= 50 ? 'warning' : 'error'} sx={{ flex: 1, height: 8, borderRadius: 4 }} />
                              <Typography variant="caption" sx={{ minWidth: 34 }}>{pct}%</Typography>
                            </Stack>
                          </TableCell>
                        </TableRow>
                      )
                    })}
                  </TableBody>
                </Table>
              </TableContainer>
            </CardContent>
          </Card>
        </>
      )}
    </Box>
  )
}
