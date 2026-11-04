try {
  document.documentElement.dataset.theme = localStorage.getItem('cells-dark') === 'true' ? 'dark' : 'light'
} catch {}
