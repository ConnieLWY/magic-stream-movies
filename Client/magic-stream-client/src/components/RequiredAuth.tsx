import { Navigate, Outlet, useLocation } from 'react-router-dom'
import useAuth from '../hook/useAuth'

function RequiredAuth() {
  const { auth, loading } = useAuth()
  const location = useLocation()

  // Wait for the session restore (refresh cookie -> access token) before deciding
  if (loading) {
    return null
  }

  // Not logged in: send to /login and remember where the user was going,
  // so Login can redirect back via location.state.from
  if (!auth) {
    return <Navigate to="/login" state={{ from: location }} replace />
  }

  return <Outlet />
}

export default RequiredAuth
