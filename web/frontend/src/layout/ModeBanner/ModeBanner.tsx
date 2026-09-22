import styles from './ModeBanner.module.scss'

export default function ModeBanner({ onExitDemo }: { onExitDemo: () => void }) {
  return (
    <div className={styles.banner}>
      Демонстрационный режим: все репозитории и показатели вымышлены.
      <button onClick={onExitDemo}>Вернуться к реальным данным →</button>
    </div>
  )
}
