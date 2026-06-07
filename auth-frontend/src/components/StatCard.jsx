import { Card, CardActionArea, CardContent, Box, Typography, Stack } from '@mui/material'

// KPI card: icon, value, label, optional unit / sub / trend / accent color / onClick.
export default function StatCard({ icon, value, unit, label, sub, trend, color = 'primary.main', onClick }) {
  const inner = (
    <CardContent sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
      <Box
        sx={{
          width: 52, height: 52, borderRadius: 2.5, flexShrink: 0,
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          color: '#fff', bgcolor: color, '& svg': { fontSize: 26 },
        }}
      >
        {icon}
      </Box>
      <Box sx={{ minWidth: 0, flex: 1 }}>
        <Stack direction="row" alignItems="baseline" spacing={0.5}>
          <Typography variant="h5" fontWeight={800} noWrap>
            {value}
          </Typography>
          {unit && (
            <Typography component="span" variant="body2" color="text.secondary">
              {unit}
            </Typography>
          )}
        </Stack>
        <Typography variant="body2" color="text.secondary" noWrap>
          {label}
        </Typography>
        {sub && (
          <Typography variant="caption" color="text.disabled" noWrap display="block">
            {sub}
          </Typography>
        )}
      </Box>
      {trend && (
        <Typography variant="caption" sx={{ color, fontWeight: 700, whiteSpace: 'nowrap' }}>
          {trend}
        </Typography>
      )}
    </CardContent>
  )
  return (
    <Card sx={{ height: '100%' }}>
      {onClick ? <CardActionArea onClick={onClick} sx={{ height: '100%' }}>{inner}</CardActionArea> : inner}
    </Card>
  )
}
