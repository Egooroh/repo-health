import type { HTMLAttributes } from 'react'
import styles from './Tag.module.scss'

export type TagVariant = 'default' | 'green' | 'amber' | 'red'

interface TagProps extends HTMLAttributes<HTMLSpanElement> {
  variant?: TagVariant
}

export default function Tag({ variant = 'default', className, ...rest }: TagProps) {
  const classes = [styles.tag, variant !== 'default' && styles[variant], className].filter(Boolean).join(' ')
  return <span className={classes} {...rest} />
}
