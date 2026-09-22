import { Link, type LinkProps } from 'react-router-dom'
import styles from './TextLink.module.scss'

export default function TextLink({ className, ...rest }: LinkProps) {
  const classes = [styles.link, className].filter(Boolean).join(' ')
  return <Link className={classes} {...rest} />
}
