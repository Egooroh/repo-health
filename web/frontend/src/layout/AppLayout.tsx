import { useRef } from 'react'
import { Outlet, useLocation } from 'react-router-dom'
import { useDemoMode } from '../hooks/useDemoMode'
import { useSession } from '../hooks/useSession'
import { useTheme } from '../hooks/useTheme'
import AnalysisDialog from './AnalysisDialog/AnalysisDialog'
import styles from './AppLayout.module.scss'
import ModeBanner from './ModeBanner/ModeBanner'
import PageFooter from './PageFooter/PageFooter'
import PersonalBanner from './PersonalBanner/PersonalBanner'
import Sidebar from './Sidebar/Sidebar'
import TokenDialog from './TokenDialog/TokenDialog'
import Topbar from './Topbar/Topbar'

function breadcrumbFor(pathname: string): string {
  if (pathname.startsWith('/leaderboard')) return 'Рейтинг'
  if (pathname.startsWith('/methodology')) return 'Как считается оценка'
  if (pathname.startsWith('/repositories/')) return 'Аналитика репозитория'
  return 'Репозитории'
}

export default function AppLayout() {
  const location = useLocation()
  const analysisDialogRef = useRef<HTMLDialogElement>(null)
  const tokenDialogRef = useRef<HTMLDialogElement>(null)
  const { demoMode, pending: demoPending, toggleDemo, syncDemoMode } = useDemoMode()
  const session = useSession()
  const { theme, toggleTheme } = useTheme()

  // Connected + "Мои анализы" selected + not currently looking at demo data.
  const personalMode = session.connected && session.scope === 'personal' && !demoMode

  function openAnalysis() {
    // Starting a new analysis always means leaving the demo catalog.
    syncDemoMode(false)
    analysisDialogRef.current?.showModal()
  }

  function setScope(scope: 'public' | 'personal') {
    syncDemoMode(false)
    session.setScope(scope)
  }

  return (
    <>
      <a className={styles.skip} href="#main">
        Перейти к содержимому
      </a>
      <Sidebar
        demoMode={demoMode}
        demoPending={demoPending}
        onToggleDemo={toggleDemo}
        session={session}
        onScopeChange={setScope}
        onConnectToken={() => {
          session.resetConnectError()
          tokenDialogRef.current?.showModal()
        }}
        theme={theme}
        onToggleTheme={toggleTheme}
      />
      <div className={styles.shell}>
        <Topbar breadcrumb={breadcrumbFor(location.pathname)} onOpenAnalysis={openAnalysis} />
        {demoMode && <ModeBanner onExitDemo={toggleDemo} />}
        {personalMode && <PersonalBanner />}
        <main id="main" className={styles.main} tabIndex={-1}>
          <Outlet
            context={{
              demoMode,
              onOpenAnalysis: openAnalysis,
              onToggleDemo: toggleDemo,
              onSyncDemo: syncDemoMode,
              personalMode,
            }}
          />
        </main>
        <PageFooter />
      </div>
      <AnalysisDialog ref={analysisDialogRef} onClose={() => analysisDialogRef.current?.close()} onSyncDemo={syncDemoMode} personalMode={personalMode} />
      <TokenDialog ref={tokenDialogRef} onClose={() => tokenDialogRef.current?.close()} session={session} />
    </>
  )
}
