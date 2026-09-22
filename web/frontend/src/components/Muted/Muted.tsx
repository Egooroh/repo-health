import type { ElementType, HTMLAttributes } from 'react'
import styles from './Muted.module.scss'

interface MutedProps extends HTMLAttributes<HTMLElement> {
  as?: ElementType
}

export default function Muted({ as: Tag = 'span', className, ...rest }: MutedProps) {
  const classes = [styles.muted, className].filter(Boolean).join(' ')
  return <Tag className={classes} {...rest} />
}
