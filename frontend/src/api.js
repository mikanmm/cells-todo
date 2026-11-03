export class ApiError extends Error {
  constructor(status, message) {
    super(message)
    this.status = status
  }
}

async function request(method, path, body) {
  let res
  try {
    res = await fetch(`/api${path}`, {
      method,
      credentials: 'same-origin',
      headers: {
        'Content-Type': 'application/json',
        'X-Requested-With': 'cells',
      },
      body: body === undefined ? undefined : JSON.stringify(body),
    })
  } catch {
    throw new ApiError(0, 'Tidak dapat terhubung ke server. Periksa koneksi Anda.')
  }

  if (res.status === 204) return null

  let data = null
  try {
    data = await res.json()
  } catch {
    // respons bukan JSON
  }
  if (!res.ok) {
    throw new ApiError(res.status, data?.error || 'Terjadi kesalahan. Coba lagi.')
  }
  return data
}

export const api = {
  discordSettings: () => request('GET', '/notify/discord'),
  saveDiscord: (settings) => request('PUT', '/notify/discord', settings),
  previewDiscord: (settings) => request('POST', '/notify/discord/preview', settings),
  sendDiscord: (settings) => request('POST', '/notify/discord/send', settings),
  notifySchedule: () => request('GET', '/notify/schedule'),
  saveNotifySchedule: (schedule) => request('PUT', '/notify/schedule', schedule),
  previewNotify: (template) => request('POST', '/notify/preview', template),
  notifySettings: () => request('GET', '/notify'),
  saveNotifySettings: (recipients) => request('PUT', '/notify', { recipients }),
  testNotifyEmail: () => request('POST', '/notify/test'),
  broadcastEmail: (subject, body, mode = 'text', format = 'text') =>
    request('POST', '/notify/broadcast', { subject, body, mode, format }),
  categories: () => request('GET', '/categories'),
  addCategory: (name) => request('POST', '/categories', { name }),
  deleteCategory: (name) => request('DELETE', `/categories/${encodeURIComponent(name)}`),
  me: () => request('GET', '/me'),
  login: (username, password) => request('POST', '/login', { username, password }),
  logout: () => request('POST', '/logout'),
  listTodos: () => request('GET', '/todos'),
  createTodo: (todo) => request('POST', '/todos', todo),
  updateTodo: (id, todo) => request('PUT', `/todos/${id}`, todo),
  setDone: (id, done) => request('PATCH', `/todos/${id}`, { done }),
  deleteTodo: (id) => request('DELETE', `/todos/${id}`),
}
