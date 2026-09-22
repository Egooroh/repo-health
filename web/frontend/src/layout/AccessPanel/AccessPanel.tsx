import type { ChangeEvent } from 'react'
import Button from '../../components/Button/Button'
import SmallLabel from '../../components/SmallLabel/SmallLabel'
import type { SessionScope, UseSessionResult } from '../../hooks/useSession'
import styles from './AccessPanel.module.scss'

interface AccessPanelProps {
  session: UseSessionResult
  onScopeChange: (scope: SessionScope) => void
  onConnectToken: () => void
}

export default function AccessPanel({ session, onScopeChange, onConnectToken }: AccessPanelProps) {
  const { connected, scope, disconnectPending, disconnect } = session

  function handleScopeChange(event: ChangeEvent<HTMLSelectElement>) {
    onScopeChange(event.target.value as SessionScope)
  }

  return (
    <div className={styles.panel} role="group" aria-label="Доступ к SourceCraft">
      <SmallLabel>ДОСТУП К SOURCECRAFT</SmallLabel>
      <p role="status">{connected ? 'Токен подключён · сессия на 1 час' : 'Токен не подключён'}</p>
      <Button variant="white" className={styles.connect} onClick={onConnectToken}>
        {connected ? 'Заменить токен' : 'Подключить токен'}
      </Button>
      {connected && (
        <Button variant="white" className={styles.connect} onClick={disconnect} disabled={disconnectPending}>
          Отключить и удалить данные
        </Button>
      )}
      <label htmlFor="access-scope">Репозитории</label>
      <select id="access-scope" value={scope} onChange={handleScopeChange}>
        <option value="public">Публичный каталог</option>
        <option value="personal" disabled={!connected}>
          Мои анализы
        </option>
      </select>
    </div>
  )
}
