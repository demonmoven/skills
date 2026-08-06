import type { PhaseNode } from '../domain/questSelectors'
import i18n from '../i18n'
import Icon from './Icon'
import { useTranslation } from 'react-i18next'

type Props = {
  phases: PhaseNode[]
  currentIdx: number
}

function phaseIcon(phase: PhaseNode): string {
  if (phase.status === 'failed') return 'triangle-alert'
  if (phase.status === 'done') return 'check'
  if (phase.class_ === 'mage' || phase.role === 'review') return 'wand-sparkles'
  if (phase.class_ === 'user') return 'user'
  return 'swords'
}

function phaseState(phase: PhaseNode, currentIdx: number): 'done' | 'active' | 'failed' | 'pending' {
  if (phase.status === 'failed') return 'failed'
  if (phase.status === 'done') return 'done'
  if (phase.phaseIdx === currentIdx || phase.status === 'running') return 'active'
  return 'pending'
}

function phaseMeta(phase: PhaseNode): string {
  const bits: string[] = []
  if (phase.turns > 0) bits.push(`${phase.turns} turns`)
  if (phase.reworkCount > 0) bits.push(i18n.t('phaseProgress.rework', { count: phase.reworkCount }))
  return bits.join(' · ')
}

export default function PhaseProgressBar({ phases, currentIdx }: Props) {
  const { t } = useTranslation()
  if (phases.length === 0) return null

  return (
    <div className="phase-progress" role="list" aria-label={t('aria.questPhaseProgress')}>
      {phases.map((phase, index) => {
        const state = phaseState(phase, currentIdx)
        const meta = phaseMeta(phase)
        return (
          <div className={'phase-progress-step ' + state} role="listitem" key={`${phase.phaseIdx}:${phase.name}`}>
            <div className="phase-progress-rail">
              <div className="phase-progress-dot">
                <Icon name={phaseIcon(phase)} size={13} />
              </div>
              {index < phases.length - 1 && <div className="phase-progress-line" />}
            </div>
            <div className="phase-progress-label">
              <strong title={phase.displayName}>{phase.displayName}</strong>
              {meta && <span>{meta}</span>}
            </div>
          </div>
        )
      })}
    </div>
  )
}
