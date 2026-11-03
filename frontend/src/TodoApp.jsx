import { useEffect, useRef, useState } from 'react'
import { api } from './api.js'
import CompleteTaskDialog from './CompleteTaskDialog.jsx'
import CategorySettings from './CategorySettings.jsx'
import NotifySettings from './NotifyHub.jsx'
import TodoForm from './TodoForm.jsx'
import TodoItem from './TodoItem.jsx'
import { groupTasks } from './taskGroups.js'
import { ProgressGem } from './Gem.jsx'
import { dayDiff, todayLong } from './dates.js'
export default function TodoApp({ user, onLogout, onSessionExpired }) {
  const [todos, setTodos] = useState([]),
    [categories, setCategories] = useState([]),
    [error, setError] = useState(''),
    [editing, setEditing] = useState(null),
    [filter, setFilter] = useState('all'),
    [query, setQuery] = useState('')
  const [completing, setCompleting] = useState(null)
  const [category, setCategory] = useState('')
  const [menuOpen, setMenuOpen] = useState(false)
  const menuRef = useRef(null)
  useEffect(() => {
    function outside(e) {
      if (!menuRef.current?.contains(e.target)) setMenuOpen(false)
    }
    function escape(e) {
      if (e.key === 'Escape') setMenuOpen(false)
    }
    document.addEventListener('pointerdown', outside)
    document.addEventListener('keydown', escape)
    return () => {
      document.removeEventListener('pointerdown', outside)
      document.removeEventListener('keydown', escape)
    }
  }, [])
  const settings =
    window.location.pathname === '/setting' || window.location.pathname.startsWith('/setting/')
  const notify = window.location.pathname === '/setting/notify'
  function fail(e) {
    if (e.status === 401) onSessionExpired()
    else setError(e.message)
  }
  async function load() {
    try {
      const [t, c] = await Promise.all([api.listTodos(), api.categories()])
      setTodos(t)
      setCategories(c)
    } catch (e) {
      fail(e)
    }
  }
  useEffect(() => {
    load()
  }, [])
  const groups = groupTasks(todos, { category, filter, query })
  async function toggle(t) {
    if (!t.done) {
      setCompleting(t)
      return
    }
    try {
      const updated = await api.setDone(t.id, false)
      setTodos((list) => list.map((item) => (item.id === updated.id ? updated : item)))
    } catch (e) {
      fail(e)
    }
  }
  async function completeTask() {
    try {
      const updated = await api.setDone(completing.id, true)
      setTodos((list) => list.map((item) => (item.id === updated.id ? updated : item)))
      setCompleting(null)
    } catch (e) {
      if (e.status === 401) onSessionExpired()
      throw e
    }
  }
  return (
    <div className="shell">
      <header className="topbar">
        <div className="topbar-text">
          <div className="greeting-row">
            <h1 className="greeting">Halo, {user.displayName}</h1>
            <div className="settings-menu" ref={menuRef}>
              <button
                type="button"
                className="account-chevron"
                aria-label="Buka menu"
                aria-expanded={menuOpen}
                aria-controls="account-menu"
                onClick={() => setMenuOpen(!menuOpen)}
              >
                <span>Menu</span>
                <svg
                  width="18"
                  height="18"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  aria-hidden="true"
                >
                  <path d="m6 9 6 6 6-6" />
                </svg>
              </button>
              {menuOpen && (
                <nav id="account-menu" className="card settings-links" aria-label="Menu akun">
                  <a href="/setting" onClick={() => setMenuOpen(false)}>
                    Setting
                  </a>
                </nav>
              )}
            </div>
          </div>
          <p className="muted">{todayLong()}</p>
        </div>
        <button
          type="button"
          className="btn btn-ghost"
          onClick={settings ? () => window.location.assign('/dashboard') : onLogout}
        >
          {settings ? 'Home' : 'Keluar'}
        </button>
      </header>
      <aside className="side">
        {!settings && (
          <section className="card progress-card" aria-label="Progres">
            <ProgressGem done={todos.filter((t) => t.done).length} total={todos.length} />
            <p className="progress-text">
              {todos.length ? (
                <>
                  <strong>{todos.filter((t) => t.done).length}</strong> dari{' '}
                  <strong>{todos.length}</strong> tugas selesai
                </>
              ) : (
                'Belum ada tugas'
              )}
            </p>
            {todos.some((t) => !t.done && t.dueDate && dayDiff(t.dueDate) < 0) && (
              <p className="overdue-note">
                {todos.filter((t) => !t.done && t.dueDate && dayDiff(t.dueDate) < 0).length} tugas
                terlambat
              </p>
            )}
          </section>
        )}
        <nav className="card filter-card" aria-label="Filter tugas">
          {settings ? (
            <>
              <h2 className="side-heading settings-nav-title">Setting</h2>
              <ul className="filter-list">
                <li>
                  <a
                    className={`filter-btn settings-nav-link ${!notify ? 'on' : ''}`}
                    href="/setting"
                    aria-current={!notify ? 'page' : undefined}
                  >
                    List kategori
                  </a>
                </li>
                <li>
                  <a
                    className={`filter-btn settings-nav-link ${notify ? 'on' : ''}`}
                    href="/setting/notify"
                    aria-current={notify ? 'page' : undefined}
                  >
                    Notify
                  </a>
                </li>
              </ul>
            </>
          ) : (
            <>
              {/*<div className="priority-order"><span>Urutan prioritas</span><strong>Tinggi → Sedang → Rendah</strong></div>*/}
              <ul className="filter-list">
                {['all', 'active', 'done'].map((v) => (
                  <li key={v}>
                    <button
                      type="button"
                      className={`filter-btn ${filter === v ? 'on' : ''}`}
                      aria-pressed={filter === v}
                      onClick={() => setFilter(v)}
                    >
                      <span>{{ all: 'Semua', active: 'Belum selesai', done: 'Selesai' }[v]}</span>
                      <span className="count">
                        {todos.filter((t) => (v === 'done' ? t.done : !t.done)).length}
                      </span>
                    </button>
                  </li>
                ))}
              </ul>
              {categories.length > 0 && (
                <div className="cat-block">
                  <h2 className="side-heading">Kategori</h2>
                  <div className="cat-list">
                    {categories.map((c) => (
                      <button
                        key={c}
                        type="button"
                        className={`chip chip-btn ${category === c ? 'on' : ''}`}
                        aria-pressed={category === c}
                        onClick={() => setCategory(category === c ? '' : c)}
                      >
                        {c}
                      </button>
                    ))}
                  </div>
                </div>
              )}
            </>
          )}
        </nav>
      </aside>
      <main className="content">
        {error && <p role="alert">{error}</p>}
        {settings ? (
          notify ? (
            <NotifySettings onSessionExpired={onSessionExpired} />
          ) : (
            <CategorySettings
              categories={categories}
              onChange={load}
              onSessionExpired={onSessionExpired}
            />
          )
        ) : (
          <>
            <TodoForm
              composer
              categories={categories}
              submitLabel="Tambah"
              onSubmit={async (v) => {
                await api.createTodo(v)
                await load()
              }}
            />
            <div className="search">
              <input
                aria-label="Cari tugas"
                placeholder="Cari tugas..."
                value={query}
                onChange={(e) => setQuery(e.target.value)}
              />
            </div>
            {groups.map((group) => (
              <section key={group.key} className="group" aria-labelledby={`priority-${group.key}`}>
                <h2
                  id={`priority-${group.key}`}
                  className={`priority-banner priority-banner-${group.key}`}
                >
                  Prioritas: {group.label}
                  <span className="group-count">{group.items.length}</span>
                </h2>
                <ul className="task-list">
                  {group.items.map((t) =>
                    editing === t.id ? (
                      <li key={t.id}>
                        <TodoForm
                          initial={{ ...t, dueDate: t.dueDate ?? '' }}
                          categories={categories}
                          submitLabel="Simpan perubahan"
                          onCancel={() => setEditing(null)}
                          onSubmit={async (v) => {
                            await api.updateTodo(t.id, v)
                            setEditing(null)
                            await load()
                          }}
                        />
                      </li>
                    ) : (
                      <TodoItem
                        key={t.id}
                        todo={t}
                        onToggle={toggle}
                        onEdit={(t) => setEditing(t.id)}
                        onDelete={async (t) => {
                          try {
                            await api.deleteTodo(t.id)
                            await load()
                          } catch (e) {
                            fail(e)
                          }
                        }}
                      />
                    ),
                  )}
                </ul>
              </section>
            ))}
            {!todos.length && <p>Belum ada tugas.</p>}
            {todos.length > 0 && groups.length === 0 && <p>Tidak ada tugas yang cocok.</p>}
          </>
        )}
      </main>
      {completing && (
        <CompleteTaskDialog
          todo={completing}
          onConfirm={completeTask}
          onCancel={() => setCompleting(null)}
        />
      )}
    </div>
  )
}
