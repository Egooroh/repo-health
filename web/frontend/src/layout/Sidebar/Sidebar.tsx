import { NavLink } from 'react-router-dom'
import AccessPanel from '../AccessPanel/AccessPanel'
import DemoPanel from '../DemoPanel/DemoPanel'
import ThemeToggle from '../ThemeToggle/ThemeToggle'
import type { UseSessionResult, SessionScope } from '../../hooks/useSession'
import type { Theme } from '../../hooks/useTheme'
import styles from './Sidebar.module.scss'

const navLinkClassName = ({ isActive }: { isActive: boolean }) => (isActive ? styles.active : undefined)

interface SidebarProps {
    demoMode: boolean
    demoPending: boolean
    onToggleDemo: () => void
    session: UseSessionResult
    onScopeChange: (scope: SessionScope) => void
    onConnectToken: () => void
    theme: Theme
    onToggleTheme: () => void
}

export default function Sidebar({ demoMode, demoPending, onToggleDemo, session, onScopeChange, onConnectToken, theme, onToggleTheme }: SidebarProps) {
    return (
        <div className={styles.sidebar}>
            <NavLink className={styles.brand} to="/repositories" aria-label="Repo Health — главная">
        <span className={styles.mark} aria-hidden="true">
          <svg viewBox="0 0 32 32">
            <path d="M3 17h7l4-9 5 17 4-8h6" />
          </svg>
        </span>
                <span>
          repo<span className={styles.brandLight}>health</span>
          <small className={styles.tagline}>АНАЛИТИКА РЕПОЗИТОРИЕВ</small>
        </span>
            </NavLink>
            <ThemeToggle theme={theme} onToggle={onToggleTheme} />
            <div className={styles.workspaceLabel}>РАБОЧЕЕ ПРОСТРАНСТВО</div>
            <nav className={styles.nav} aria-label="Основная навигация">
                <NavLink to="/repositories" className={navLinkClassName}>
          <span className={styles.navIcon} aria-hidden="true">
            ▦
          </span>{' '}
                    Репозитории
                </NavLink>
                <NavLink to="/leaderboard" className={navLinkClassName}>
          <span className={styles.navIcon} aria-hidden="true">
            ↗
          </span>{' '}
                    Рейтинг
                </NavLink>
                <NavLink to="/methodology" className={navLinkClassName}>
          <span className={styles.navIcon} aria-hidden="true">
            ≡
          </span>{' '}
                    Как считается оценка
                </NavLink>
            </nav>
            <div className={styles.bottom}>
                <AccessPanel session={session} onScopeChange={onScopeChange} onConnectToken={onConnectToken} />
                <div className={styles.hideOnNarrow}>
                    <DemoPanel demoMode={demoMode} pending={demoPending} onToggle={onToggleDemo} />
                    <div className={styles.foot}>
                        <span className={styles.statusDot} /> Локальное приложение <span className={styles.version}>MVP</span>
                    </div>
                </div>
            </div>
        </div>
    )
}
