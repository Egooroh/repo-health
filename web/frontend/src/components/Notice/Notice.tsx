import type { HTMLAttributes } from 'react'
import styles from './Notice.module.scss'

export type NoticeVariant = 'default' | 'amber' | 'error'

interface NoticeProps extends HTMLAttributes<HTMLDivElement> {
  variant?: NoticeVariant
}

export default function Notice({ variant = 'default', className, ...rest }: NoticeProps) {
  const classes = [styles.notice, variant !== 'default' && styles[variant], className].filter(Boolean).join(' ')
  return <div className={classes} {...rest} />
}
