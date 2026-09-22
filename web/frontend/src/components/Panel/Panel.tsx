import type { HTMLAttributes, ReactNode } from 'react'
import styles from './Panel.module.scss'

interface PanelProps extends Omit<HTMLAttributes<HTMLDivElement>, 'title'> {
  title?: ReactNode
  spaced?: boolean
}

export default function Panel({ title, spaced = false, className, children, ...rest }: PanelProps) {
  const classes = [styles.panel, spaced && styles.spaced, className].filter(Boolean).join(' ')
  return (
    <div className={classes} {...rest}>
      {title && <h2 className={styles.title}>{title}</h2>}
      {children}
    </div>
  )
}
