<template>
  <div 
    class="v-switch" 
    :class="{ 'is-checked': modelValue, 'is-disabled': disabled || loading }"
    @click="handleClick"
  >
    <input
      type="checkbox"
      class="v-switch-input"
      :checked="modelValue"
      :disabled="disabled || loading"
      @change.stop
    />
    
    <span class="v-switch-core">
      <div v-if="loading" class="v-switch-loading-icon"></div>
      <span class="v-switch-button"></span>
    </span>
  </div>
</template>

<script setup>
/**
 * @property {Boolean} modelValue - 绑定值，对应 v-model
 * @property {Boolean} disabled - 是否禁用
 * @property {Boolean} loading - 是否处于加载状态
 */
const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false
  },
  disabled: Boolean,
  loading: Boolean,
});

const emit = defineEmits(['update:modelValue', 'change']);

const handleClick = () => {
  if (props.disabled || props.loading) return;

  const newValue = !props.modelValue;
  
  // 1. 触发 v-model 更新
  emit('update:modelValue', newValue);
  
  // 2. 触发专门的 change 事件，方便用户处理逻辑
  emit('change', newValue);
};
</script>

<style scoped>
.v-switch {
  display: inline-flex;
  align-items: center;
  position: relative;
  height: 20px;
  cursor: pointer;
  user-select: none;
  border: rgba(255, 255, 255, 0.541) solid 1px;
  border-radius: 12px;
}

.v-switch-input {
  position: absolute;
  width: 0;
  height: 0;
  opacity: 0;
}

.v-switch-core {
  position: relative;
  width: 40px;
  height: 20px;
  background-color: #0000002a;
  border-radius: 11px;
  transition: all 0.4s var(--f-a);
}

.v-switch.is-checked .v-switch-core {
  background-color: var(--user-color);
}

.v-switch-button {
  position: absolute;
  top: 4px;
  left: 4px;
  width: 12px;
  height: 12px;
  background-color: #bababa;
  border-radius: 50%;
  transition: all 0.4s var(--f-a);
  box-shadow: 0 2px 4px rgba(0,0,0,0.1);
}


.v-switch.is-checked .v-switch-button {
  left: 100%;
  transform: translateX(-16px);
  background-color: #000000;
}

/* 禁用与加载状态样式 */
.is-disabled {
  cursor: not-allowed;
  opacity: 0.6;
}


@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>