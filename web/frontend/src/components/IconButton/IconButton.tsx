import type { ButtonHTMLAttributes } from 'react'
import styles from './IconButton.module.scss'

export default function IconButton({ className, type = 'button', ...rest }: ButtonHTMLAttributes<HTMLButtonElement>) {
  const classes = [styles.iconButton, className].filter(Boolean).join(' ')
  return <button type={type} className={classes} {...rest} />
}
