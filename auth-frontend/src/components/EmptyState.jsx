import { Box, Typography, Stack } from '@mui/material'
import InboxIcon from '@mui/icons-material/Inbox'

// Friendly empty/placeholder block with an icon, message and optional action.
export default function EmptyState({ icon, title, description, action, dense }) {
  return (
    <Stack alignItems="center" spacing={1.25} sx={{ py: dense ? 4 : 7, px: 2, color: 'text.secondary', textAlign: 'center' }}>
      <Box
        sx={{
          width: 64, height: 64, borderRadius: '50%',
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          bgcolor: 'action.hover', color: 'text.disabled',
          '& svg': { fontSize: 32 },
        }}
      >
        {icon || <InboxIcon />}
      </Box>
      <Typography variant="subtitle1" fontWeight={700} color="text.primary">
        {title}
      </Typography>
      {description && (
        <Typography variant="body2" sx={{ maxWidth: 420 }}>
          {description}
        </Typography>
      )}
      {action && <Box sx={{ mt: 1 }}>{action}</Box>}
    </Stack>
  )
}
