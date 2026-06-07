import { Navigate, useLocation } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'

// Guards routes: requires authentication, and optionally a role in `roles`.
export default function ProtectedRoute({ children, roles }) {
  const { isAuthenticated, role } = useAuth()
  const location = useLocation()

  if (!isAuthenticated) {
    return <Navigate to="/login" replace state={{ from: location }} />
  }
  if (roles && roles.length > 0 && !roles.includes(role)) {
    return <Navigate to="/" replace />
  }
  return children
}
