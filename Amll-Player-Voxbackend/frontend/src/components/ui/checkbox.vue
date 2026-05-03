<script setup>
import { inject, computed } from 'vue'
import Icon from './icon.vue'

const props = defineProps({
    value: {
        required: true
    }
})

const groupValue = inject('checkboxGroupValue')
const toggle = inject<any>('checkboxGroupToggle')

const checked = computed(() => groupValue.value && groupValue.value.includes(props.value))

function click() {
    toggle(props.value)
}
</script>

<template>
    <label class="checkbox">
        <input type="checkbox" :checked="checked" @change="click" hidden />
        <span class="icon" :class="{ checked }">
            <Icon icon="&#xE73E;" :size="12" class="inico" />
        </span>
        <span>
            <slot />
        </span>
    </label>
</template>

<style scoped>
.checkbox {
    display: flex;
    align-items: center;
    cursor: pointer;

    margin-top: 10px;
    margin-bottom: 10px;
}


.icon {
    display: inline-flex;
    justify-content: center;
    align-items: center;
    width: 20px;
    height: 20px;
    border-radius: 5px;
    margin-right: 8px;
    border: 1px solid #ffffff96;
    box-sizing: border-box;

}

.icon>.inico {
    color: black;
    font-weight: bold;

    animation: iconclose 0.2s ease-in-out forwards;
}

.icon.checked {
    border-color: transparent;
    background-color: var(--user-color) !important;

}

.icon.checked>.inico {
    animation: iconopen 0.2s ease-in-out forwards;
}

@keyframes iconopen {
    0% {
        clip-path: polygon(0% 0%, 0% 0%, 0% 100%, 0% 100%);
    }

    100% {
        clip-path: polygon(0% 0%, 100% 0%, 100% 100%, 0% 100%);
    }
}

@keyframes iconclose {
    0% {

        clip-path: polygon(0% 0%, 100% 0%, 100% 100%, 0% 100%);
    }

    100% {

        clip-path: polygon(100% 0%, 100% 0%, 100% 100%, 100% 100%);
    }
}
</style>