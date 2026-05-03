<template>
    <main style="width: 100%;" class="padding-view scroll">
        <div style="max-width: 1000px;position: relative;margin: 0 auto;">
            <p style="font-size: 28px;">设置</p>
            <AutoLayout v-slot="{ onLayout }" style="margin-top: 20px;">

                <AutoLayoutItem @layout="onLayout" icon="&#xE99A;" title="当前模型" desc="模型选择" :id=1 >
                    <template #end>
                        <Combobox v-model="settingsStore.model" :data="settingsStore.ModelNames" />
                    </template>
                    
                </AutoLayoutItem>
            </AutoLayout>


        </div>
    </main>
</template>
<script setup>
import AutoLayoutItem from '../components/ui/auto-layout-item.vue';
import AutoLayout from '../components/ui/auto-layout.vue';
import Button from '../components/ui/button.vue';
import { onMounted, onUnmounted, ref, watch } from 'vue';
import EditBox from '../components/ui/edit-box.vue';
import { useSettingsStore } from '../store/settings';
import { Events, Window } from '@wailsio/runtime';
import ListItem from '../components/ui/list-item.vue';
import ItemCard from '../components/ui/item-card.vue';
import Switch from '../components/ui/switch.vue';
import Combobox from '../components/ui/combobox.vue';
import LinkButton from '../components/ui/link-button.vue';
import Dialog from '../components/ui/dialog.vue';


const settingsStore = useSettingsStore();
</script>

<style lang="css" scoped>
@import url("../components/ui/basic.css");
</style>