import { useState } from 'react'
import Button from '../components/Button/Button'
import EmptyState from '../components/EmptyState/EmptyState'
import ErrorState from '../components/ErrorState/ErrorState'
import LoadingState from '../components/LoadingState/LoadingState'
import PageHeading from '../components/PageHeading/PageHeading'
import RepoCard from '../components/RepoCard/RepoCard'
import SearchInput from '../components/SearchInput/SearchInput'
import SectionHeading from '../components/SectionHeading/SectionHeading'
import StatsGrid from '../components/StatsGrid/StatsGrid'
import Steps from '../components/Steps/Steps'
import Tag from '../components/Tag/Tag'
import WelcomeHero from '../components/WelcomeHero/WelcomeHero'
import { useProjects } from '../hooks/useProjects'
import { useLayoutContext } from '../layout/context'
import { formatNumber } from '../utils/format'
import styles from './RepositoriesPage.module.scss'

export default function RepositoriesPage() {
  const { demoMode, onOpenAnalysis, onToggleDemo, personalMode } = useLayoutContext()
  const [search, setSearch] = useState('')
  const { items, loading, error, reload } = useProjects('repositories', demoMode, personalMode)

  if (loading) return <LoadingState />
  if (error) return <ErrorState message={error} onRetry={reload} />

  const scored = items.filter((p) => p.report?.score != null)
  const average = scored.length ? scored.reduce((sum, p) => sum + (p.report?.score ?? 0), 0) / scored.length : null
  const recommendations = items.reduce((sum, p) => sum + (p.report?.recommendations.length || 0), 0)
  const filtered = items.filter((p) => `${p.ref} ${p.report?.repository.language || ''}`.toLowerCase().includes(search.toLowerCase()))

  return (
    <>
      <PageHeading
        eyebrow="ОБЗОР РАБОЧЕГО ПРОСТРАНСТВА"
        title="Ваши репозитории"
        description="Состояние проекта, причины оценки и следующий шаг — в одном месте."
        action={
          <Tag variant={demoMode ? 'amber' : 'green'}>
            {demoMode ? 'Вымышленные примеры' : personalMode ? 'Личные отчёты · без AppSec' : 'Открытые репозитории SourceCraft'}
          </Tag>
        }
      />
      <StatsGrid
        ariaLabel="Краткая статистика"
        items={[
          {
            label: (
              <>
                Всего репозиториев <span aria-hidden="true">▦</span>
              </>
            ),
            value: items.length,
            note: demoMode ? 'Демонстрационные данные' : personalMode ? 'Только в вашей сессии' : 'В этом экземпляре сервиса',
          },
          {
            label: (
              <>
                Средняя оценка <span aria-hidden="true">↗</span>
              </>
            ),
            value: (
              <>
                {formatNumber(average)} <small>/ 100</small>
              </>
            ),
            note: `По ${scored.length} рассчитанным отчётам`,
          },
          {
            label: (
              <>
                Рекомендации <span aria-hidden="true">≡</span>
              </>
            ),
            value: recommendations,
            note: 'Конкретные действия для улучшения',
          },
        ]}
      />
      {!items.length && <WelcomeHero onOpenAnalysis={onOpenAnalysis} />}
      <SectionHeading
        title="Все проекты"
        count={items.length}
        action={
          items.length > 0 && (
            <SearchInput
              aria-label="Поиск по названию или языку"
              placeholder="Поиск по названию или языку…"
              value={search}
              onChange={(event) => setSearch(event.target.value)}
            />
          )
        }
      />
      {filtered.length ? (
        <div className={styles.cards}>
          {filtered.map((project) => (
            <RepoCard key={project.id} project={project} />
          ))}
        </div>
      ) : (
        <>
          <EmptyState
            title={items.length ? 'Ничего не найдено' : 'Здесь появятся ваши проекты'}
            description={items.length ? 'Попробуйте другое название или язык.' : 'Добавьте репозиторий в выбранный раздел или посмотрите демонстрационные примеры.'}
            action={
              !items.length && (
                <Button variant="white" onClick={demoMode ? onOpenAnalysis : onToggleDemo}>
                  {demoMode ? 'Анализировать реальный репозиторий' : 'Посмотреть демо →'}
                </Button>
              )
            }
          />
          {!items.length && <Steps />}
        </>
      )}
    </>
  )
}
