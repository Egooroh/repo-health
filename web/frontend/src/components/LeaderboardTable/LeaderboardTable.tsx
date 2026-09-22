import { Link } from 'react-router-dom'
import Muted from '../Muted/Muted'
import ScoreValue from '../ScoreValue/ScoreValue'
import TextLink from '../TextLink/TextLink'
import type { Project } from '../../types'
import { formatDate, formatNumber } from '../../utils/format'
import styles from './LeaderboardTable.module.scss'

export default function LeaderboardTable({ items }: { items: Project[] }) {
  return (
    <div className={styles.wrap}>
      <table className={styles.table}>
        <thead>
          <tr>
            <th>Место</th>
            <th>Репозиторий</th>
            <th>Оценка</th>
            <th>Полнота данных</th>
            <th>Последний анализ</th>
            <th>
              <Muted>Отчёт</Muted>
            </th>
          </tr>
        </thead>
        <tbody>
          {items.map((project, index) => (
            <tr key={project.id}>
              <td>
                <span className={`${styles.rank} ${index === 0 ? styles.rankFirst : ''}`}>{String(index + 1).padStart(2, '0')}</span>
              </td>
              <td>
                <Link className={styles.repoLink} to={`/repositories/${project.id}`}>
                  {project.ref}
                  <small>
                    {project.report?.repository.language || 'Язык не указан'}
                    {project.demo ? ' · Демо' : ''}
                  </small>
                </Link>
              </td>
              <td>
                <ScoreValue score={project.report?.score} as="strong">
                  {formatNumber(project.report?.score)}
                </ScoreValue>
                <Muted> / 100</Muted>
              </td>
              <td>{project.report?.coverage}%</td>
              <td>
                <Muted>{formatDate(project.report?.analyzed_at)}</Muted>
              </td>
              <td>
                <TextLink to={`/repositories/${project.id}`} aria-label={`Отчёт ${project.ref}`}>
                  Подробнее ↗
                </TextLink>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
