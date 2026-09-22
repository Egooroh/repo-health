import Tag from '../Tag/Tag'
import type { Recommendation as RecommendationType } from '../../types'
import styles from './Recommendation.module.scss'

interface RecommendationProps {
  recommendation: RecommendationType
  index: number
}

export default function Recommendation({ recommendation, index }: RecommendationProps) {
  return (
    <div className={styles.recommendation}>
      <span className={styles.index}>{String(index + 1).padStart(2, '0')}</span>
      <div>
        <h3>{recommendation.title}</h3>
        <p>{recommendation.reason}</p>
        <p className={styles.action}>{recommendation.action}</p>
      </div>
      <Tag variant={recommendation.priority === 'high' ? 'amber' : 'default'}>
        {recommendation.priority === 'high' ? 'Высокий приоритет' : 'Средний приоритет'}
      </Tag>
    </div>
  )
}
