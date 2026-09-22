import type { ReactNode } from 'react'
import styles from './EmptyState.module.scss'

interface EmptyStateProps {
  title: ReactNode
  description: ReactNode
  action?: ReactNode
}

export default function EmptyState({ title, description, action }: EmptyStateProps) {
  return (
    <div className={styles.empty}>
      <h3>{title}</h3>
      <p>{description}</p>
      {action && <div className={styles.action}>{action}</div>}
    </div>
  )
}
