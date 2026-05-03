<script setup>
import { ref, provide, watch } from 'vue'

const props = defineProps({
    modelValue: {
        type: Array,
        default: () => []
    }
})
const emit = defineEmits(['update:modelValue'])

const values = ref([...props.modelValue])

provide('checkboxGroupValue', values)

provide('checkboxGroupToggle', ((val) => {
    const index = values.value.indexOf(val)
    if (index > -1) values.value.splice(index, 1)
    else values.value.push(val)
}))


watch(values, () => {
    emit('update:modelValue', [...values.value])
}, { deep: true })
</script>

<template>
    <div class="checkbox-group">
        <slot />
    </div>
</template>
