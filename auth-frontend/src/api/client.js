import axios from 'axios'

export const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8091'
export const WS_BASE_URL = import.meta.env.VITE_WS_BASE_URL || 'ws://localhost:8082'

const client = axios.create({
  baseURL: API_BASE_URL,
  headers: { 'Content-Type': 'application/json' },
})

// Attach the JWT (if present) to every request.
client.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

// On 401, clear the stale session so the app redirects to login.
client.interceptors.response.use(
  (res) => res,
  (err) => {
    if (err.response && err.response.status === 401) {
      localStorage.removeItem('token')
      localStorage.removeItem('user')
    }
    return Promise.reject(err)
  }
)

// ---- Auth ----
export const authApi = {
  login: (username, password) => client.post('/login', { username, password }),
  signup: (username, password, role) => client.post('/signup', { username, password, role }),
  logout: () => client.post('/logout'),
  me: () => client.get('/user'),
}

// ---- Dashboard stats ----
export const statsApi = {
  overview: () => client.get('/stats/overview'),
}

// ---- Attendance reports / analytics ----
export const reportApi = {
  attendance: (params) => client.get('/reports/attendance', { params }),
  exportCsv: async (params) => {
    const res = await client.get('/reports/attendance/export', { params, responseType: 'blob' })
    const url = URL.createObjectURL(res.data)
    const a = document.createElement('a')
    a.href = url
    a.download = `diem_danh_${params?.date || 'report'}.csv`
    document.body.appendChild(a)
    a.click()
    a.remove()
    URL.revokeObjectURL(url)
  },
}

// ---- Personal (student) ----
export const meApi = {
  attendance: (params) => client.get('/my/attendance', { params }),
}

// ---- Teacher ↔ classroom assignment (admin) ----
export const classroomTeacherApi = {
  list: () => client.get('/classroom-teachers'),
  assign: (classroom_id, teacher_id) => client.post('/classroom-teachers', { classroom_id, teacher_id }),
  remove: (classroom_id, teacher_id) => client.delete('/classroom-teachers', { params: { classroom_id, teacher_id } }),
}

// Normalize an axios error into a human message.
export function apiError(err, fallback = 'Đã xảy ra lỗi, vui lòng thử lại.') {
  return err?.response?.data?.error || err?.message || fallback
}

// ---- Sensors / devices ----
export const sensorApi = {
  history: (deviceId, start, end) =>
    client.get(`/sensor/${deviceId}`, { params: { start, end } }),
  listSensors: () => client.get('/sensorinf'),
  createSensor: (s) => client.post('/sensorinf', s),
  updateSensor: (deviceId, s) => client.put(`/sensorinf/${deviceId}`, s),
  deleteSensor: (deviceId) => client.delete(`/sensorinf/${deviceId}`),
  setDeviceMode: (deviceType, deviceId, mode) =>
    client.post(`/device/${deviceType}/${deviceId}/mode`, { mode }),
}

// ---- School entities ----
export const schoolApi = {
  getBuildings: () => client.get('/buildings'),
  createBuilding: (b) => client.post('/buildings', b),
  updateBuilding: (id, b) => client.put(`/buildings/${id}`, b),
  deleteBuilding: (id) => client.delete(`/buildings/${id}`),

  getClassrooms: () => client.get('/classrooms'),
  // Classrooms in the caller's scope (admin/student = all, teacher = assigned).
  getMyClassrooms: () => client.get('/my/classrooms'),
  createClassroom: (c) => client.post('/classrooms', c),
  updateClassroom: (id, c) => client.put(`/classrooms/${id}`, c),
  deleteClassroom: (id) => client.delete(`/classrooms/${id}`),

  getStudents: () => client.get('/students'),
  createStudent: (s) => client.post('/students', s),
  updateStudent: (id, s) => client.put(`/students/${id}`, s),
  deleteStudent: (id) => client.delete(`/students/${id}`),

  getTeachers: () => client.get('/teachers'),
  createTeacher: (t) => client.post('/teachers', t),
  updateTeacher: (id, t) => client.put(`/teachers/${id}`, t),
  deleteTeacher: (id) => client.delete(`/teachers/${id}`),
}

// ---- Attendance ----
export const attendanceApi = {
  list: (classroomId) => client.get('/attendance', { params: { classroom_id: classroomId } }),
  create: (a) => client.post('/attendance', a),
  update: (id, a) => client.put(`/attendance/${id}`, a),
  remove: (id) => client.delete(`/attendance/${id}`),
}

// ---- Schedule (per-account) ----
export const scheduleApi = {
  getWeekly: () => client.get('/schedules'),
  create: (s) => client.post('/schedules', s),
  update: (id, s) => client.put(`/schedules/${id}`, s),
  remove: (id) => client.delete(`/schedules/${id}`),
}

// ---- Notifications ----
export const notificationApi = {
  list: () => client.get('/notifications'),
  create: (n, accountId) =>
    client.post('/notifications', n, { params: accountId ? { account_id: accountId } : {} }),
  markRead: (id, n) => client.put(`/notifications/${id}`, n),
  remove: (id) => client.delete(`/notifications/${id}`),
}

export default client
