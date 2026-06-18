// Reference thresholds for classroom sensors.
// Each entry: { lo (inclusive), hi (exclusive), label, color }

export const LIGHT_THRESHOLDS = [
  { lo: 0,   hi: 200,  label: 'Thiếu sáng',    color: '#64748b' },
  { lo: 200, hi: 750,  label: 'Bình thường',   color: '#16a34a' },
  { lo: 750, hi: 1001, label: 'Quá sáng',      color: '#f59e0b' },
]

export const TEMP_THRESHOLDS = [
  { lo: 0,  hi: 18, label: 'Quá lạnh',     color: '#0284c7' },
  { lo: 18, hi: 28, label: 'Bình thường',  color: '#16a34a' },
  { lo: 28, hi: 50, label: 'Nóng',         color: '#f59e0b' },
  { lo: 50, hi: 61, label: 'Nguy hiểm',    color: '#dc2626' },
]

export const HUMIDITY_THRESHOLDS = [
  { lo: 0,  hi: 30,  label: 'Quá khô',      color: '#f59e0b' },
  { lo: 30, hi: 70,  label: 'Bình thường',  color: '#16a34a' },
  { lo: 70, hi: 101, label: 'Độ ẩm cao',    color: '#0284c7' },
]

export const SMOKE_THRESHOLDS = [
  { lo: 0,   hi: 100, label: 'An toàn',      color: '#16a34a' },
  { lo: 100, hi: 300, label: 'Chú ý',        color: '#f59e0b' },
  { lo: 300, hi: 601, label: 'Nguy hiểm',    color: '#dc2626' },
]
