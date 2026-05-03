import { reactive } from 'vue'

const state = reactive({
  list: [],
  seed: 1
})

const defaultDuration = 3000

const normalize = (input) => {
  if (typeof input === 'string') {
    return { text: input }
  }
  if (input && typeof input === 'object') {
    return { ...input }
  }
  return { text: String(input) }
}

export const removeMessage = (id) => {
  const idx = state.list.findIndex((item) => item.id === id)
  if (idx === -1) return
  const item = state.list[idx]
  if (item.timer) {
    clearTimeout(item.timer)
  }
  state.list.splice(idx, 1)
}

const pushMessage = (input) => {
  const opt = normalize(input)
  const item = {
    id: state.seed++,
    text: opt.text ?? '',
    type: opt.type ?? 'info',
    duration: Number.isFinite(opt.duration) ? opt.duration : defaultDuration,
    closable: opt.closable ?? false,
    timer: null
  }

  if (item.duration > 0) {
    item.timer = setTimeout(() => removeMessage(item.id), item.duration)
  }

  state.list.push(item)
  return item.id
}

const message = (input) => pushMessage(input)
message.success = (text, options = {}) => pushMessage({ ...options, text, type: 'success' })
message.error = (text, options = {}) => pushMessage({ ...options, text, type: 'error' })
message.warning = (text, options = {}) => pushMessage({ ...options, text, type: 'warning' })
message.info = (text, options = {}) => pushMessage({ ...options, text, type: 'info' })

export const useMessageState = () => state

export default {
  install(app) {
    app.config.globalProperties.$message = message
    app.provide('message', message)
  }
}
