import { Box, Typography, Button, Stack } from '@mui/material'
import { useNavigate } from 'react-router-dom'
import SentimentDissatisfiedIcon from '@mui/icons-material/SentimentDissatisfied'

export default function NotFound() {
  const navigate = useNavigate()
  return (
    <Box sx={{ minHeight: '60vh', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
      <Stack alignItems="center" spacing={2}>
        <SentimentDissatisfiedIcon sx={{ fontSize: 72, color: 'text.disabled' }} />
        <Typography variant="h4">404</Typography>
        <Typography color="text.secondary">Không tìm thấy trang bạn yêu cầu.</Typography>
        <Button variant="contained" onClick={() => navigate('/')}>Về trang tổng quan</Button>
      </Stack>
    </Box>
  )
}
