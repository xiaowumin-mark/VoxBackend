<template>
    <div :class="{ button: true, disabled: props.disabled, accent: (props.accent || checked),full:props.full }" @click="c" ref="main">
        <slot name="begin" ref="begin"></slot>

        <span class="text">
            <slot></slot>
        </span>
        <slot name="end" ref="end"></slot>
    </div>
</template>

<script setup>
import { ref, watch, onMounted, onUnmounted, defineExpose, defineProps, useSlots, readonly, defineModel } from 'vue'
const main = ref()
const checked = defineModel({ default: false })
const props = defineProps({
    disabled: {
        type: Boolean,
        default: false
    },
    accent: {
        type: Boolean,
        default: false
    },
    toggle: {
        type: Boolean,
        default: false
    },
    full: {
        type: Boolean,
        default: false
    }
})



defineExpose({
    main
})
/*
onMounted(() => {
    if (!useSlots().begin && !useSlots().end) {
        main.value.style.transform = 'translateY(-2px)';
    }
})*/
const c = () => {

    if (props.toggle) {
        checked.value = !checked.value;


    }
}
</script>

<style lang="css" scoped>
.button {
    min-height: 32px;
    width: auto;
    padding-left: 12px;
    padding-right: 12px;

    background-color: rgba(100, 100, 100, 0.2);
    border-radius: 5px;

    border: solid 1px rgba(90, 90, 90, 0.1);
    border-top: solid 1px rgba(90, 90, 90, 0.2);

    display: inline-flex;
    align-items: center;
    flex-direction: row;

    gap: 8px;
    margin: 2px;
    margin-left: 4px;
    margin-right: 4px;
    box-sizing: border-box;
    transition: background-color 100ms ease-in-out;

    .text {
        width: 100%;
        font-size: 14px;
        margin: 0;
        padding: 0;
        line-height: 1;
        height: 32px;
        display: flex;
        align-items: center;
        justify-content: center;
        text-align: center;
        
    }



}


.disabled {
    opacity: 0.9;
    filter: brightness(0.7);
    border: solid 1px rgba(90, 90, 90, 0.3);
    pointer-events: none;
}



.button:hover {
    background-color: rgba(90, 90, 90, 0.3);
    border: solid 1px rgba(90, 90, 90, 0.05);
    border-top: solid 1px rgba(90, 90, 90, 0.05);
}

.button:active {
    background-color: rgba(90, 90, 90, 0.12);
    border: solid 1px rgba(90, 90, 90, 0.2);
    filter: brightness(1);
    opacity: 0.7;
}

.accent {
    background-color: var(--user-color);
    color: black;
}

.accent:hover {
    filter: brightness(0.9);
    background-color: var(--user-color);
}

.accent:active {

    background-color: var(--user-color);
    filter: none;
    opacity: none;
    color: rgba(0, 0, 0, 0.4);
}
.full {
    width: 100%;
}
</style>