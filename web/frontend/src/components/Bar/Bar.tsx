import styles from './Bar.module.scss'

interface BarProps {
  value: number
  max?: number
  ariaLabel: string
}

export default function Bar({ value, max = 100, ariaLabel }: BarProps) {
  return <meter className={styles.bar} min={0} max={max} value={value} aria-label={ariaLabel} />
}
