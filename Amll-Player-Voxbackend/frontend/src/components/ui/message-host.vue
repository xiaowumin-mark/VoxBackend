<script setup>
import { useMessageState, removeMessage } from './message'

const state = useMessageState()
</script>

<template>
  <teleport to="body">
    <div class="message-container" aria-live="polite" aria-atomic="true">
      <transition-group name="msg" tag="div" class="message-list">
        <div v-for="item in state.list" :key="item.id" class="message" :class="item.type">
          <div class="message-text">{{ item.text }}</div>
          <button v-if="item.closable" class="message-close" @click="removeMessage(item.id)">
            ×
          </button>
        </div>
      </transition-group>
    </div>
  </teleport>
</template>

<style scoped>
.message-container {
  position: fixed;
  top: 16px;
  right: 16px;
  z-index: 9999;
  pointer-events: none;
}

.message-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.message {
  pointer-events: auto;
  min-width: 220px;
  max-width: 360px;
  padding: 10px 12px;
  border-radius: 8px;
  background: rgba(40, 40, 40, 0.9);
  border: 1px solid rgba(255, 255, 255, 0.08);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.25);
  backdrop-filter: blur(8px);
}

.message-text {
  flex: 1;
  font-size: 13px;
  line-height: 1.4;
  word-break: break-word;
}

.message-close {
  appearance: none;
  border: none;
  background: transparent;
  color: inherit;
  cursor: pointer;
  font-size: 16px;
  line-height: 1;
  padding: 2px 4px;
}

.message.info {
  border-left: 3px solid #6bb5ff;
}

.message.success {
  border-left: 3px solid #64d488;
}

.message.warning {
  border-left: 3px solid #f3c64d;
}

.message.error {
  border-left: 3px solid #ff7a7a;
}

.msg-enter-active,
.msg-leave-active {
  transition: transform 200ms var(--f-a), opacity 200ms var(--f-a);
}

.msg-enter-from,
.msg-leave-to {
  transform: translateY(-6px);
  opacity: 0;
}
</style>
