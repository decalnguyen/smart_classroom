import { Card, CardContent, Box, Typography, Stack } from '@mui/material'
import { useTheme } from '@mui/material/styles'

// Semicircular gauge card for a single sensor metric.
// Props: label, value (number|null), unit, min, max, color, icon, danger(bool)
export default function GaugeCard({ label, value, unit, min = 0, max = 100, color = '#2563eb', icon, danger = false }) {
  const theme = useTheme()
  const hasValue = value !== null && value !== undefined && !Number.isNaN(Number(value))
  const v = hasValue ? Number(value) : 0
  const frac = Math.max(0, Math.min(1, (v - min) / (max - min || 1)))

  const cx = 100, cy = 100, r = 84
  const arcLen = Math.PI * r
  const track = theme.palette.mode === 'dark' ? 'rgba(148,163,184,0.18)' : 'rgba(15,23,42,0.08)'
  const arcColor = danger ? theme.palette.error.main : color

  const semicircle = `M ${cx - r} ${cy} A ${r} ${r} 0 0 1 ${cx + r} ${cy}`

  return (
    <Card sx={{ height: '100%' }}>
      <CardContent>
        <Stack direction="row" alignItems="center" spacing={1} mb={0.5}>
          {icon && <Box sx={{ color: arcColor, display: 'flex' }}>{icon}</Box>}
          <Typography variant="subtitle2" color="text.secondary">{label}</Typography>
        </Stack>
        <Box sx={{ position: 'relative', width: '100%' }}>
          <svg viewBox="0 0 200 116" width="100%" style={{ display: 'block' }}>
            <path d={semicircle} fill="none" stroke={track} strokeWidth="14" strokeLinecap="round" />
            <path
              d={semicircle}
              fill="none"
              stroke={arcColor}
              strokeWidth="14"
              strokeLinecap="round"
              strokeDasharray={arcLen}
              strokeDashoffset={arcLen * (1 - frac)}
              style={{ transition: 'stroke-dashoffset .6s ease, stroke .3s ease' }}
            />
          </svg>
          <Box sx={{ position: 'absolute', inset: 0, display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'flex-end', pb: 0.5 }}>
            <Typography variant="h4" fontWeight={800} sx={{ color: arcColor, lineHeight: 1 }}>
              {hasValue ? v : '--'}
            </Typography>
            <Typography variant="caption" color="text.secondary">{unit}</Typography>
          </Box>
        </Box>
        <Stack direction="row" justifyContent="space-between" sx={{ mt: -0.5 }}>
          <Typography variant="caption" color="text.disabled">{min}</Typography>
          <Typography variant="caption" color="text.disabled">{max}</Typography>
        </Stack>
      </CardContent>
    </Card>
  )
}
