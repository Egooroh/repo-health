import type { ElementType, ReactNode } from 'react'
import { scoreTone } from '../../utils/format'
import styles from './ScoreValue.module.scss'

interface ScoreValueProps {
  score: number | null | undefined
  children: ReactNode
  as?: ElementType
  className?: string
}

export default function ScoreValue({ score, children, as: Tag = 'span', className }: ScoreValueProps) {
  const tone = scoreTone(score)
  const classes = [tone && styles[tone], className].filter(Boolean).join(' ')
  return <Tag className={classes}>{children}</Tag>
}
