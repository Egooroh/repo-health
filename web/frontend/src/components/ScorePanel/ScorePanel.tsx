import Bar from '../Bar/Bar'
import Panel from '../Panel/Panel'
import ScoreRing from '../ScoreRing/ScoreRing'
import Tag from '../Tag/Tag'
import type { Report } from '../../types'
import { scoreLabel } from '../../utils/format'
import styles from './ScorePanel.module.scss'

export default function ScorePanel({ report }: { report: Report }) {
  const title = report.score == null ? 'Недостаточно данных' : report.coverage < 60 ? 'Предварительная оценка' : 'Оценка здоровья'

  return (
    <Panel className={styles.panel} title={title}>
      <ScoreRing score={report.score} />
      <Tag variant={report.score == null ? 'default' : report.score >= 80 ? 'green' : 'amber'}>{scoreLabel(report.score)}</Tag>
      <p className={styles.explainer}>
        {report.score == null
          ? 'Численный балл появится, когда будет достаточно проверенных данных для новой методики. Причины указаны справа.'
          : 'Оценка по проверенным фактам. Каждый показатель можно раскрыть справа.'}
      </p>
      <div className={styles.coverage}>
        <div className={styles.coverageLine}>
          <span>Полнота данных</span>
          <strong>{report.coverage}%</strong>
        </div>
        <Bar value={report.coverage} ariaLabel={`Полнота данных ${report.coverage}%`} />
        <p>Доля весов с измеренными показателями. Не вероятность правильности оценки.</p>
      </div>
    </Panel>
  )
}
