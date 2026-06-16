import { useState, useEffect, useCallback } from 'react'
import {
  Box, Card, CardContent, Typography, Chip, Skeleton, Alert, Stack,
  Table, TableContainer, TableHead, TableBody, TableRow, TableCell, Paper,
  ToggleButtonGroup, ToggleButton,
} from '@mui/material'
import dayjs from 'dayjs'
import PageHeader from '../components/PageHeader'
import EmptyState from '../components/EmptyState'
import { auditApi, apiError } from '../api/client'

const ACTION_COLOR = { create: 'success', update: 'info', delete: 'error', approved: 'success', rejected: 'error' }
const ENTITY_LABEL = { attendance: 'Điểm danh', leave_request: 'Đơn nghỉ', holiday: 'Ngày lễ', makeup: 'Buổi bù', enrollment: 'Ghi danh' }

export default function Audit() {
  const [rows, setRows] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [entity, setEntity] = useState('')

  const load = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const { data } = await auditApi.list(entity ? { entity } : {})
      setRows(Array.isArray(data) ? data : [])
    } catch (err) {
      setError(apiError(err, 'Không tải được nhật ký.'))
    } finally {
      setLoading(false)
    }
  }, [entity])

  useEffect(() => { load() }, [load])

  return (
    <Box>
      <PageHeader
        title="Nhật ký hệ thống"
        subtitle="Lịch sử thao tác nhạy cảm: ai – làm gì – khi nào (audit log)"
        action={
          <ToggleButtonGroup size="small" exclusive value={entity} onChange={(_, v) => setEntity(v ?? '')}>
            <ToggleButton value="">Tất cả</ToggleButton>
            <ToggleButton value="attendance">Điểm danh</ToggleButton>
            <ToggleButton value="leave_request">Đơn nghỉ</ToggleButton>
            <ToggleButton value="enrollment">Ghi danh</ToggleButton>
          </ToggleButtonGroup>
        }
      />

      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

      <Card>
        <CardContent>
          {loading ? (
            <Stack spacing={1}>{Array.from({ length: 6 }).map((_, i) => <Skeleton key={i} variant="rounded" height={40} />)}</Stack>
          ) : rows.length === 0 ? (
            <EmptyState dense title="Chưa có nhật ký" description="Chưa ghi nhận thao tác nào." />
          ) : (
            <TableContainer component={Paper} variant="outlined" sx={{ maxHeight: 640 }}>
              <Table size="small" stickyHeader>
                <TableHead>
                  <TableRow>
                    <TableCell>Thời gian</TableCell>
                    <TableCell>Người thực hiện</TableCell>
                    <TableCell>Hành động</TableCell>
                    <TableCell>Đối tượng</TableCell>
                    <TableCell>Chi tiết</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {rows.map((r) => (
                    <TableRow key={r.id} hover>
                      <TableCell sx={{ whiteSpace: 'nowrap' }}>{dayjs(r.created_at).format('DD/MM HH:mm:ss')}</TableCell>
                      <TableCell>{r.actor_name || '—'} <Typography component="span" variant="caption" color="text.secondary">({r.actor_role})</Typography></TableCell>
                      <TableCell><Chip size="small" color={ACTION_COLOR[r.action] || 'default'} label={r.action} /></TableCell>
                      <TableCell>{ENTITY_LABEL[r.entity] || r.entity}</TableCell>
                      <TableCell>{r.detail}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          )}
        </CardContent>
      </Card>
    </Box>
  )
}
