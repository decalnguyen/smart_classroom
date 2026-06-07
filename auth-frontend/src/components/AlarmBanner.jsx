import { Alert, Collapse, Box } from '@mui/material'
import WarningAmberIcon from '@mui/icons-material/WarningAmber'
import { useRealtime } from '../context/RealtimeContext'

// Full-width red banner shown when a danger alert arrives over the realtime
// channel. Pulses to draw attention; dismissable.
export default function AlarmBanner() {
  const { activeAlert, dismissAlert } = useRealtime()

  return (
    <Collapse in={!!activeAlert} unmountOnExit>
      <Box sx={{ mb: 2 }}>
        <Alert
          severity="error"
          variant="filled"
          icon={<WarningAmberIcon />}
          onClose={dismissAlert}
          sx={{
            alignItems: 'center',
            fontWeight: 700,
            animation: 'pulseAlarm 1s ease-in-out infinite',
            '@keyframes pulseAlarm': {
              '0%, 100%': { opacity: 1 },
              '50%': { opacity: 0.78 },
            },
          }}
        >
          CẢNH BÁO AN TOÀN — {activeAlert?.message}
        </Alert>
      </Box>
    </Collapse>
  )
}
