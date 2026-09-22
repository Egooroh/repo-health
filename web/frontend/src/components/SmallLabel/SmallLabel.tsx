import type { HTMLAttributes } from 'react'
import styles from './SmallLabel.module.scss'

export default function SmallLabel({ className, ...rest }: HTMLAttributes<HTMLSpanElement>) {
  const classes = [styles.label, className].filter(Boolean).join(' ')
  return <span className={classes} {...rest} />
}
