import { forwardRef, type FormEvent, type ReactNode } from 'react'
import IconButton from '../IconButton/IconButton'
import styles from './Dialog.module.scss'

interface DialogProps {
  eyebrow: string
  titleId: string
  title: ReactNode
  onClose: () => void
  onSubmit: (event: FormEvent<HTMLFormElement>) => void
  wide?: boolean
  children: ReactNode
}

const Dialog = forwardRef<HTMLDialogElement, DialogProps>(({ eyebrow, titleId, title, onClose, onSubmit, wide = false, children }, ref) => {
  const classes = [styles.dialog, wide && styles.wide].filter(Boolean).join(' ')
  return (
    <dialog ref={ref} className={classes} aria-labelledby={titleId} onClick={(event) => event.target === event.currentTarget && onClose()}>
      <form onSubmit={onSubmit}>
        <div className={styles.top}>
          <span className={styles.eyebrow}>{eyebrow}</span>
          <IconButton aria-label="Закрыть" onClick={onClose}>
            ×
          </IconButton>
        </div>
        <h2 id={titleId}>{title}</h2>
        {children}
      </form>
    </dialog>
  )
})

export default Dialog
