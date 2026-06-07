import { Box, Typography, Stack } from '@mui/material'

// Consistent page title + subtitle + optional right-aligned action area.
export default function PageHeader({ title, subtitle, action }) {
  return (
    <Stack
      direction="row"
      alignItems={{ xs: 'flex-start', sm: 'center' }}
      justifyContent="space-between"
      flexWrap="wrap"
      gap={1.5}
      sx={{ mb: 3 }}
    >
      <Box>
        <Typography variant="h4">{title}</Typography>
        {subtitle && (
          <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }}>
            {subtitle}
          </Typography>
        )}
      </Box>
      {action && <Box>{action}</Box>}
    </Stack>
  )
}
