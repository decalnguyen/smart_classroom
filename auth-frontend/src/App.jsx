import { lazy, Suspense } from 'react'
import { Routes, Route } from 'react-router-dom'
import { Box, CircularProgress } from '@mui/material'
import ProtectedRoute from './components/ProtectedRoute'
import Layout from './components/Layout'
import { RealtimeProvider } from './context/RealtimeContext'

// Code-split each page so the initial bundle stays small.
const Login = lazy(() => import('./pages/Login'))
const Dashboard = lazy(() => import('./pages/Dashboard'))
const Sensors = lazy(() => import('./pages/Sensors'))
const Attendance = lazy(() => import('./pages/Attendance'))
const Schedule = lazy(() => import('./pages/Schedule'))
const Notifications = lazy(() => import('./pages/Notifications'))
const MyAttendance = lazy(() => import('./pages/MyAttendance'))
const Reports = lazy(() => import('./pages/Reports'))
const Leaves = lazy(() => import('./pages/Leaves'))
const Review = lazy(() => import('./pages/Review'))
const Enrollment = lazy(() => import('./pages/Enrollment'))
const Audit = lazy(() => import('./pages/Audit'))
const Admin = lazy(() => import('./pages/Admin'))
const NotFound = lazy(() => import('./pages/NotFound'))

function Loading() {
  return (
    <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '50vh' }}>
      <CircularProgress />
    </Box>
  )
}

export default function App() {
  return (
    <Suspense fallback={<Loading />}>
      <Routes>
        <Route path="/login" element={<Login />} />

        <Route
          element={
            <ProtectedRoute>
              <RealtimeProvider>
                <Layout />
              </RealtimeProvider>
            </ProtectedRoute>
          }
        >
          <Route path="/" element={<Dashboard />} />
          <Route path="/sensors" element={<Sensors />} />
          <Route
            path="/attendance"
            element={
              <ProtectedRoute roles={['admin', 'teacher']}>
                <Attendance />
              </ProtectedRoute>
            }
          />
          <Route path="/my-attendance" element={<MyAttendance />} />
          <Route path="/schedule" element={<Schedule />} />
          <Route path="/notifications" element={<Notifications />} />
          <Route path="/leaves" element={<Leaves />} />
          <Route
            path="/audit"
            element={
              <ProtectedRoute roles={['admin']}>
                <Audit />
              </ProtectedRoute>
            }
          />
          <Route
            path="/reports"
            element={
              <ProtectedRoute roles={['admin', 'teacher']}>
                <Reports />
              </ProtectedRoute>
            }
          />
          <Route
            path="/review"
            element={
              <ProtectedRoute roles={['admin', 'teacher']}>
                <Review />
              </ProtectedRoute>
            }
          />
          <Route
            path="/enrollment"
            element={
              <ProtectedRoute roles={['admin']}>
                <Enrollment />
              </ProtectedRoute>
            }
          />
          <Route
            path="/admin"
            element={
              <ProtectedRoute roles={['admin']}>
                <Admin />
              </ProtectedRoute>
            }
          />
          <Route path="*" element={<NotFound />} />
        </Route>
      </Routes>
    </Suspense>
  )
}
