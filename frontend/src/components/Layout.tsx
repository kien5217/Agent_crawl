import { NavLink, Outlet } from 'react-router-dom'
import styles from './Layout.module.css'

const navItems = [
  { to: '/documents', label: 'Documents' },
  { to: '/topics', label: 'Topics' },
  { to: '/workflows', label: 'Workflows' },
]

export default function Layout() {
  return (
    <div className={styles.shell}>
      <aside className={styles.sidebar}>
        <h1 className={styles.brand}>Agent Crawl</h1>
        <nav>
          {navItems.map(({ to, label }) => (
            <NavLink
              key={to}
              to={to}
              className={({ isActive }) =>
                `${styles.navLink} ${isActive ? styles.active : ''}`
              }
            >
              {label}
            </NavLink>
          ))}
        </nav>
      </aside>
      <main className={styles.content}>
        <Outlet />
      </main>
    </div>
  )
}
