import Bar from '../Bar/Bar'
import ScoreValue from '../ScoreValue/ScoreValue'
import type { Category } from '../../types'
import { formatNumber } from '../../utils/format'
import styles from './CategoryPanel.module.scss'

export default function CategoryPanel({ category }: { category: Category }) {
  return (
    <div className={styles.category}>
      <div className={styles.line}>
        <strong>
          {category.title} <small>· вес {category.weight}%</small>
        </strong>
        <ScoreValue score={category.score}>{category.score == null ? 'Нет данных' : `${formatNumber(category.score)} / 100`}</ScoreValue>
      </div>
      <Bar value={category.score ?? 0} ariaLabel={`${category.title}: ${category.score == null ? 'нет данных' : formatNumber(category.score)}`} />
      <details>
        <summary>Показатели и объяснение</summary>
        {category.metrics.map((metric) => (
          <div className={styles.metric} key={metric.title}>
            <strong>
              {metric.title} · {metric.score == null ? 'Неизвестно' : `${formatNumber(metric.score)} / 100`} · вес {metric.weight}%
            </strong>
            {metric.evidence}
          </div>
        ))}
      </details>
    </div>
  )
}
