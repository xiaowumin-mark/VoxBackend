<template>
    <main class="editboxmain" :style="{
        '--edit-box-bar-visibility': isFoucus ? 'visible' : 'hidden'
    }
        ">

        <input type="text" :placeholder="props.placeholder" class="basic editbox" @focusin="isFoucus = true" @focusout="isFoucus = false"
            :class="{
                disabled: props.disabled
            }" v-model="text"
            @change="change"
            >
    </main>
</template>

<script setup>
import { ref, defineProps, defineModel,watch,defineEmits } from 'vue';
const props = defineProps({
    value: {
        type: String,
        default: ""
    },
    type: {
        type: String,
        default: "text"
    },

    placeholder: {
        type: String,
        default: ""
    },
    disabled: {
        type: Boolean,
        default: false
    }

})
const text = defineModel({ default: "" })
watch(() => props.value, val => {
    if (val != null) text.value = val
}, { immediate: true })
const isFoucus = ref(false)
const emit = defineEmits(["change"])
const change = (e ) => {
    emit("change", text.value)
}
</script>

<style scoped>
@import url(./basic.css);



.editboxmain {
    display: inline-block;
    border-radius: 5px;
    overflow: hidden;
    position: relative;

    height: 32px;
    min-width: 100px;
}

.editboxmain::after {
    content: "";
    display: block;
    width: 100%;
    height: 2px;
    background-color: var(--user-color);
    position: absolute;
    bottom: 0;
    visibility: var(--edit-box-bar-visibility);
}

.editbox {
    width: 100%;
    height: 32px;
    box-sizing: border-box;

    outline: none;
    appearance: none;
    padding: 0;
    margin: 0;
    color: white;
    font-size: 14px;
    line-height: 100%;
    padding-left: 10px;
    padding-right: 10px;
    border-bottom-color: rgb(154, 154, 154)
}

.editbox:hover {

    border-bottom-color: rgb(154, 154, 154)
}

.editbox:focus {
    background-color: transparent;
    border-bottom-color: var(--user-color);


}



.editbox::selection {
    background-color: var(--user-color);
    color: white;
}

.editbox::placeholder {
    color: rgba(255, 255, 255, 0.5);
}
</style>