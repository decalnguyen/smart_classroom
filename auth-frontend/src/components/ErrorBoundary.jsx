import { Component } from 'react'
import { Box, Typography, Button, Card, CardContent } from '@mui/material'
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline'

// Catches render-time errors so a single bad component doesn't blank the app.
export default class ErrorBoundary extends Component {
  constructor(props) {
    super(props)
    this.state = { error: null }
  }

  static getDerivedStateFromError(error) {
    return { error }
  }

  componentDidCatch(error, info) {
    console.error('UI error:', error, info)
  }

  render() {
    if (this.state.error) {
      return (
        <Box sx={{ minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center', p: 2 }}>
          <Card sx={{ maxWidth: 440 }}>
            <CardContent sx={{ textAlign: 'center', p: 4 }}>
              <ErrorOutlineIcon color="error" sx={{ fontSize: 48, mb: 1 }} />
              <Typography variant="h6" gutterBottom>
                Đã xảy ra lỗi giao diện
              </Typography>
              <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
                {String(this.state.error?.message || this.state.error)}
              </Typography>
              <Button variant="contained" onClick={() => window.location.reload()}>
                Tải lại trang
              </Button>
            </CardContent>
          </Card>
        </Box>
      )
    }
    return this.props.children
  }
}
