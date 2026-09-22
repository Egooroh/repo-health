import type { ReactNode } from 'react'
import StatCard from '../StatCard/StatCard'
import styles from './StatsGrid.module.scss'

export interface StatItem {
  label: ReactNode
  value: ReactNode
  note: ReactNode
}

export default function StatsGrid({ items, ariaLabel }: { items: StatItem[]; ariaLabel: string }) {
  return (
    <div className={styles.grid} aria-label={ariaLabel} role="group">
      {items.map((item, index) => (
        <StatCard key={index} label={item.label} value={item.value} note={item.note} />
      ))}
    </div>
  )
}
