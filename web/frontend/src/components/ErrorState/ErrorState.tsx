import Button from '../Button/Button'
import styles from './ErrorState.module.scss'

interface ErrorStateProps {
  message: string
  onRetry: () => void
}

export default function ErrorState({ message, onRetry }: ErrorStateProps) {
  return (
    <div className={styles.error} role="alert">
      <h2>Не удалось загрузить данные</h2>
      <p>{message}</p>
      <Button variant="white" onClick={onRetry}>
        Повторить
      </Button>
    </div>
  )
}
