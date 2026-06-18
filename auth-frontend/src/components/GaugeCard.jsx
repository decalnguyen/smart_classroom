import { Card, CardContent, Box, Typography, Stack, Tooltip } from '@mui/material'
import { useTheme } from '@mui/material/styles'

// thresholds = [{ lo, hi, label, color }] — sorted ranges covering [min, max].
// The active zone is highlighted; a legend row lists all zones.
export default function GaugeCard({ label, value, unit, min = 0, max = 100, color = '#2563eb', icon, danger = false, thresholds }) {
  const theme = useTheme()
  const hasValue = value !== null && value !== undefined && !Number.isNaN(Number(value))
  const v = hasValue ? Number(value) : 0
  const frac = Math.max(0, Math.min(1, (v - min) / (max - min || 1)))

  const cx = 100, cy = 100, r = 84
  const arcLen = Math.PI * r
  const track = theme.palette.mode === 'dark' ? 'rgba(148,163,184,0.18)' : 'rgba(15,23,42,0.08)'

  // Active threshold zone
  const activeZone = hasValue && thresholds
    ? thresholds.find((t) => v >= t.lo && v < t.hi) || thresholds[thresholds.length - 1]
    : null

  const arcColor = danger ? theme.palette.error.main : (activeZone?.color || color)

  const semicircle = `M ${cx - r} ${cy} A ${r} ${r} 0 0 1 ${cx + r} ${cy}`

  return (
    <Card sx={{ height: '100%' }}>
      <CardContent sx={{ pb: '12px !important' }}>
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

        {/* Active zone chip */}
        {activeZone && (
          <Box sx={{ mt: 0.75, textAlign: 'center' }}>
            <Typography
              variant="caption"
              fontWeight={700}
              sx={{
                px: 1.2,
                py: 0.3,
                borderRadius: 10,
                bgcolor: `${activeZone.color}22`,
                color: activeZone.color,
                border: `1px solid ${activeZone.color}55`,
                display: 'inline-block',
              }}
            >
              {activeZone.label}
            </Typography>
          </Box>
        )}

        {/* Threshold legend */}
        {thresholds && thresholds.length > 0 && (
          <Box sx={{ mt: 1, pt: 1, borderTop: `1px solid ${theme.palette.divider}` }}>
            <Typography variant="caption" color="text.disabled" sx={{ display: 'block', mb: 0.5 }}>
              Ngưỡng tham chiếu:
            </Typography>
            <Stack spacing={0.3}>
              {thresholds.map((t, i) => {
                const isActive = activeZone === t
                return (
                  <Stack key={i} direction="row" alignItems="center" spacing={0.75}>
                    <Box
                      sx={{
                        width: 8,
                        height: 8,
                        borderRadius: '50%',
                        bgcolor: t.color,
                        flexShrink: 0,
                        boxShadow: isActive ? `0 0 0 2px ${t.color}55` : 'none',
                      }}
                    />
                    <Typography
                      variant="caption"
                      sx={{
                        color: isActive ? t.color : 'text.secondary',
                        fontWeight: isActive ? 700 : 400,
                        lineHeight: 1.3,
                      }}
                    >
                      {t.lo}–{t.hi} {unit}: {t.label}
                    </Typography>
                  </Stack>
                )
              })}
            </Stack>
          </Box>
        )}
      </CardContent>
    </Card>
  )
}
