import Button from '../../components/Button/Button'
import SmallLabel from '../../components/SmallLabel/SmallLabel'
import styles from './DemoPanel.module.scss'

interface DemoPanelProps {
  demoMode: boolean
  pending: boolean
  onToggle: () => void
}

export default function DemoPanel({ demoMode, pending, onToggle }: DemoPanelProps) {
  return (
    <div className={styles.panel}>
      <SmallLabel>ПОЗНАКОМИТЬСЯ С СЕРВИСОМ</SmallLabel>
      <p>Три примера. Разное состояние. Понятные причины.</p>
      <Button variant="white" full className={styles.toggle} onClick={onToggle} disabled={pending}>
        {demoMode ? 'Реальные репозитории' : pending ? 'Загружаем демо…' : 'Посмотреть демо'} <span aria-hidden="true">↗</span>
      </Button>
    </div>
  )
}
