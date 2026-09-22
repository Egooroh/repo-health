import styles from './LoadingState.module.scss'

export default function LoadingState({ message = 'Загрузка…' }: { message?: string }) {
  return (
    <div className={styles.loading} role="status">
      {message}
    </div>
  )
}
