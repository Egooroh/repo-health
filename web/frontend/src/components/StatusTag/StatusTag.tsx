import PulseDot from '../PulseDot/PulseDot'
import Tag from '../Tag/Tag'
import type { Project } from '../../types'
import { isPendingProject } from '../../utils/format'

export default function StatusTag({ project }: { project: Project }) {
  if (isPendingProject(project)) {
    return (
      <Tag variant="green">
        <PulseDot />
        Анализируем
      </Tag>
    )
  }
  if (project.job?.status === 'failed') {
    return <Tag variant="red">Ошибка анализа</Tag>
  }
  if (project.report?.score == null) {
    return <Tag variant="default">Недостаточно данных</Tag>
  }
  return <Tag variant={project.report.score >= 80 ? 'green' : 'amber'}>{project.report.score >= 80 ? 'В порядке' : 'Есть рекомендации'}</Tag>
}
