import { useEffect } from 'react'
import { useParams } from 'react-router-dom'
import CategoryPanel from '../components/CategoryPanel/CategoryPanel'
import EmptyState from '../components/EmptyState/EmptyState'
import ErrorState from '../components/ErrorState/ErrorState'
import LoadingState from '../components/LoadingState/LoadingState'
import Muted from '../components/Muted/Muted'
import Notice from '../components/Notice/Notice'
import Panel from '../components/Panel/Panel'
import Recommendation from '../components/Recommendation/Recommendation'
import ScorePanel from '../components/ScorePanel/ScorePanel'
import SectionHeading from '../components/SectionHeading/SectionHeading'
import SmallLabel from '../components/SmallLabel/SmallLabel'
import SourcesGrid from '../components/SourcesGrid/SourcesGrid'
import Tag from '../components/Tag/Tag'
import TextLink from '../components/TextLink/TextLink'
import { useProject } from '../hooks/useProject'
import { useLayoutContext } from '../layout/context'
import { formatDate, isPendingProject } from '../utils/format'
import styles from './RepositoryDetailPage.module.scss'

export default function RepositoryDetailPage() {
  const { id } = useParams<{ id: string }>()
  const { onSyncDemo, personalMode } = useLayoutContext()
  const { project, loading, error, notFound, reload } = useProject(id, personalMode)
  const projectDemo = project?.demo
  useEffect(() => {
    if (projectDemo != null) onSyncDemo(projectDemo)
  }, [projectDemo, onSyncDemo])

  if (loading) return <LoadingState />
  if (error) return <ErrorState message={error} onRetry={reload} />
  if (notFound || !project) {
    return (
      <EmptyState
        title="Репозиторий не найден"
        description="Проверьте адрес или вернитесь к списку."
        action={<TextLink to="/repositories">К репозиториям</TextLink>}
      />
    )
  }

  const { report } = project
  const [owner, name] = project.ref.split('/')
  const pending = isPendingProject(project)

  const header = (
    <>
      <TextLink className={styles.backLink} to="/repositories">
        ← Все репозитории
      </TextLink>
      <div className={styles.heading}>
        <div>
          <span className={styles.eyebrow}>{project.demo ? 'ДЕМОНСТРАЦИОННЫЙ ПРИМЕР' : 'АНАЛИТИКА РЕПОЗИТОРИЯ'}</span>
          <h1>
            <span>{owner} /</span> {name}
          </h1>
          <p>{report?.repository.description || 'Результат анализа репозитория SourceCraft.'}</p>
          <div className={styles.submeta}>
            {report && (
              <>
                <Tag>{report.repository.language || 'Язык не указан'}</Tag>
                <Tag>{report.repository.default_branch || 'Ветка не указана'}</Tag>
                <Tag>{formatDate(report.analyzed_at)}</Tag>
              </>
            )}
            {report?.repository.fork && <Tag variant="amber">Форк</Tag>}
          </div>
        </div>
        {!project.demo && !personalMode && (
          <Muted as="p" className={styles.autoUpdateNote}>
            {pending ? 'Анализируем…' : 'Оценка обновляется автоматически раз в сутки.'}
          </Muted>
        )}
        {personalMode && pending && (
          <Muted as="p" className={styles.autoUpdateNote}>
            Анализируем…
          </Muted>
        )}
      </div>
    </>
  )

  if (!report) {
    return (
      <>
        {header}
        {pending ? (
          <div className={styles.loading} role="status">
            Проверяем документацию, CI и задачи…
          </div>
        ) : (
          <EmptyState title="Готового отчёта пока нет" description="Причина ошибки будет устранена при следующем автоматическом обновлении." />
        )}
      </>
    )
  }

  return (
    <>
      {header}
      {!report.eligible && <Notice variant="amber">{report.eligibility_reason} Репозиторий не включён в рейтинг.</Notice>}
      <div className={styles.detailGrid}>
        <ScorePanel report={report} />
        <Panel title="Из чего складывается оценка">
          {report.categories.map((category) => (
            <CategoryPanel key={category.title} category={category} />
          ))}
        </Panel>
      </div>
      <Panel spaced>
        <SectionHeading title="Что улучшить в первую очередь" count={report.recommendations.length} action={<SmallLabel>НА ОСНОВЕ ФАКТОВ</SmallLabel>} />
        {report.recommendations.length ? (
          report.recommendations.map((recommendation, index) => <Recommendation key={recommendation.title} recommendation={recommendation} index={index} />)
        ) : (
          <Notice>
            {report.score == null
              ? 'Пока не хватает проверенных данных для оценки. Откройте показатели и ограничения источников выше.'
              : 'По измеренным показателям рекомендаций нет. Неизвестные источники, включая AppSec, не позволяют сделать вывод о состоянии всего проекта.'}
          </Notice>
        )}
      </Panel>
      <Panel spaced title="Источники и ограничения">
        <SourcesGrid sources={report.sources} />
        <p className={styles.explainer}>
          Методика {report.version}. Наличие файла не означает высокое качество содержимого. Для форков и зрелых проектов интерпретируйте результат с
          учётом назначения. <TextLink to="/methodology">Как считается оценка →</TextLink>
        </p>
      </Panel>
    </>
  )
}
