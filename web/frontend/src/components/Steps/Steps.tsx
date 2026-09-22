import styles from './Steps.module.scss'

const items = [
  { title: 'Добавьте репозиторий', text: 'Укажите адрес проекта; для приватного подключите токен и выберите «Мои анализы».' },
  { title: 'Изучите оценку', text: 'У каждого показателя есть источник и объяснение.' },
  { title: 'Выберите следующий шаг', text: 'Рекомендации связаны с обнаруженными фактами.' },
]

export default function Steps() {
  return (
    <div className={styles.steps}>
      {items.map((item, index) => (
        <div className={styles.step} key={item.title}>
          <span className={styles.stepNo}>{String(index + 1).padStart(2, '0')}</span>
          <h3>{item.title}</h3>
          <p>{item.text}</p>
        </div>
      ))}
    </div>
  )
}
