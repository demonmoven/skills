import type { QuestStatus } from '../api/types'
import { statusLabel } from './util'

export default function StatusBadge({
  status,
  title,
  muted = false,
}: {
  status: QuestStatus
  title?: string
  muted?: boolean
}) {
  const cls = ['status-badge', 'status-' + status]
  if (muted) cls.push('muted')
  return <span className={cls.join(' ')} title={title}>{statusLabel(status) || status}</span>
}
