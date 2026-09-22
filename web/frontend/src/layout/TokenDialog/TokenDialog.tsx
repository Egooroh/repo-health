import { forwardRef, useEffect, useImperativeHandle, useRef, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import Button from '../../components/Button/Button'
import Dialog from '../../components/Dialog/Dialog'
import Muted from '../../components/Muted/Muted'
import type { UseSessionResult } from '../../hooks/useSession'
import styles from './TokenDialog.module.scss'

interface Props {
  onClose: () => void
  session: UseSessionResult
}

const TokenDialog = forwardRef<HTMLDialogElement, Props>(({ onClose, session }, ref) => {
  const dialogRef = useRef<HTMLDialogElement>(null)
  useImperativeHandle(ref, () => dialogRef.current as HTMLDialogElement)

  const navigate = useNavigate()
  const { connectPending, connectError, connect } = session

  useEffect(() => {
    const node = dialogRef.current
    if (!node) return
    // Also fires on ESC / backdrop close, not just the explicit close button.
    const reset = () => node.querySelector('form')?.reset()
    node.addEventListener('close', reset)
    return () => node.removeEventListener('close', reset)
  }, [])

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const form = event.currentTarget
    const input = form.elements.namedItem('token') as HTMLInputElement
    const token = input.value
    // The token goes only into the request body — clear the field right away.
    input.value = ''
    const ok = await connect(token)
    if (ok) {
      onClose()
      navigate('/repositories')
    }
  }

  return (
    <Dialog ref={dialogRef} wide eyebrow="ЛИЧНЫЙ ДОСТУП" titleId="token-title" title="Подключить SourceCraft" onClose={onClose} onSubmit={handleSubmit}>
      <Muted as="p">
        Используйте персональный токен с правами чтения нужных репозиториев. Он даёт доступ в пределах ваших прав, включая проекты, куда вас пригласили.
      </Muted>
      <label htmlFor="sourcecraft-token">Персональный токен (PAT)</label>
      <input id="sourcecraft-token" name="token" type="password" required maxLength={2048} autoComplete="off" autoCapitalize="none" spellCheck={false} />
      <p className={styles.hint}>
        <a href="https://sourcecraft.dev/portal/docs/ru/sourcecraft/security/pat" target="_blank" rel="noopener noreferrer">
          Как создать токен в SourceCraft ↗
        </a>
      </p>
      <p className={styles.privacy}>
        Токен и личные отчёты хранятся в памяти сервера до отключения, перезапуска или истечения часа. В браузере хранится только идентификатор сессии.
        AppSec ещё не подключён: токен сам по себе не добавляет проверку безопасности или подписку.
      </p>
      {connectError && (
        <p id="token-error" className={styles.error} role="alert">
          {connectError}
        </p>
      )}
      <Button full type="submit" disabled={connectPending}>
        {connectPending ? 'Проверяем…' : 'Проверить и подключить'}
      </Button>
    </Dialog>
  )
})

export default TokenDialog
