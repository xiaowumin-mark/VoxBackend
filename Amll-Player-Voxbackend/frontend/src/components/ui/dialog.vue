<script setup>
import { ref, defineProps, defineEmits, watch, onBeforeUnmount ,Teleport} from 'vue'
import Button from './button.vue'

const props = defineProps({
    modelValue: Boolean,
    title: String,
    width: {
        type: String,
        default: '300px'
    },
    escClose: {
        type: Boolean,
        default: true
    },
    maskClose: {
        type: Boolean,
        default: true
    }
})

const emit = defineEmits(['update:modelValue', 'open', 'close', 'after-leave'])

const visible = ref(props.modelValue)

watch(() => props.modelValue, v => visible.value = v)
watch(visible, v => emit('update:modelValue', v))

function open() {
    visible.value = true
    emit('open')
    if (props.escClose) document.addEventListener('keydown', onKeydown)
}

function close() {
    visible.value = false
    emit('close')
    document.removeEventListener('keydown', onKeydown)
}

function onMaskClick() {
    if (props.maskClose) close()
}

function onKeydown(e) {
    if (e.key === 'Escape' && props.escClose) close()
}

function onAfterLeave() {
    // 过场动画结束后通知外部（service）可以销毁 DOM 了
    emit('after-leave')
}

onBeforeUnmount(() => {
    document.removeEventListener('keydown', onKeydown)
})
</script>

<template>
    <span @click="open">
        <slot name="trigger" />
    </span>
    <Teleport to="#app">
        <transition name="dialog-fade" @after-leave="onAfterLeave">
            <div v-if="visible" class="dialog-mask" @click="onMaskClick">
                <div class="dialog-box basic" :style="{ width }" @click.stop>
                    <header v-if="title" class="dialog-header">{{ title }}</header>
                    <section class="dialog-body">
                        <slot />
                    </section>
                    <footer class="dialog-footer">
                        <slot name="footer">
                            <Button @click="close">关闭</Button>
                        </slot>
                    </footer>
                </div>
            </div>
        </transition>
    </Teleport>
</template>

<style scoped>
@import url(./basic.css);

.dialog-mask {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.3);
    display: flex;
    align-items: center;
    justify-content: center;
}

.dialog-box {
    background-color: #2B2B2B;
    border-radius: 10px;
}

.dialog-box:hover {
    background-color: #2B2B2B;
}

.dialog-header {
    font-size: 20px;
    font-weight: 500;
    padding: 25px;
    padding-bottom: 20px;
}

.dialog-body {
    margin-bottom: 15px;
    padding: 25px;
    padding-top: 0px;
}

.dialog-footer {
    display: flex;
    justify-content: flex-end;
    background-color: #202020;
    padding: 20px;
}

.dialog-fade-enter-active,
.dialog-fade-leave-active {
    transition: opacity .15s ease, transform .15s ease;
}

.dialog-fade-enter-to,
.dialog-fade-leave-from {
    opacity: 1;
    transform: scale(1);
}

.dialog-fade-enter-from,
.dialog-fade-leave-to {
    opacity: 0;
    transform: scale(1.05);
}
</style>
