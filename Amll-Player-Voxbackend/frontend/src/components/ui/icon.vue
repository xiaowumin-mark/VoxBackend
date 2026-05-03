<template>
    <main class="icon" :style="{ height: props.size + 'px', width: props.size + 'px', fontSize: props.size + 'px' }"
        v-html="props.icon" ref="icon"></main>
</template>

<script setup>
import { defineProps, ref, defineExpose } from 'vue'
const icon = ref(null);
const props = defineProps({
    icon: String,
    size: {
        type: Number,
        default: 16
    }
})
defineExpose({
    icon
})
</script>
<style scoped>
.icon {
    font-family: "Segoe Fluent Icons" !important;
    line-height: 1;
    display: inline-flex;
    /* 使用 flex 居中对齐 */
    align-items: center;
    justify-content: center;
    vertical-align: middle;
    /* 保证基线对齐 */
    border: none !important;
    outline: none !important;
    background: none !important;
    margin: 0 !important;
    padding: 0 !important;
}
</style>