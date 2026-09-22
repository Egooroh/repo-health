import type { ReactNode } from 'react'
import styles from './PageHeading.module.scss'

interface PageHeadingProps {
  eyebrow: string
  title: ReactNode
  description?: ReactNode
  action?: ReactNode
}

export default function PageHeading({ eyebrow, title, description, action }: PageHeadingProps) {
  return (
    <div className={styles.heading}>
      <div>
        <span className={styles.eyebrow}>{eyebrow}</span>
        <h1>{title}</h1>
        {description && <p className={styles.description}>{description}</p>}
      </div>
      {action && <div className={styles.action}>{action}</div>}
    </div>
  )
}
