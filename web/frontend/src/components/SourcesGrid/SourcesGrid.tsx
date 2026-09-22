import Tag from '../Tag/Tag'
import type { Report } from '../../types'
import styles from './SourcesGrid.module.scss'

const sourceLabels: Record<string, string> = {
  files: 'Документы',
  revision: 'Ревизия основной ветки',
  ci: 'История CI',
  issues: 'Issues',
  pulls: 'Merge Requests',
  appsec: 'AppSec',
}

const sourceStatusLabels: Record<string, string> = {
  available: 'Доступно',
  unavailable: 'Недоступно',
  truncated: 'Неполные данные',
  not_connected: 'Не подключено',
}

export default function SourcesGrid({ sources }: { sources: Report['sources'] }) {
  return (
    <div className={styles.grid}>
      {Object.entries(sources).map(([key, source]) => (
        <div className={styles.source} key={key}>
          <strong>{sourceLabels[key] || key}</strong>
          <Tag variant={source.status === 'available' ? 'green' : 'amber'}>{sourceStatusLabels[source.status] || 'Неизвестно'}</Tag>
          <p>{source.detail}</p>
        </div>
      ))}
    </div>
  )
}
