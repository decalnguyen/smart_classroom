import { useState } from 'react'
import { useNavigate, useLocation, Navigate } from 'react-router-dom'
import {
  Box,
  Card,
  CardContent,
  TextField,
  Button,
  Typography,
  Tabs,
  Tab,
  Alert,
  MenuItem,
  InputAdornment,
  IconButton,
  Stack,
  Divider,
} from '@mui/material'
import Visibility from '@mui/icons-material/Visibility'
import VisibilityOff from '@mui/icons-material/VisibilityOff'
import SchoolIcon from '@mui/icons-material/School'
import { useAuth } from '../context/AuthContext'

export default function Login() {
  const { login, signup, isAuthenticated } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const from = location.state?.from?.pathname || '/'

  const [tab, setTab] = useState(0)
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [role, setRole] = useState('student')
  const [showPw, setShowPw] = useState(false)
  const [error, setError] = useState('')
  const [info, setInfo] = useState('')
  const [loading, setLoading] = useState(false)

  // Already logged in -> redirect away from the login screen.
  if (isAuthenticated) {
    return <Navigate to={from} replace />
  }

  const submit = async (e) => {
    e.preventDefault()
    setError('')
    setInfo('')
    setLoading(true)
    try {
      if (tab === 0) {
        await login(username, password)
        navigate(from, { replace: true })
      } else {
        await signup(username, password, role)
        setInfo('Đăng ký thành công! Vui lòng đăng nhập.')
        setTab(0)
      }
    } catch (err) {
      setError(err?.response?.data?.error || 'Có lỗi xảy ra, vui lòng thử lại.')
    } finally {
      setLoading(false)
    }
  }

  const fillDemo = (u) => {
    setUsername(u)
    setPassword(`${u}123`)
    setTab(0)
  }

  return (
    <Box
      sx={{
        minHeight: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        background: 'linear-gradient(135deg, #1976d2 0%, #00897b 100%)',
        p: 2,
      }}
    >
      <Card sx={{ width: '100%', maxWidth: 420 }}>
        <CardContent sx={{ p: 4 }}>
          <Stack alignItems="center" spacing={1} mb={2}>
            <SchoolIcon color="primary" sx={{ fontSize: 44 }} />
            <Typography variant="h5" fontWeight={800} textAlign="center">
              Lớp học thông minh
            </Typography>
            <Typography variant="body2" color="text.secondary" textAlign="center">
              Giám sát & điều khiển IoT
            </Typography>
          </Stack>

          <Tabs value={tab} onChange={(_, v) => { setTab(v); setError(''); setInfo('') }} variant="fullWidth" sx={{ mb: 2 }}>
            <Tab label="Đăng nhập" />
            <Tab label="Đăng ký" />
          </Tabs>

          {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
          {info && <Alert severity="success" sx={{ mb: 2 }}>{info}</Alert>}

          <form onSubmit={submit}>
            <Stack spacing={2}>
              <TextField
                label="Tên đăng nhập"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                fullWidth
                required
                autoFocus
              />
              <TextField
                label="Mật khẩu"
                type={showPw ? 'text' : 'password'}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                fullWidth
                required
                InputProps={{
                  endAdornment: (
                    <InputAdornment position="end">
                      <IconButton onClick={() => setShowPw((v) => !v)} edge="end">
                        {showPw ? <VisibilityOff /> : <Visibility />}
                      </IconButton>
                    </InputAdornment>
                  ),
                }}
              />
              {tab === 1 && (
                <TextField select label="Vai trò" value={role} onChange={(e) => setRole(e.target.value)} fullWidth>
                  <MenuItem value="student">Học sinh</MenuItem>
                  <MenuItem value="teacher">Giáo viên</MenuItem>
                  <MenuItem value="admin">Quản trị viên</MenuItem>
                </TextField>
              )}
              <Button type="submit" variant="contained" size="large" disabled={loading} fullWidth>
                {loading ? 'Đang xử lý...' : tab === 0 ? 'Đăng nhập' : 'Đăng ký'}
              </Button>
            </Stack>
          </form>

          <Divider sx={{ my: 2 }}>Tài khoản demo</Divider>
          <Stack direction="row" spacing={1} justifyContent="center">
            {['admin', 'teacher', 'student'].map((u) => (
              <Button key={u} size="small" variant="outlined" onClick={() => fillDemo(u)}>
                {u}
              </Button>
            ))}
          </Stack>
        </CardContent>
      </Card>
    </Box>
  )
}
