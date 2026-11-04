import { todayStr } from './dates.js'

export function isOverdue(todo, today = todayStr()) {
  return !todo.done && !!todo.dueDate && todo.dueDate < today
}

export const PRIORITY_GROUPS = [
  { key: 'high', label: 'High' },
  { key: 'medium', label: 'Medium' },
  { key: 'low', label: 'Low' },
]

export function groupTasks(todos, { category = '', filter = 'all', query = '' } = {}) {
  const search = query.trim().toLowerCase()
  const visible = todos.filter(
    (todo) =>
      (!category || todo.category === category) &&
      (filter === 'overdue' ? isOverdue(todo) : todo.done === (filter === 'done')) &&
      `${todo.title} ${todo.notes}`.toLowerCase().includes(search),
  )
  return PRIORITY_GROUPS.map((group) => ({
    ...group,
    items: visible.filter((todo) => todo.priority === group.key).sort((a, b) => b.id - a.id),
  })).filter((group) => group.items.length > 0)
}
