export const PRIORITY_GROUPS = [
  { key: 'high', label: 'Tinggi' },
  { key: 'medium', label: 'Sedang' },
  { key: 'low', label: 'Rendah' },
]

export function groupTasks(todos, { category = '', filter = 'all', query = '' } = {}) {
  const search = query.trim().toLowerCase()
  const visible = todos.filter(
    (todo) =>
      (!category || todo.category === category) &&
      (filter === 'done' ? todo.done : !todo.done) &&
      `${todo.title} ${todo.notes}`.toLowerCase().includes(search),
  )
  return PRIORITY_GROUPS.map((group) => ({
    ...group,
    items: visible.filter((todo) => todo.priority === group.key).sort((a, b) => b.id - a.id),
  })).filter((group) => group.items.length > 0)
}
