<template>
    <div class="auto-layout-item" ref="mainitem" :class="{ open: open }" v-show="props.show">
        <div class="main-item  basic" :class="{ canactive: props.toggle, disabled: props.disabled }" ref="itemele"
            @click="props.toggle ? open = !open : null">
            <div class="icon" v-show="props.icon">
                <Icon :icon="props.icon" :size="20"></Icon>
            </div>
            <div class="cont">
                <div class="title">
                    {{ props.title }}
                </div>
                <div class="desc">
                    {{ props.desc }}
                </div>
            </div>
            <div class="active-icon">
                <slot name="end"></slot>
            </div>
            <div class="toggle-icon" v-show="props.toggle">
                <Icon icon="&#xE70D;" :size="14"></Icon>
            </div>
        </div>
        <div class="toggle-wrapper" :class="{ is_open: open && props.toggle }">
            <div class="toggle-content">
                <div ref="toggleele" class="inner-cont">
                    <slot name="toggle"></slot>
                </div>
            </div>
        </div>
    </div>
</template>

<script setup>

import { ref, onMounted, defineProps, watch, nextTick } from "vue"
import Icon from "./icon.vue"
const props = defineProps({
    id: {
        type: Number,
        default: 0
    },
    toggle: {
        type: Boolean,
        default: false
    },
    disabled: {
        type: Boolean,
        default: false
    },
    icon: {
        type: String,
        default: ""
    },
    title: {
        type: String,
        default: ""
    },
    desc: {
        type: String,
        default: ""
    },
    show: {
        type: Boolean,
        default: true
    }
})
const open = defineModel({ default: false })
watch(
    () => props.show,
    async (visible) => {
        await nextTick()

        if (visible) {
            emit('layout', {
                id: props.id,
                item: mainitem.value,
                height: mainitem.value.offsetHeight,
                open: open.value,
                show: true,
                animate: false
            })
        } else {
            emit('layout', {
                id: props.id,
                item: mainitem.value,
                show: false,
                height: 0,
                open: false,
                animate: false
            })
        }
    },
    { immediate: true }
)
watch(open, async () => {
    // 必须等待 nextTick，否则 getHeight 获取的是展开前/收起前的高度
    await nextTick();
    emit('layout', {
        id: props.id,
        item: mainitem.value,
        height: getHeight(),
        open: open.value,
        show: props.show,
        animate: true
    });
});
const mainitem = ref()
const toggleele = ref(null)
const itemele = ref()
const emit = defineEmits(['layout'])

onMounted(() => {
    console.log("组件创建");
    if (!props.show) {
        return
    }
    emit('layout', { id: props.id, item: mainitem.value, height: getHeight(), open: open.value, show: props.show, animate: true })

})

const getHeight = () => {
    if (!props.show) return 0;
    // 基础高度 (main-item)
    const baseHeight = itemele.value ? itemele.value.offsetHeight : 66;
    // 如果开启了 toggle 并且是 open 状态，累加 toggle 的高度
    // 注意：这里使用 scrollHeight 获取内容真实高度，不受 v-show 影响
    const toggleHeight = (props.toggle && open.value && toggleele.value)
        ? toggleele.value.scrollHeight
        : 0;
    return baseHeight + toggleHeight;
}
</script>

<style scoped>
@import url("./basic.css");

.auto-layout-item {
    position: absolute;
    transform: translateY(var(--position));
    width: 100%;
    box-sizing: border-box;
    overflow: hidden;
    will-change: transform;
}

.main-item {
    position: relative;
    width: 100%;
    height: 66px;
    box-sizing: border-box;
    z-index: 2;
    display: flex;
    align-items: center;
    justify-content: flex-start;
    padding-right: 10px;
}

.open > .main-item {
    border-bottom-left-radius: 0;
    border-bottom-right-radius: 0;
}

.main-item > .icon {
    width: 60px;
    height: 48px;
    display: flex;
    align-items: center;
    justify-content: center;
}

.main-item > .cont {
    flex: 1;
    display: flex;
    flex-direction: column;
    justify-content: flex-start;
    align-items: flex-start;
    box-sizing: border-box;
}

.main-item > .cont > .title {
    font-size: 14px;
}

.main-item > .cont > .desc {
    font-size: 12px;
    opacity: 0.8;
}

.main-item > .active-icon {
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
}

.main-item > .toggle-icon {
    width: 24px;
    height: 24px;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: transform 0.4s cubic-bezier(0.190, 1.000, 0.220, 1.000);
    transition-delay: 150ms;
}

/* 使用Grid实现的高度过渡 */
.toggle-wrapper {
    display: grid;
    grid-template-rows: 0fr; /* 默认关闭状态 */
    transition: grid-template-rows 0.4s cubic-bezier(0.190, 1.000, 0.220, 1.000);
    overflow: hidden;
    border-bottom-left-radius: 5px;
    border-bottom-right-radius: 5px;
}

.toggle-wrapper.is_open {
    grid-template-rows: 1fr; /* 展开状态 */
}

.toggle-content {
    min-height: 0;
}

.inner-cont {
    padding: 1px 0;
}

/* 旋转图标动画 */
.open .toggle-icon {
    transform: rotate(180deg);
}

.toggle-icon {
    transition: transform 0.4s ease;
}
</style>