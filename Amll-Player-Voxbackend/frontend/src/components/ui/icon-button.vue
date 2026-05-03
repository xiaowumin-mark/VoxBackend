<template>
    <main @mousedown="md" @mouseup="mu" @mouseleave="ml">
        <icon :icon="props.icon" ref="sif"></icon>
    </main>
</template>
<script setup>
import { ref, defineProps, onMounted } from 'vue'
import icon from './icon.vue'

// 修复类型错误：为ref添加具体类型
const sif = ref(null)

const props = defineProps(
    {
        icon: {
            type: String,
            required: true
        },
        montion: {
            type: String,
            default: 'none'
        }
    }
)


const montions= {
    GlobalNavButton: {
        enter: [[
            {}, {
                transform: 'scaleX(0.5)'
            }
        ], {
            duration: 100,
            easing: 'ease-in-out', // 修复错误：ease应该为easing
            fill: 'forwards'
        }],
        leave: [
            [{}, {
                transform: 'scaleX(1)'
            }], {
                duration: 100,
                easing: 'ease-in-out', // 修复错误：ease应该为easing
                fill: 'forwards'
            }
        ]
    }
}

onMounted(() => {
    // 可以在这里添加初始化逻辑
})

const md = () => {
    if (!props.montion || props.montion === 'none') return;
    const motion = montions[props.montion];
    if (motion) {
        add(motion.enter);
    }
    console.log("enter");
}

const mu = () => {
    if (!props.montion || props.montion === 'none') return;
    const motion = montions[props.montion];
    if (motion) {
        add(motion.leave);
    }
    console.log("leave");
}

const ml = () => {
    // 可以在这里添加mouseleave事件处理
}

// 修复类型错误：为add函数添加类型
const add = (r) => {
    if (sif.value && sif.value.$el) {
        sif.value.$el.animate(r[0], r[1])
    }
}
</script>

<style scoped>
main {
    width: 40px;
    height: 40px;
    max-height: 40px;
    max-width: 40px;
    border-radius: 4px;

    display: flex;
    justify-content: center;
    align-items: center;
    text-align: center;

    transition: background-color 0.1s ease-in-out,filter 0.1s ease-in-out,opacity 0.1s ease-in-out;

}

main:hover {
    background-color: rgba(104, 104, 104, 0.1);
}

main:active {
    background-color: rgba(104, 104, 104, 0.2);
    filter: brightness(0.8);
    opacity: 0.8;
}
</style>