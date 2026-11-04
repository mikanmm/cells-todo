import { useEffect, useRef, useState } from 'react'
import { api } from './api.js'
import CompleteTaskDialog from './CompleteTaskDialog.jsx'
import CategorySettings from './CategorySettings.jsx'
import NotifySettings from './NotifyHub.jsx'
import TodoForm from './TodoForm.jsx'
import TodoItem from './TodoItem.jsx'
import { groupTasks, isOverdue } from './taskGroups.js'
import { ProgressGem } from './Gem.jsx'
import { todayStr, todayLong } from './dates.js'
export default function TodoApp({ user, onLogout, onSessionExpired }) {
  const [dark, setDark] = useState(() => document.documentElement.dataset.theme === 'dark')
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
  const [today, setToday] = useState(todayStr)
  useEffect(() => {
    const refreshDate = () => setToday(todayStr())
    const timer = window.setInterval(refreshDate, 30000)
    window.addEventListener('focus', refreshDate)
    return () => {
      window.clearInterval(timer)
      window.removeEventListener('focus', refreshDate)
    }
  }, [])
  const overdueCount = todos.filter(t => isOverdue(t, today)).length
  const completedCount = todos.filter(t => t.done).length
  const activeCount = todos.length - completedCount
  const counts = { all: activeCount, active: activeCount, done: completedCount, overdue: overdueCount }
  useEffect(() => {
    if (!overdueCount && filter === 'overdue') setFilter('all')
  }, [overdueCount, filter])
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
            <h1 className="greeting">Hello, {user.displayName}</h1>
            <div className="settings-menu" ref={menuRef}>
              <button
                type="button"
                className="account-chevron"
                aria-label="Open menu"
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
                <nav id="account-menu" className="card settings-links" aria-label="Account menu">
                  <a href="/setting" onClick={() => setMenuOpen(false)}>
                    Settings
                  </a>
                </nav>
              )}
            </div>
          </div>
          <p className="muted">{todayLong()}</p>
        </div>
        <div className="topbar-actions">
          <label className="notify-toggle">
            <input type="checkbox" role="switch" checked={dark} onChange={e => {
              const enabled = e.target.checked
              setDark(enabled)
              document.documentElement.dataset.theme = enabled ? 'dark' : 'light'
              try { localStorage.setItem('cells-dark', String(enabled)) } catch {}
            }} />
            <span className="toggle-track" aria-hidden="true" />
            Dark mode
          </label>
        <button
          type="button"
          className="btn btn-ghost"
          onClick={settings ? () => window.location.assign('/dashboard') : onLogout}
        >
          {settings ? 'Home' : 'Sign out'}
        </button>
        </div>
      </header>
      <aside className="side">
        {!settings && (
          <section className="card progress-card" aria-label="Progress">
            <ProgressGem done={todos.filter((t) => t.done).length} total={todos.length} />
            <p className="progress-text">
              {todos.length ? (
                <>
                  <strong>{todos.filter((t) => t.done).length}</strong> of{' '}
                  <strong>{todos.length}</strong> tasks completed
                </>
              ) : (
                'No tasks yet'
              )}
            </p>
            {overdueCount > 0 && (
              <p className="overdue-note">
                {overdueCount} overdue
                tasks
              </p>
            )}
          </section>
        )}
        <nav className="card filter-card" aria-label="Task filters">
          {settings ? (
            <>
              <h2 className="side-heading settings-nav-title">Settings</h2>
              <ul className="filter-list">
                <li>
                  <a
                    className={`filter-btn settings-nav-link ${!notify ? 'on' : ''}`}
                    href="/setting"
                    aria-current={!notify ? 'page' : undefined}
                  >
                    Categories
                  </a>
                </li>
                <li>
                  <a
                    className={`filter-btn settings-nav-link ${notify ? 'on' : ''}`}
                    href="/setting/notify"
                    aria-current={notify ? 'page' : undefined}
                  >
                    Notifications
                  </a>
                </li>
              </ul>
            </>
          ) : (
            <>
              {/*<div className="priority-order"><span>Priority order</span><strong>High → Medium → Low</strong></div>*/}
              <ul className="filter-list">
                {Object.keys(counts).filter(v => v !== 'overdue' || overdueCount > 0).map((v) => (
                  <li key={v}>
                    <button
                      type="button"
                      className={`filter-btn ${filter === v ? 'on' : ''}`}
                      aria-pressed={filter === v}
                      onClick={() => setFilter(v)}
                    >
                      <span>
                        {{ all: 'All tasks', active: 'Incomplete', done: 'Completed', overdue: 'Overdue' }[v]}
                      </span>
                      <span className="count">
                        {counts[v]}
                      </span>
                    </button>
                  </li>
                ))}
              </ul>
              {categories.length > 0 && (
                <div className="cat-block">
                  <h2 className="side-heading">Categories</h2>
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
              submitLabel="Add"
              onSubmit={async (v) => {
                await api.createTodo(v)
                await load()
              }}
            />
            <div className="search">
              <input
                aria-label="Search tasks"
                placeholder="Search tasks..."
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
                  Priority: {group.label}
                  <span className="group-count">{group.items.length}</span>
                </h2>
                <ul className="task-list">
                  {group.items.map((t) =>
                    editing === t.id ? (
                      <li key={t.id}>
                        <TodoForm
                          initial={{ ...t, dueDate: t.dueDate ?? '' }}
                          categories={categories}
                          submitLabel="Save changes"
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
            {!todos.length && <p>No tasks yet.</p>}
            {todos.length > 0 && groups.length === 0 && <p>No matching tasks.</p>}
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
