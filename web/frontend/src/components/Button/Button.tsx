import type { ButtonHTMLAttributes } from 'react'
import styles from './Button.module.scss'

export type ButtonVariant = 'primary' | 'white' | 'black'

interface ButtonClassNameOptions {
  variant?: ButtonVariant
  full?: boolean
  small?: boolean
  className?: string
}

export function buttonClassName({ variant = 'primary', full = false, small = false, className }: ButtonClassNameOptions = {}) {
  return [styles.button, styles[variant], full && styles.full, small && styles.small, className].filter(Boolean).join(' ')
}

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant
  full?: boolean
  small?: boolean
}

export default function Button({ variant = 'primary', full = false, small = false, className, ...rest }: ButtonProps) {
  return <button className={buttonClassName({ variant, full, small, className })} {...rest} />
}
