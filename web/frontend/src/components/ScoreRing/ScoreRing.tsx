import { formatNumber } from '../../utils/format'
import styles from './ScoreRing.module.scss'

const CIRCUMFERENCE = 440

export default function ScoreRing({ score }: { score: number | null }) {
  const offset = CIRCUMFERENCE - ((score ?? 0) / 100) * CIRCUMFERENCE
  const valueClass = score != null && score < 50 ? styles.red : score != null && score < 80 ? styles.amber : ''

  return (
    <div className={styles.ring}>
      <svg viewBox="0 0 160 160" aria-hidden="true">
        <circle className={styles.background} cx="80" cy="80" r="70" />
        <circle className={`${styles.value} ${valueClass}`} cx="80" cy="80" r="70" strokeDasharray={CIRCUMFERENCE} strokeDashoffset={offset} />
      </svg>
      <div className={styles.text}>
        <strong>{formatNumber(score)}</strong>
        <small>из 100 баллов</small>
      </div>
    </div>
  )
}
