<script setup>
import { provide, ref, watch } from 'vue'

const props = defineProps({
  modelValue: null
})
const emit = defineEmits(['update:modelValue'])

const value = ref(props.modelValue)

provide('radioGroupValue', value)
provide('radioGroupChange', (val) => {
  value.value = val
})

watch(value, () => {
  emit('update:modelValue', value.value)
})
</script>

<template>
  <div class="radio-group">
    <slot />
  </div>
</template>
