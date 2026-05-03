<template>
    <main class="item basic">
        <slot></slot>
    </main>
</template>
<script setup>
</script>
<style scoped>
@import url("./basic.css");

.item {

    font-size: 14px;
    width: 100%;
    min-height: 50px;
    margin-top: 1px;
    display: flex;
    align-items: center;
    justify-content: flex-start;

    box-sizing: border-box;
    padding-left: 60px;
    padding-top: 10px;
    padding-bottom: 10px;
    border-radius: 0px !important;
    border: none !important;

}
</style>
<style lang="css">
.item>.link_button {
    transform: translateX(-15px)
}
</style>