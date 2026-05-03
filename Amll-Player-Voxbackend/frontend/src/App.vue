<script setup>
import { RouterView, useRoute, useRouter } from 'vue-router'
import { onMounted, reactive, ref, watch } from 'vue'
import iconButton from './components/ui/icon-button.vue'
import icon from './components/ui/icon.vue'
import listItem from './components/ui/list-item.vue'
import MessageHost from './components/ui/message-host.vue'

const route = useRoute()
const router = useRouter()
let inited = false

// 获取meta
const showNav = ref(route.meta.showNav)

// 监听路由变化
router.afterEach(() => {
    showNav.value = route.meta.showNav
})
const isMoreNav = ref(false)
const floatbar = ref()
const items = ref([])
const nav = reactive([
])

// 获取全部路由
router.getRoutes().forEach((route) => {
    if (route.meta.home) {
        nav.push({
            name: route.meta.title,
            path: route.path,
            active: false,
            icon: route.meta.icon,
            id: items.value.length
        })
    }
})

// 全部路由发生变化时，更新导航栏
watch(
    () => router.getRoutes(),
    (newRoutes) => {
        nav.length = 0 // 清空现有导航项
        newRoutes.forEach((route) => {
            if (route.meta.home) {
                nav.push({
                    name: route.meta.title,
                    path: route.path,
                    active: false,
                    icon: route.meta.icon,
                    id: items.value.length
                })
            }
        })
    }
)

onMounted(() => {
    nav.forEach((item) => {
        if (item.path === route.path) {
            item.active = true
        } else {
            item.active = false
        }
    })

})

watch(route, (to, from) => {
    nav.forEach((item) => {
        if (item.path === to.path) {
            item.active = true
        } else {
            item.active = false
        }
    })
})
watch(nav, (to, from) => {
    if (!route.meta.showNav) {
        return
    }

    const e = to.findIndex((item) => item.active)
    toNevItem(e)
})

const toNevItem = (e) => {
    const rect = items.value[e].$el.getBoundingClientRect();
    let x = rect.left
    let y = rect.top + (items.value[e].$el.offsetHeight / 2 - floatbar.value.offsetHeight / 2)
    if (!inited) {
        // 👇 第一次：直接设置
        floatbar.value.style.transform = `translate(${x}px, ${y}px)`
        inited = true
        return
    }

    // 计算是向上移动还是向下移动
    // 获取当前位置
    const n = floatbar.value.getBoundingClientRect();
    if (y - n.top == 0) { // 移动到同一位置
        return
    }
    const ys = n.top - rect.top
    if (ys < 0) {
        if (floatbar.value) {
            floatbar.value.style.transformOrigin = "bottom";
        }
    } else {
        if (floatbar.value) {
            floatbar.value.style.transformOrigin = "top";
        }
    }


    floatbar.value.animate([{
        transform: `translate(${n.left}px, ${n.top}px) scaleY(1)`,
    }, {
        transform: `translate(${x}px, ${y}px) scaleY(${2})`,
        offset: 0.7,
    }, {
        transform: `translate(${x}px, ${y}px) scaleY(1)`,
    }], {
        duration: 500,
        fill: "forwards",
        easing: "cubic-bezier(0.075, 0.820, 0.165, 1.000)"

    }
    )
}
const switchPage = (e, path) => {
    nav.forEach((item) => {
        item.active = item.path === path && e === item.id
        if (item.active) {
            router.push(item.path)
        }
    })

}

</script>

<template>
    <MessageHost />
    <div class="floatbar" ref="floatbar" v-show="showNav"></div>
    <div class="main">
        <div :class="{ nav: true, min_nav: !isMoreNav }" v-show="showNav">
            <div class="top">
                <iconButton icon="&#xE700;" montion="GlobalNavButton" @click="isMoreNav = !isMoreNav"></iconButton>
            </div>
            <div class="cont">

                <listItem v-for="item in nav" :active="item.active" @click="switchPage(item.id, item.path)"
                    :key="item.id" ref="items">
                    <template #icon>
                        <icon :icon="item.icon"></icon>
                    </template>
                    {{ item.name }}
                </listItem>
            </div>
            <div class="bottom"></div>
        </div>
        <div :class="{ view: true, more_view: !isMoreNav, only: !showNav }">
            <router-view v-slot="{ Component, route }">
                <transition name="fade-slide">
                    <component :is="Component" :key="route.fullPath" class="view-page" />
                </transition>
            </router-view>
        </div>
    </div>
</template>

<style scoped>
.view-page {
    position: absolute;
    width: 100%;
    height: 100%;

}

/* 离开动画：旧页面淡出 */
.fade-slide-leave-active {
    transition: opacity 0.2s ease;
}

.fade-slide-leave-to {
    opacity: 0;
}

/* 进入动画：新页面淡入并上移 */
.fade-slide-enter-active {
    transition: opacity 0.3s var(--f-a), transform 0.3s var(--f-a);
    transition-delay: 100ms;
}

.fade-slide-enter-from {
    opacity: 0;
    transform: translateY(25vh);
}

.fade-slide-enter-to {
    opacity: 1;
    transform: translateY(0);
}

.main {
    height: 100%;
    width: 100%;
    display: flex;
    flex-direction: row;
    justify-content: flex-start;
}

.nav {
    height: 100%;
    width: 300px;
    padding: 4px;

    transition: width 250ms var(--f-a);

    box-sizing: border-box;


    display: flex;
    flex-direction: column;
    justify-content: flex-start;
    align-items: flex-start;
    gap: 4px;

    .top {
        width: 100%;
    }

    .cont {
        width: 100%;
        flex: 1;
        padding-top: 8px;
    }

    .bottom {
        width: 100%;
    }


}

.view {
    height: 100%;
    width: calc(100% - 300px);
    background-color: var(--user-view-bg-color);
    overflow: hidden;
    border-top-left-radius: 8px;
    box-sizing: border-box;
    position: relative;
    display: flex;
    flex-direction: column;
    align-items: center;

}

.min_nav {
    width: 48px;

}

.more_view {
    width: calc(100% - 48px);
}

.floatbar {
    position: absolute;
    top: 0;
    left: 0;
    width: 3px;
    height: 18px;
    background-color: var(--user-color);
    border-radius: 2px;
    transform: translate(4px, -100%);
}

.only {
    width: 100%;
    height: 100%;
    border: none;
    background-color: transparent;
    border-radius: 0px;
}
</style>
