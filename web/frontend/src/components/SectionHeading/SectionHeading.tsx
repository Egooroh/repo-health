import type { ReactNode } from 'react'
import styles from './SectionHeading.module.scss'

interface SectionHeadingProps {
  title: ReactNode
  count: number
  action?: ReactNode
}

export default function SectionHeading({ title, count, action }: SectionHeadingProps) {
  return (
    <div className={styles.heading}>
      <h2>
        {title} <span className={styles.count}>{count}</span>
      </h2>
      {action}
    </div>
  )
}
