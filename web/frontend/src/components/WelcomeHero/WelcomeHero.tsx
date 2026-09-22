import Button from '../Button/Button'
import styles from './WelcomeHero.module.scss'

export default function WelcomeHero({ onOpenAnalysis }: { onOpenAnalysis: () => void }) {
  return (
    <div className={styles.hero}>
      <div className={styles.content}>
        <span className={styles.eyebrow}>ОТ ДАННЫХ К ДЕЙСТВИЯМ</span>
        <h2>
          Узнайте, что работает.
          <br />
          Поймите, что улучшить.
        </h2>
        <p>Добавьте репозиторий SourceCraft. Мы проверим документацию, CI и работу с задачами — и объясним каждый балл.</p>
        <Button variant="white" className={styles.cta} onClick={onOpenAnalysis}>
          Добавить первый репозиторий <span aria-hidden="true">→</span>
        </Button>
      </div>
      <div className={styles.visual} aria-hidden="true">
        <div className={styles.orbit}>
          <svg className={styles.pulse} viewBox="0 0 100 65">
            <path d="M0 34h24l12-21 15 40 13-28 9 9h27" />
          </svg>
          <span className={styles.label}>Факты → оценка → действия</span>
        </div>
      </div>
    </div>
  )
}
