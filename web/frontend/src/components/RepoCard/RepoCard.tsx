import { Link } from 'react-router-dom'
import StatusTag from '../StatusTag/StatusTag'
import ScoreValue from '../ScoreValue/ScoreValue'
import Tag from '../Tag/Tag'
import TextLink from '../TextLink/TextLink'
import type { Project } from '../../types'
import { formatNumber, isPendingProject } from '../../utils/format'
import styles from './RepoCard.module.scss'

export default function RepoCard({ project }: { project: Project }) {
  const [owner, name] = project.ref.split('/')
  const report = project.report
  const description =
    report?.repository.description ||
    (isPendingProject(project) ? 'Собираем сведения о репозитории. Страница обновится автоматически.' : project.job?.error || 'Описание не указано.')

  return (
    <div className={styles.card}>
      <div className={styles.top}>
        <span className={styles.icon} aria-hidden="true">
          ⌘
        </span>
        <StatusTag project={project} />
      </div>
      <h3>
        <Link to={`/repositories/${project.id}`}>
          <span>{owner} /</span>
          {name}
        </Link>
      </h3>
      <p className={styles.description}>{description}</p>
      <div className={styles.scoreRow}>
        <div>
          <ScoreValue score={report?.score} as="div" className={styles.scoreText}>
            {formatNumber(report?.score)} <small>/ 100</small>
          </ScoreValue>
          <div className={styles.scoreCaption}>
            {report?.score == null ? 'Недостаточно данных' : report.coverage < 60 ? 'Предварительная оценка' : 'Оценка здоровья'}
          </div>
        </div>
        <div>
          <Tag>{report ? `${report.coverage}% данных` : 'Ожидает анализа'}</Tag>
        </div>
      </div>
      <div className={styles.footer}>
        <span>
          {report?.repository.language || 'Язык не указан'}
          {project.demo ? ' · Демо' : ''}
        </span>
        <TextLink to={`/repositories/${project.id}`}>Подробнее ↗</TextLink>
      </div>
    </div>
  )
}
