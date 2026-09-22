import type { ReactNode } from 'react'
import styles from './StatCard.module.scss'

interface StatCardProps {
  label: ReactNode
  value: ReactNode
  note: ReactNode
}

export default function StatCard({ label, value, note }: StatCardProps) {
  return (
    <div className={styles.stat}>
      <div className={styles.label}>{label}</div>
      <div className={styles.value}>{value}</div>
      <div className={styles.note}>{note}</div>
    </div>
  )
}
