import type { InputHTMLAttributes } from 'react'
import styles from './SearchInput.module.scss'

export default function SearchInput({ className, ...rest }: InputHTMLAttributes<HTMLInputElement>) {
  const classes = [styles.search, className].filter(Boolean).join(' ')
  return <input className={classes} {...rest} />
}
