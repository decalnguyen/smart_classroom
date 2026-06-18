import { useState, useEffect, useCallback, useMemo } from 'react'
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
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  FormControlLabel,
  Checkbox,
  Tooltip as MuiTooltip,
} from '@mui/material'
import FileDownloadIcon from '@mui/icons-material/FileDownload'
import { useTheme } from '@mui/material/styles'
import CheckCircleIcon from '@mui/icons-material/CheckCircle'
import CancelIcon from '@mui/icons-material/Cancel'
import AccessTimeIcon from '@mui/icons-material/AccessTime'
import EventBusyIcon from '@mui/icons-material/EventBusy'
import PercentIcon from '@mui/icons-material/Percent'
import GroupsIcon from '@mui/icons-material/Groups'
import AssessmentIcon from '@mui/icons-material/Assessment'
import {
  ResponsiveContainer, BarChart, Bar, ComposedChart,
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

// minutes-from-midnight → "HH:MM"
function hhmm(m) {
  if (m == null) return ''
  return `${String(Math.floor(m / 60)).padStart(2, '0')}:${String(m % 60).padStart(2, '0')}`
}

export default function Reports() {
  const theme = useTheme()
  const { role } = useAuth()
  const [from, setFrom] = useState(todayStr(-6))
  const [to, setTo] = useState(todayStr())
  const [data, setData] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const load = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const res = await reportApi.attendance({ from, to })
      setData(res.data)
    } catch (err) {
      setError(apiError(err, 'Không tải được báo cáo.'))
      setData(null)
    } finally {
      setLoading(false)
    }
  }, [from, to])

  useEffect(() => { load() }, [load])

  const [exporting, setExporting] = useState(false)
  const [exportDetail, setExportDetail] = useState(false)
  const [exportFormat, setExportFormat] = useState('csv')
  const handleExport = async () => {
    setExporting(true)
    try {
      // Export the selected Từ–Đến range (not just a single day).
      await reportApi.exportReport({ from, to, detail: exportDetail, format: exportFormat })
    } catch (err) {
      setError(apiError(err, 'Xuất báo cáo thất bại.'))
    } finally {
      setExporting(false)
    }
  }

  const grid = theme.palette.mode === 'dark' ? 'rgba(148,163,184,0.15)' : '#eef2f7'
  const axisTick = { fontSize: 11, fill: theme.palette.text.secondary }
  const totals = data?.totals || { present: 0, late: 0, excused: 0, absent: 0, enrolled: 0, rate: 0 }
  const ratePct = Math.round((totals.rate || 0) * 100)
  const byClassroom = data?.by_classroom || []
  const bySession = data?.by_session || []
  const byDate = data?.by_date || []

  // ---- Intra-day derivations (snapshot day), excluding the synthetic all-day class ----
  const sessions = useMemo(() => bySession.filter((s) => !s.all_day), [bySession])
  const rateColor = (pct) => (pct < 60 ? '#dc2626' : pct < 85 ? '#ea580c' : '#16a34a')

  // Room × period grid: cols ordered by real start time, cell = the session.
  const heatmap = useMemo(() => {
    const periodStart = {}
    sessions.forEach((s) => { if (periodStart[s.period] == null) periodStart[s.period] = s.start_min })
    const cols = [...new Set(sessions.map((s) => s.period))].sort((a, b) => (periodStart[a] ?? 0) - (periodStart[b] ?? 0))
    const rows = [...new Set(sessions.map((s) => s.classroom_name))].sort()
    const cell = {}
    sessions.forEach((s) => { cell[`${s.classroom_name}|${s.period}`] = s })
    return { cols, rows, cell, periodStart }
  }, [sessions])

  // Participation rate (%) + late count per period, ordered by real start time.
  const byPeriod = useMemo(() => {
    const acc = {}
    sessions.forEach((s) => {
      if (!acc[s.period]) acc[s.period] = { period: s.period, start_min: s.start_min, present: 0, late: 0, excused: 0, enrolled: 0 }
      const a = acc[s.period]
      a.present += s.present; a.late += s.late; a.excused += s.excused; a.enrolled += s.enrolled
    })
    return Object.values(acc).sort((a, b) => a.start_min - b.start_min).map((p) => {
      const denom = p.enrolled - p.excused
      return { label: `T${p.period} · ${hhmm(p.start_min)}`, ratePct: denom > 0 ? Math.round((100 * (p.present + p.late)) / denom) : 0, late: p.late }
    })
  }, [sessions])

  // Triage: sessions with no-shows, ongoing first then most-absent, top 8.
  const triage = useMemo(() => (
    sessions.filter((s) => s.absent > 0)
      .sort((a, b) => (a.ended === b.ended ? b.absent - a.absent : a.ended ? 1 : -1))
      .slice(0, 8)
      .map((s) => ({ label: `${hhmm(s.start_min)} ${s.classroom_name} · ${s.subject}`, absent: s.absent, excused: s.excused, ended: s.ended }))
  ), [sessions])

  const dateInputs = (
    <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1.5} alignItems="center" flexWrap="wrap" useFlexGap>
      <TextField size="small" type="date" label="Từ" value={from} onChange={(e) => setFrom(e.target.value)} InputLabelProps={{ shrink: true }} />
      <TextField size="small" type="date" label="Đến" value={to} onChange={(e) => setTo(e.target.value)} InputLabelProps={{ shrink: true }} />
      <FormControlLabel
        control={<Checkbox size="small" checked={exportDetail} onChange={(e) => setExportDetail(e.target.checked)} />}
        label="Chi tiết SV"
      />
      <FormControl size="small" sx={{ minWidth: 110 }}>
        <InputLabel>Định dạng</InputLabel>
        <Select label="Định dạng" value={exportFormat} onChange={(e) => setExportFormat(e.target.value)}>
          <MenuItem value="csv">CSV</MenuItem>
          <MenuItem value="xlsx">Excel</MenuItem>
        </Select>
      </FormControl>
      <Button variant="outlined" startIcon={<FileDownloadIcon />} onClick={handleExport} disabled={exporting}>
        {exporting ? 'Đang xuất...' : 'Xuất (Từ–Đến)'}
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

      {/* KPI — snapshot of the range end day */}
      {!loading && data && (
        <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
          Chỉ số tổng hợp cho ngày <strong>{data.snapshot_day || data.to}</strong> · biểu đồ xu hướng theo khoảng {data.from} → {data.to}
        </Typography>
      )}
      <Box sx={{ display: 'grid', gap: 2, gridTemplateColumns: { xs: '1fr 1fr', md: 'repeat(3, 1fr)', lg: 'repeat(6, 1fr)' }, mb: 3 }}>
        {loading ? (
          Array.from({ length: 6 }).map((_, i) => <Skeleton key={i} variant="rounded" height={92} />)
        ) : (
          <>
            <StatCard icon={<CheckCircleIcon />} value={totals.present} label="Có mặt" color="#16a34a" />
            <StatCard icon={<AccessTimeIcon />} value={totals.late || 0} label="Đi muộn" color="#ea580c" />
            <StatCard icon={<EventBusyIcon />} value={totals.excused || 0} label="Có phép" color="#0891b2" />
            <StatCard icon={<CancelIcon />} value={totals.absent} label="Vắng" color="#dc2626" />
            <StatCard icon={<GroupsIcon />} value={totals.enrolled} label="Tổng lượt" color="#2563eb" />
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
                <Typography variant="h6">Lượt điểm danh theo từng phòng (cộng dồn các tiết)</Typography>
                <Typography variant="caption" color="text.secondary" display="block" mb={1}>
                  Ngày {data?.snapshot_day || data?.to} · mỗi cột = tổng lượt trong phòng, cộng dồn các tiết đã diễn ra (có mặt + muộn + phép + vắng)
                </Typography>
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
                        <Bar dataKey="excused" name="Có phép" stackId="a" fill="#0891b2" />
                        <Bar dataKey="absent" name="Vắng" stackId="a" fill="#dc2626" radius={[4, 4, 0, 0]} />
                      </BarChart>
                    </ResponsiveContainer>
                  )}
                </Box>
              </CardContent>
            </Card>

            {/* Heatmap: room × period participation rate */}
            <Card>
              <CardContent>
                <Typography variant="h6">Bản đồ nhiệt: Phòng × Tiết</Typography>
                <Typography variant="caption" color="text.secondary" display="block" mb={1.5}>
                  Tỉ lệ tham gia mỗi tiết · đỏ &lt;60% · cam 60–85% · xanh &gt;85% · ô trống = không có tiết
                </Typography>
                {loading ? <Skeleton variant="rounded" height={300} /> : heatmap.rows.length === 0 ? (
                  <Typography variant="body2" color="text.secondary">Chưa có tiết nào trong ngày.</Typography>
                ) : (
                  <Box sx={{ overflowX: 'auto' }}>
                    <Box sx={{ display: 'grid', gridTemplateColumns: `96px repeat(${heatmap.cols.length}, minmax(40px, 1fr))`, gap: 0.5, minWidth: 96 + heatmap.cols.length * 44 }}>
                      <Box />
                      {heatmap.cols.map((p) => (
                        <Box key={p} sx={{ textAlign: 'center', lineHeight: 1.1, py: 0.5 }}>
                          <Typography variant="caption" fontWeight={700} display="block">T{p}</Typography>
                          <Typography variant="caption" color="text.secondary" sx={{ fontSize: 10 }}>{hhmm(heatmap.periodStart[p])}</Typography>
                        </Box>
                      ))}
                      {heatmap.rows.map((room) => (
                        <Box key={room} sx={{ display: 'contents' }}>
                          <Box sx={{ display: 'flex', alignItems: 'center', pr: 1 }}>
                            <Typography variant="caption" fontWeight={600} noWrap>{room}</Typography>
                          </Box>
                          {heatmap.cols.map((p) => {
                            const s = heatmap.cell[`${room}|${p}`]
                            if (!s) return <Box key={p} sx={{ height: 38, borderRadius: 1, bgcolor: 'action.hover' }} />
                            const pct = Math.round((s.rate || 0) * 100)
                            return (
                              <MuiTooltip key={p} arrow title={`${room} · T${p} ${s.subject} (${hhmm(s.start_min)}–${hhmm(s.end_min)}) — Sĩ số ${s.enrolled}: ✓${s.present} ⏱${s.late} 📋${s.excused} ✗${s.absent} · ${pct}%`}>
                                <Box sx={{ height: 38, borderRadius: 1, bgcolor: rateColor(pct), opacity: 0.88, display: 'flex', alignItems: 'center', justifyContent: 'center', cursor: 'default', '&:hover': { opacity: 1 } }}>
                                  <Typography variant="caption" fontWeight={700} sx={{ color: '#fff', fontSize: 11 }}>{pct}</Typography>
                                </Box>
                              </MuiTooltip>
                            )
                          })}
                        </Box>
                      ))}
                    </Box>
                  </Box>
                )}
              </CardContent>
            </Card>
          </Box>

          {/* Participation rate + late arrivals by period (time-of-day) */}
          <Card sx={{ mb: 3 }}>
            <CardContent>
              <Typography variant="h6">Tỉ lệ tham gia &amp; lượt đi muộn theo tiết</Typography>
              <Typography variant="caption" color="text.secondary" display="block" mb={1}>
                Diễn biến trong ngày {data?.snapshot_day || data?.to} — các tiết xếp theo giờ bắt đầu thực tế
              </Typography>
              <Box sx={{ height: 300 }}>
                {loading ? <Skeleton variant="rounded" height={280} /> : byPeriod.length === 0 ? (
                  <Typography variant="body2" color="text.secondary">Chưa có tiết nào trong ngày.</Typography>
                ) : (
                  <ResponsiveContainer width="100%" height="100%">
                    <ComposedChart data={byPeriod} margin={{ top: 8, right: 8, bottom: 0, left: -12 }}>
                      <CartesianGrid strokeDasharray="3 3" stroke={grid} />
                      <XAxis dataKey="label" tick={axisTick} />
                      <YAxis yAxisId="left" tick={axisTick} allowDecimals={false} />
                      <YAxis yAxisId="right" orientation="right" domain={[0, 100]} tick={axisTick} unit="%" />
                      <RTooltip contentStyle={{ background: theme.palette.background.paper, border: `1px solid ${theme.palette.divider}`, borderRadius: 8 }} />
                      <Legend />
                      <Bar yAxisId="left" dataKey="late" name="Lượt đi muộn" fill="#ea580c" radius={[4, 4, 0, 0]} barSize={28} />
                      <Line yAxisId="right" type="monotone" dataKey="ratePct" name="Tỉ lệ tham gia (%)" stroke="#16a34a" strokeWidth={2} dot={{ r: 3 }} />
                    </ComposedChart>
                  </ResponsiveContainer>
                )}
              </Box>
            </CardContent>
          </Card>

          {/* Sessions needing attention: most no-shows, ongoing first */}
          {triage.length > 0 && (
            <Card sx={{ mb: 3 }}>
              <CardContent>
                <Typography variant="h6">Phiên cần can thiệp — vắng nhiều nhất</Typography>
                <Typography variant="caption" color="text.secondary" display="block" mb={1}>
                  Ưu tiên các tiết đang diễn ra · vắng không phép (đỏ) tách khỏi có phép (xám)
                </Typography>
                <Box sx={{ height: Math.max(160, triage.length * 38) }}>
                  <ResponsiveContainer width="100%" height="100%">
                    <BarChart layout="vertical" data={triage} margin={{ top: 4, right: 16, bottom: 0, left: 8 }}>
                      <CartesianGrid strokeDasharray="3 3" stroke={grid} horizontal={false} />
                      <XAxis type="number" allowDecimals={false} tick={axisTick} />
                      <YAxis type="category" dataKey="label" width={210} tick={{ fontSize: 11, fill: theme.palette.text.secondary }} />
                      <RTooltip contentStyle={{ background: theme.palette.background.paper, border: `1px solid ${theme.palette.divider}`, borderRadius: 8 }} />
                      <Legend />
                      <Bar dataKey="absent" name="Vắng (không phép)" stackId="t" fill="#dc2626" radius={[0, 0, 0, 0]} />
                      <Bar dataKey="excused" name="Có phép" stackId="t" fill="#94a3b8" radius={[0, 4, 4, 0]} />
                    </BarChart>
                  </ResponsiveContainer>
                </Box>
              </CardContent>
            </Card>
          )}

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

          {/* Per-classroom table (summed across the room's sessions) */}
          <Card sx={{ mb: 3 }}>
            <CardContent>
              <Typography variant="h6">Tổng hợp theo phòng</Typography>
              <Typography variant="caption" color="text.secondary" display="block" mb={2}>
                Cộng dồn tất cả các tiết đã diễn ra trong phòng ngày {data?.snapshot_day || data?.to}
              </Typography>
              <TableContainer component={Paper} variant="outlined">
                <Table size="small">
                  <TableHead>
                    <TableRow>
                      <TableCell>Phòng</TableCell>
                      <TableCell align="right">Tổng lượt</TableCell>
                      <TableCell align="right">Có mặt</TableCell>
                      <TableCell align="right">Đi muộn</TableCell>
                      <TableCell align="right">Có phép</TableCell>
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
                          <TableCell align="right">{r.enrolled}</TableCell>
                          <TableCell align="right"><Chip size="small" color="success" label={r.present} /></TableCell>
                          <TableCell align="right"><Chip size="small" color="warning" label={r.late || 0} /></TableCell>
                          <TableCell align="right"><Chip size="small" color="info" label={r.excused || 0} /></TableCell>
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

          {/* Per-SESSION breakdown: each môn học / tiết as its own row with its own sĩ số */}
          <Card>
            <CardContent>
              <Typography variant="h6">Chi tiết theo tiết / môn học</Typography>
              <Typography variant="caption" color="text.secondary" display="block" mb={2}>
                Mỗi dòng là một tiết học (môn) diễn ra trong phòng — sĩ số riêng theo từng lớp · ngày {data?.snapshot_day || data?.to}
              </Typography>
              {bySession.length === 0 ? (
                <Typography variant="body2" color="text.secondary">Chưa có tiết nào diễn ra cho ngày này.</Typography>
              ) : (
                <TableContainer component={Paper} variant="outlined" sx={{ maxHeight: 520 }}>
                  <Table size="small" stickyHeader>
                    <TableHead>
                      <TableRow>
                        <TableCell>Phòng</TableCell>
                        <TableCell>Môn học</TableCell>
                        <TableCell align="right">Tiết</TableCell>
                        <TableCell align="right">Sĩ số</TableCell>
                        <TableCell align="right">Có mặt</TableCell>
                        <TableCell align="right">Đi muộn</TableCell>
                        <TableCell align="right">Có phép</TableCell>
                        <TableCell align="right">Vắng</TableCell>
                        <TableCell>Trạng thái</TableCell>
                        <TableCell sx={{ minWidth: 140 }}>Tỉ lệ</TableCell>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {bySession.map((s) => {
                        const pct = Math.round((s.rate || 0) * 100)
                        return (
                          <TableRow key={s.class_id} hover>
                            <TableCell>{s.classroom_name}</TableCell>
                            <TableCell>{s.subject}</TableCell>
                            <TableCell align="right">{s.period}</TableCell>
                            <TableCell align="right">{s.enrolled}</TableCell>
                            <TableCell align="right"><Chip size="small" color="success" label={s.present} /></TableCell>
                            <TableCell align="right"><Chip size="small" color="warning" label={s.late || 0} /></TableCell>
                            <TableCell align="right"><Chip size="small" color="info" label={s.excused || 0} /></TableCell>
                            <TableCell align="right"><Chip size="small" color="error" label={s.absent} /></TableCell>
                            <TableCell>
                              <Chip size="small" variant="outlined" color={s.ended ? 'default' : 'primary'} label={s.ended ? 'Đã kết thúc' : 'Đang diễn ra'} />
                            </TableCell>
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
              )}
            </CardContent>
          </Card>
        </>
      )}
    </Box>
  )
}
