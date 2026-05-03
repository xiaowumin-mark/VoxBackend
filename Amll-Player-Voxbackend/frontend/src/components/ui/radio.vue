<script setup>
import { inject, computed } from 'vue'

const props = defineProps({
    value: {
        required: true
    }
})

const groupValue = inject('radioGroupValue')
const changeHandler = inject('radioGroupChange')

const checked = computed(() => groupValue.value === props.value)

function select() {
    changeHandler(props.value)
}
</script>

<template>
    <label class="radio">
        <input type="radio" :checked="checked" @change="select" hidden />
        <span class="icon" :class="{ checked }"></span>
        <span>
            <slot />
        </span>
    </label>
</template>
<style scoped>
.radio {
    display: flex;
    align-items: center;
    cursor: pointer;
    margin-top:  10px;
    margin-bottom: 10px;
}

.icon {
    display: inline-flex;
    justify-content: center;
    align-items: center;
    width: 20px;
    height: 20px;
    border-radius: 50%;
    margin-right: 8px;
    border: 1px solid #ffffff96;
    box-sizing: border-box;


}

.icon.checked {
    border-color: transparent;
    background-color: var(--user-color) !important;
}

.icon.checked::before {
    content: "";
    display: block;
    width: 100%;
    height: 100%;
    border-radius: 50%;
    background-color: rgb(0, 0, 0);
    transform: scale(0.6);
    transition: transform 0.1s ease;
}

.radio:hover .icon{
    background-color: rgba(255, 255, 255, 0.062);
}

.radio:hover .icon.checked::before {
    transform: scale(0.7);
}

.radio:active .icon.checked::before {
    transform: scale(0.6);
}

.radio:active .icon {
    filter: brightness(0.8);
}
</style>
