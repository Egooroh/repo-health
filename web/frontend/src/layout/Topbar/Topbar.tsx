import Button from '../../components/Button/Button'
import styles from './Topbar.module.scss'

interface TopbarProps {
  breadcrumb: string
  onOpenAnalysis: () => void
}

export default function Topbar({ breadcrumb, onOpenAnalysis }: TopbarProps) {
  return (
    <div className={styles.topbar}>
      <div className={styles.breadcrumbs}>
        SourceCraft <span>/</span> <strong>{breadcrumb}</strong>
      </div>
      <Button className={styles.action} onClick={onOpenAnalysis}>
        <span aria-hidden="true">＋</span> Анализировать репозиторий
      </Button>
    </div>
  )
}
