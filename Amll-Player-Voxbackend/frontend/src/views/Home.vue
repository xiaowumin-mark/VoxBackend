<template>
    <main style="max-width: 1000px;" class="padding-view scroll mainview">
    </main>
</template>

<script setup>
import { Application, Events } from '@wailsio/runtime';
import { useSettingsStore } from '../store/settings';
import { onMounted, onBeforeMount } from 'vue';
import { inject } from 'vue'
</script>