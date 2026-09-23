import { forwardRef, useEffect, useImperativeHandle, useRef, useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { ApiError } from '../../api/client'
import { createAnalysis } from '../../api/repositories'
import Button from '../../components/Button/Button'
import Dialog from '../../components/Dialog/Dialog'
import Muted from '../../components/Muted/Muted'
import { useToast } from '../../components/Toast/ToastProvider'
import styles from './AnalysisDialog.module.scss'

interface Props {
  onClose: () => void
  onSyncDemo: (demo: boolean) => void
  personalMode: boolean
}

const AnalysisDialog = forwardRef<HTMLDialogElement, Props>(({ onClose, onSyncDemo, personalMode }, ref) => {
  const dialogRef = useRef<HTMLDialogElement>(null)
  useImperativeHandle(ref, () => dialogRef.current as HTMLDialogElement)

  const navigate = useNavigate()
  const { showToast } = useToast()
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    const node = dialogRef.current
    if (!node) return
    // Also fires on ESC / backdrop close, not just the explicit close button.
    const reset = () => {
      setError(null)
      setSubmitting(false)
    }
    node.addEventListener('close', reset)
    return () => node.removeEventListener('close', reset)
  }, [])

  function handleClose() {
    setError(null)
    onClose()
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const form = event.currentTarget
    const ref = String(new FormData(form).get('ref') ?? '')
    setError(null)
    setSubmitting(true)
    try {
      const job = await createAnalysis(ref, false, personalMode)
      onSyncDemo(false)
      form.reset()
      onClose()
      navigate(`/repositories/${job.repository_id}`)
      showToast('Анализ поставлен в очередь. Результат появится на этой странице.')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Не удалось запустить анализ.')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Dialog
      ref={dialogRef}
      eyebrow="НОВЫЙ АНАЛИЗ"
      titleId="analysis-title"
      title={
        <>
          Посмотрим, как дела
          <br />у вашего репозитория
        </>
      }
      onClose={handleClose}
      onSubmit={handleSubmit}
    >
      <Muted as="p">
        Укажите репозиторий SourceCraft. Для личного анализа подключите токен: можно использовать открытые и приватные репозитории, доступные ему.
      </Muted>
      <p className={styles.hint}>
        {personalMode
          ? 'Личный анализ: результат виден только в вашей сессии.'
          : 'Публичный анализ: результат попадёт в общий каталог. Для приватного репозитория выберите «Мои анализы».'}
      </p>
      <label htmlFor="repository-ref">Адрес или название репозитория</label>
      <input id="repository-ref" name="ref" placeholder="организация/репозиторий" required maxLength={560} autoComplete="off" spellCheck={false} />
      <p className={styles.hint}>Например: examples/self-hosted-worker или полный HTTPS-адрес.</p>
      <p className={styles.privacy}>Исходный код не запускается. Недоступные источники будут отмечены в отчёте.</p>
      {error && (
        <p id="form-error" className={styles.error} role="alert">
          {error}
        </p>
      )}
      <Button full type="submit" disabled={submitting}>
        {submitting ? (
          'Запускаем…'
        ) : (
          <>
            Запустить анализ
          </>
        )}
      </Button>
    </Dialog>
  )
})

export default AnalysisDialog
