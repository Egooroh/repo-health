import EmptyState from '../components/EmptyState/EmptyState'
import ErrorState from '../components/ErrorState/ErrorState'
import LeaderboardTable from '../components/LeaderboardTable/LeaderboardTable'
import LoadingState from '../components/LoadingState/LoadingState'
import Notice from '../components/Notice/Notice'
import PageHeading from '../components/PageHeading/PageHeading'
import SectionHeading from '../components/SectionHeading/SectionHeading'
import StatsGrid from '../components/StatsGrid/StatsGrid'
import Tag from '../components/Tag/Tag'
import { useProjects } from '../hooks/useProjects'
import { useLayoutContext } from '../layout/context'
import { formatNumber } from '../utils/format'

export default function LeaderboardPage() {
  const { demoMode } = useLayoutContext()
  const { items, loading, error, reload } = useProjects('leaderboard', demoMode, false)

  if (loading) return <LoadingState />
  if (error) return <ErrorState message={error} onRetry={reload} />

  const average = items.length ? items.reduce((sum, p) => sum + (p.report?.score ?? 0), 0) / items.length : null
  const recommendations = items.reduce((sum, p) => sum + (p.report?.recommendations.length || 0), 0)

  return (
    <>
      <PageHeading
        eyebrow="ОТКРЫТЫЕ ПРОЕКТЫ"
        title="Рейтинг репозиториев"
        description="Открытые проекты с актуальной оценкой и достаточными данными."
        action={<Tag variant={demoMode ? 'amber' : 'green'}>{demoMode ? 'Вымышленные примеры' : 'Открытые репозитории SourceCraft'}</Tag>}
      />
      <StatsGrid
        ariaLabel="Краткая статистика"
        items={[
          {
            label: (
              <>
                В рейтинге <span aria-hidden="true">▦</span>
              </>
            ),
            value: items.length,
            note: 'В этом экземпляре сервиса',
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
            note: `По ${items.length} рассчитанным отчётам`,
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
      <Notice>
        В рейтинг попадают проанализированные непустые репозитории с полнотой данных от 60% и оценкой не старше 7 дней. Сортировка: балл → полнота
        данных → название. Это рейтинг проектов, добавленных в этот сервис.
      </Notice>
      <SectionHeading title="Участники рейтинга" count={items.length} />
      {items.length ? (
        <LeaderboardTable items={items} />
      ) : (
        <EmptyState title="Рейтинг пока формируется" description="Добавьте репозиторий в выбранный раздел или посмотрите демонстрационные примеры." />
      )}
    </>
  )
}
