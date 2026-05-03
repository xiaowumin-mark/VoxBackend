<template>
    <main :class="{ link_button: true, disabled: props.disabled }" @click="BrowserOpenURL(props.navigateUrl)">
        <span>
            <slot></slot>
        </span>
    </main>
</template>

<script setup>
import { ref, defineProps } from 'vue'
import { Browser } from '@wailsio/runtime';
const props = defineProps({
    disabled: {
        type: Boolean,
        default: false
    },
    navigateUrl: {
        type: String,
        default: ''
    }
})
const BrowserOpenURL = (url) => {
    Browser.OpenURL(url)
}
</script>

<style lang="css">
.link_button {
    transition: background-color 0.1s ease-in-out;
    min-height: 34px;
    width: auto;
    padding-left: 12px;
    padding-right: 12px;

    border-radius: 5px;


    display: inline-flex;
    align-items: center;
    justify-content: center;

    gap: 8px;

    margin: 2px;
    margin-left: 4px;
    margin-right: 4px;

    span {
        margin: 0;
        padding: 0;
        line-height: 1;
        font-size: 14px;
        color: var(--user-color);
        filter: brightness(1.2);
    }
}

.link_button:hover {
    background-color: rgb(90, 90, 90, 0.25);
}

.link_button:active {
    background-color: rgb(90, 90, 90, 0.15);

    span {
        filter: brightness(1.5);
    }
}

.disabled {
    background-color: none;
    pointer-events: none;

    span {
        color: gray;
    }
}
</style>