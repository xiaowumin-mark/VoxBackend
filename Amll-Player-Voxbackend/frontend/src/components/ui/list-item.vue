<template>
    <div class="list-item" :active="props.active" ref="main">
        <div class="icon" v-if="!props.disableIcon">
            <slot name="icon"></slot>
        </div>
        <div class="text" :style="{ 'paddingLeft': !props.disableIcon ? '0' : '10px' }">
            <slot></slot>
        </div>
        <div class="bar" v-show="props.active && props.activeShowBar">
            <div class="show"></div>
        </div>
    </div>
</template>

<script setup>
import { ref, defineProps, defineExpose } from "vue"
const props = defineProps({
    active: {
        type: Boolean,
        default: false
    },
    disableIcon: {
        type: Boolean,
        default: false
    },
    activeShowBar: {
        type: Boolean,
        default: false
    }
})
const main = ref(null)

defineExpose({
    target: main
}
)

</script>

<style scoped>
.list-item {
    position: relative;
    width: 100%;
    height: 35px;
    border-radius: 5px;
    display: flex;
    flex-direction: row;
    justify-content: flex-start;
    align-items: center;
    margin-bottom: 4px;
    overflow: hidden;
    padding: 0;
    transition: background-color 0.1s ease-in-out, filter 0.1s ease-in-out,opacity 0.1s ease-in-out;

    .icon {
        width: 40px;
        height: 35px;
        max-width: 40px;
        display: flex;
        justify-content: center;
        align-items: center;
    }

    .text {
        width: calc(100% - 55px);
        height: 100%;
        display: flex;
        flex-direction: row;
        justify-content: flex-start;
        align-items: center;
        overflow: hidden;
        font-size: 14px;
        white-space: nowrap;
    }

    .bar {
        position: absolute;
        left: 0;
        width: 3px;
        height: 100%;
        top: 0;
        display: flex;
        justify-content: center;
        align-items: center;

        .show {
            width: 3px;
            height: 16px;
            background-color: var(--user-color);
            border-radius: 2px;
            transform-origin: center;
            animation: onshow 0.2s ease-in-out;
            transition: transform 0.1s ease-in-out;
        }

    }

}

@keyframes onshow {
    from {
        opacity: 0;
        transform: scaleY(0.5);
    }

    to {
        opacity: 1;
        transform: scaleY(1);
    }
}

.list-item[active="true"] {
    background-color: rgba(104, 104, 104, 0.2);
}

.list-item:hover {

    background-color: rgba(104, 104, 104, 0.1);
}


.list-item:active {
    background-color: rgba(104, 104, 104, 0.2);
    filter: brightness(0.8);
    opacity: 0.8;

    .show{
        transform: scaleY(0.6);
    }
}
</style>