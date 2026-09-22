import { NavLink } from 'react-router-dom'
import styles from './PageFooter.module.scss'

export default function PageFooter() {
  return (
    <div className={styles.footer}>
      <span>Repo Health · SourceCraft</span>
      <span>
        Каждая оценка — с объяснением. <NavLink to="/methodology">Методика rhs-2.0-rc1 ↗</NavLink>
      </span>
    </div>
  )
}
