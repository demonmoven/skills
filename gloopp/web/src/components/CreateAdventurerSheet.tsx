import { useCallback, useEffect, useMemo, useState } from 'react'
import { api } from '../api/client'
import type { AdventurerFile, ExecutorInfo } from '../api/types'
import { adventurerModelPolicyLabel, agentKey, agentLabel, isSelectableAgentExecutor } from './util'
import AdvancedSection from './AdvancedSection'
import CapabilityTierBadge, { capabilityTierRank } from './CapabilityTierBadge'
import DesignSelect from './DesignSelect'
import Icon from './Icon'
import i18n from '../i18n'
import useDragToDismiss from '../hooks/useDragToDismiss'
import { useTranslation } from 'react-i18next'

const DEFAULT_CUSTOM_PROMPTS: Record<'warrior' | 'mage', string> = {
  warrior: i18n.t('createAdv.defaultPrompt.warrior'),
  mage: i18n.t('createAdv.defaultPrompt.mage'),
}

export default function CreateAdventurerSheet({
  executors,
  onClose,
  onError,
  onConfigureAgents,
  onCreated,
}: {
  executors: ExecutorInfo[]
  onClose: () => void
  onError: (msg: string) => void
  onConfigureAgents: () => void
  onCreated: (adv: AdventurerFile) => void
}) {
  const { t } = useTranslation()
  const [name, setName] = useState('')
  const [class_, setClass] = useState<'warrior' | 'mage'>('warrior')
  const [agent, setAgent] = useState('')
  const [model, setModel] = useState('')
  const [description, setDescription] = useState('')
  const [customPrompt, setCustomPrompt] = useState('')
  const [tools, setTools] = useState('')
  const [sending, setSending] = useState(false)
  const [nameErr, setNameErr] = useState('')

  const agentOptions = useMemo(() => {
    const seen = new Set<string>()
    return executors
      .map((executor) => ({ executor, agent: agentKey(executor) }))
      .filter(({ executor, agent }) => {
        if (!isSelectableAgentExecutor(executor)) return false
        if (!agent || seen.has(agent)) return false
        seen.add(agent)
        return true
      })
      .sort((a, b) => capabilityTierRank(a.executor.capability_tier) - capabilityTierRank(b.executor.capability_tier) || agentLabel(a.agent).localeCompare(agentLabel(b.agent)))
  }, [executors])

  const selectedExecutor = useMemo(() => {
    if (!agent) return null
    return agentOptions.find((item) => item.agent === agent)?.executor || null
  }, [agent, agentOptions])

  const agentSelectOptions = agentOptions.map((item) => ({ value: item.agent, label: agentLabel(item.agent) }))
  const { sheetRef, handleRef, sheetStyle, scrimStyle } = useDragToDismiss({ onDismiss: onClose })

  useEffect(() => {
    if (agent && agentOptions.some((item) => item.agent === agent)) return
    setAgent(agentOptions[0]?.agent || '')
  }, [agent, agentOptions])

  const submit = useCallback(async () => {
    if (!name.trim()) {
      setNameErr(t('createAdv.error.nameEmpty'))
      return
    }
    if (!agent.trim()) {
      onError(t('createAdv.error.needAgent'))
      return
    }
    setNameErr('')
    setSending(true)
    try {
      const res = await api.post<{ item: AdventurerFile }>('/api/adventurers', {
        name: name.trim(),
        class: class_,
        agent: agent.trim(),
        model: model.trim(),
        description: description.trim(),
        custom_prompt: customPrompt.trim() || DEFAULT_CUSTOM_PROMPTS[class_],
        tools: tools
          .split(',')
          .map((t) => t.trim())
          .filter(Boolean),
        level: 1,
        status: 'active',
      })
      onCreated(res.item)
      onClose()
    } catch (e) {
      onError(e instanceof Error ? e.message : t('createAdv.error.recruitFail'))
    } finally {
      setSending(false)
    }
  }, [
    name,
    class_,
    agent,
    model,
    description,
    customPrompt,
    tools,
    onCreated,
    onClose,
    onError,
  ])

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') {
        e.preventDefault()
        onClose()
      }
      if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
        e.preventDefault()
        submit()
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [onClose, submit])

  useEffect(() => {
    const prev = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => {
      document.body.style.overflow = prev
    }
  }, [])

  return (
    <div className="sheet-scrim" style={scrimStyle} onClick={onClose} role="dialog" aria-modal="true" aria-label={t('createAdv.aria')}>
      <div className="sheet-card design-sheet" ref={sheetRef} style={sheetStyle} onClick={(e) => e.stopPropagation()}>
        <i className="sheet-handle" ref={handleRef} aria-hidden="true" />
        <div className="sheet-head design-sheet-head">
          <div className="sheet-title-row">
            <span className="sheet-icon">
              <Icon name="users" />
            </span>
            <div>
              <div className="sheet-title-line">
                <h2>{t('createAdv.title')}</h2>
                <span className="sheet-badge">Guild slot</span>
              </div>
              <p className="sheet-subtitle">{t('createAdv.subtitle')}</p>
            </div>
          </div>
          <button className="icon-button" onClick={onClose} aria-label={t('createAdv.close')}>
            <Icon name="x" />
          </button>
        </div>
        <div className="sheet-body design-sheet-body">
          <label className="field">
            <span>{t('createAdv.field.name')}</span>
            <input
              value={name}
              onChange={(e) => {
                setName(e.target.value)
                if (nameErr) setNameErr('')
              }}
              onKeyDown={(e) => {
                if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
                  e.preventDefault()
                  submit()
                }
              }}
              placeholder={t('createAdv.field.namePlaceholder')}
              autoFocus
            />
            {nameErr && <small className="field-error">{nameErr}</small>}
          </label>
          <label className="field">
            <span>{t('createAdv.field.class')}</span>
            <div className="option-cards two-col">
              <button
                className={'option-card ' + (class_ === 'warrior' ? 'selected' : '')}
                onClick={() => setClass('warrior')}
                type="button"
              >
                <Icon name="sword" />
                <div>
                  <strong>{t('createAdv.classOption.warriorTitle')}</strong>
                  <em>{t('createAdv.classOption.warriorDesc')}</em>
                </div>
              </button>
              <button
                className={'option-card ' + (class_ === 'mage' ? 'selected' : '')}
                onClick={() => setClass('mage')}
                type="button"
              >
                <Icon name="wand" />
                <div>
                  <strong>{t('createAdv.classOption.mageTitle')}</strong>
                  <em>{t('createAdv.classOption.mageDesc')}</em>
                </div>
              </button>
            </div>
          </label>
          <div className="grid-2">
            <label className="field">
              <span>Agent</span>
              <DesignSelect
                value={agent}
                options={agentSelectOptions}
                onChange={setAgent}
                ariaLabel={t('aria.agent')}
                disabled={agentSelectOptions.length === 0}
              />
              {agentSelectOptions.length === 0 && (
                <div className="field-action-hint">
                  <small className="field-error">{t('createAdv.agent.noEnabled')}</small>
                  <button type="button" className="button tiny" onClick={onConfigureAgents}>
                    <Icon name="settings" />
                    {t('createAdv.agent.goEnable')}
                  </button>
                </div>
              )}
            </label>
            <label className="field">
              <span>{t('createAdv.field.modelPolicy')}</span>
              <input
                value={model}
                onChange={(e) => setModel(e.target.value)}
                placeholder={t('createAdv.field.modelPlaceholder')}
              />
              <small className="field-hint">{adventurerModelPolicyLabel(model, selectedExecutor)}</small>
            </label>
            <label className="field">
              <span>{t('createAdv.field.capability')}</span>
              <div className="capability-field">
                <CapabilityTierBadge tier={selectedExecutor?.capability_tier} />
              </div>
              <small className="field-hint">{t('createAdv.field.capabilityHint')}</small>
            </label>
          </div>
          <AdvancedSection hint={t('createAdv.advancedHint')}>
            <label className="field">
              <span>{t('createAdv.field.desc')}</span>
              <textarea
                rows={2}
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder={t('createAdv.field.descPlaceholder')}
              />
            </label>
            <label className="field">
              <span className="field-label-row">
                <span>{t('createAdv.field.customPrompt')}</span>
                <em>{t('createAdv.field.customPromptHint')}</em>
              </span>
              <textarea
                rows={5}
                value={customPrompt}
                onChange={(e) => setCustomPrompt(e.target.value)}
                placeholder={t('createAdv.field.customPromptPlaceholder')}
              />
              <small className="field-hint">
                {t('createAdv.field.customPromptDefaultPrefix')}
                {class_ === 'warrior' ? t('createAdv.field.customPromptDefaultWarrior') : t('createAdv.field.customPromptDefaultMage')}。
              </small>
            </label>
            <div className="grid-2">
              <label className="field">
                <span>{t('createAdv.field.tools')}</span>
                <input
                  value={tools}
                  onChange={(e) => setTools(e.target.value)}
                  placeholder={t('createAdv.field.toolsPlaceholder')}
                />
              </label>
              <label className="field">
                <span>{t('createAdv.field.level')}</span>
                <input value="1" disabled readOnly />
                <small className="field-hint">{t('createAdv.field.levelHint')}</small>
              </label>
            </div>
          </AdvancedSection>
        </div>
        <div className="sheet-foot design-sheet-foot">
          <button className="button ghost" onClick={onClose} disabled={sending}>
            {t('createAdv.action.cancel')}
          </button>
          <div className="spacer" />
          <button className="button primary" onClick={submit} disabled={sending || agentSelectOptions.length === 0}>
            {sending ? (
              <>
                <Icon name="spinner" />
                {t('createAdv.action.recruiting')}
              </>
            ) : (
              <>
                <Icon name="plus" />
                {t('createAdv.action.recruit')}
                <kbd>⌘↵</kbd>
              </>
            )}
          </button>
        </div>
      </div>
    </div>
  )
}
