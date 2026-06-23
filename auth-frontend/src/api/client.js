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
  classroomsOverview: () => client.get('/classrooms/overview'),
  // Caller's classes TODAY up to now, split { ongoing, ended } with attendance.
  classesToday: () => client.get('/classes-today'),
}

// ---- Attendance reports / analytics ----
export const reportApi = {
  attendance: (params) => client.get('/reports/attendance', { params }),
  // Export the (role-scoped) attendance report. opts: {from, to, detail, format}.
  // format 'xlsx' = Excel workbook; otherwise CSV (UTF-8 BOM). detail = per-student.
  exportReport: async ({ from, to, detail = false, format = 'csv' } = {}) => {
    const params = { format }
    if (from) params.from = from
    if (to) params.to = to
    if (detail) params.detail = 1
    const res = await client.get('/reports/attendance/export', { params, responseType: 'blob' })
    const url = URL.createObjectURL(res.data)
    const a = document.createElement('a')
    a.href = url
    const ext = format === 'xlsx' ? 'xlsx' : 'csv'
    const range = to && to !== from ? `${from}_${to}` : from || 'report'
    a.download = `diem_danh_${range}${detail ? '_chitiet' : ''}.${ext}`
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

// ---- Leave requests ----
export const leaveApi = {
  list: (params) => client.get('/leaves', { params }),
  create: (payload) => client.post('/leaves', payload), // {date, reason, student_id?}
  review: (id, status) => client.put(`/leaves/${id}/review`, { status }),
}

// ---- Audit log (admin) ----
export const auditApi = {
  list: (params) => client.get('/audit', { params }),
}

// ---- Holidays (admin) ----
export const holidayApi = {
  list: () => client.get('/holidays'),
  create: (date, name) => client.post('/holidays', { date, name }),
  remove: (id) => client.delete(`/holidays/${id}`),
}

// ---- Makeup sessions / buổi bù (admin) ----
export const makeupApi = {
  list: () => client.get('/makeups'),
  create: (body) => client.post('/makeups', body), // {class_id, date, start_min, end_min, note}
  remove: (id) => client.delete(`/makeups/${id}`),
}

// ---- Classes & enrollment / ghi danh lớp (admin) ----
export const classApi = {
  listClasses: () => client.get('/classes'),
  getRoster: (classId) => client.get(`/classes/${classId}/students`),
  enrollStudent: (classId, student_id) => client.post(`/classes/${classId}/students`, { student_id }),
  unenrollStudent: (classId, studentId) => client.delete(`/classes/${classId}/students/${studentId}`),
}

// ---- Face-recognition review queue (staff) ----
export const reviewApi = {
  list: (status) => client.get('/review-queue', { params: status ? { status } : {} }),
  decide: (id, decision) => client.post(`/review-queue/${id}`, { decision }), // confirm | reject
}

// ---- Face enrollment (đăng ký khuôn mặt) ----
export const enrollmentApi = {
  status: (params) => client.get('/enrollment/status', { params }), // {classroom_id?, q?, only?}
  enrollPhoto: (studentId, blob) => {
    const fd = new FormData()
    fd.append('student_id', studentId)
    fd.append('image', blob, 'face.jpg')
    return client.post('/enrollment/face/photo', fd, { headers: { 'Content-Type': 'multipart/form-data' } })
  },
  remove: (studentId) => client.delete(`/enrollment/face/${studentId}`),
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
