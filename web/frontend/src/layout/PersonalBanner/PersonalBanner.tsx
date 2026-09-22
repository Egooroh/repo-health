import styles from './PersonalBanner.module.scss'

export default function PersonalBanner() {
  return (
    <div className={styles.banner} role="status">
      Личная сессия: результаты доступны только в этом браузере в течение часа. В общий рейтинг не попадают. AppSec пока не подключён.
    </div>
  )
}
