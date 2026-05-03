<template>
    <main class="basic canactive combobox" ref="combobox" @click="showPopup" :class="{ disabled: props.disabled }">
        <div class="text">
            {{ getValueByIndexAndNode(checkedindex) }}
        </div>
        <div class="icon">
            <Icon icon="&#xE70D;" :size="10"></Icon>
        </div>
    </main>
    <Teleport to="#app">
        <div class="popupbg" @click="isShowPopup = false" :style="{ visibility: isShowPopup ? 'visible' : 'hidden' }">
            <div class="popup" ref="popup" @click.stop :style="{
                transform: `translate(${popupPosition.x}px,${popupPosition.y}px)`
            }">
                <ListItem v-for="(item, index) in props.data" :key="index" disable-icon active-show-bar
                    :active="checkedindex == index" @click="changeChecked(index)">

                    {{ props.dataNode?get(item, props.dataNode):item }}
                </ListItem>
            </div>
        </div>
    </Teleport>

</template>
<script setup>
import Icon from "./icon.vue"
import { ref, defineProps, watch, defineModel, Teleport, defineEmits } from 'vue'
import ListItem from "./list-item.vue"
const popup = ref()
const combobox = ref()
const isShowPopup = ref(false)
const checkedindex = defineModel({ default: 0 })
const emit = defineEmits(['change'])
const popupPosition = ref({
    x: 0,
    y: 0
})
const props = defineProps({
    data: {
        type: Array,
        default: () => []
    },
    disabled: {
        type: Boolean,
        default: false
    },
    dataNode: { // 指向数据节点
        type: String,
        default: ""
    }
})


const showPopup = () => {
    const rect = combobox.value.getBoundingClientRect()


    //   clipPath: `polygon(0 ${clipTop}px,100% ${clipTop}px,100% ${clipBottom}px, 0 ${clipBottom}px)`
    // clip-path: polygon(-15px -15px, calc(100% + 15px) -15px, calc(100% + 15px) calc(100% + 15px), -15px calc(100% + 15px)) !important;
    isShowPopup.value = true
    const l = popup.value.children
    const ele = l[checkedindex.value]
    const rectitem = ele.getBoundingClientRect()

    let y = rect.bottom
    if (l[0]) {
        y -= (checkedindex.value + 1) * (l[0]).offsetHeight + 4
    }
    popupPosition.value = {
        x: rect.left,
        y: y
    }

    if ((y < 0 && y + popup.value.offsetHeight > window.innerHeight)) {

    } else {
        if (y + popup.value.offsetHeight > window.innerHeight) {
            // 计算超出屏幕部分的长度
            const clipBottom = window.innerHeight - (y + popup.value.offsetHeight)
            y += clipBottom - 4

        } else if (y < 0) {
            y = 4
        }
    }


    popupPosition.value = {
        x: rect.left,
        y: y
    }

    const clipTop = (l[0].offsetHeight * checkedindex.value) + 4
    const clipBottom = clipTop + (l[0].offsetHeight * checkedindex.value)
    const a = popup.value.animate([
        {
            clipPath: `polygon(0 ${clipTop}px,100% ${clipTop}px,100% ${clipBottom}px, 0 ${clipBottom}px)`
        },
        {
            clipPath: `polygon(-15px -15px, ${popup.value.offsetWidth + 15}px -15px, ${popup.value.offsetWidth + 15}px ${popup.value.offsetHeight + 15}px, -15px ${popup.value.offsetHeight + 15}px)`
        }
    ], { duration: 800, easing: 'cubic-bezier(0.102, 0.700, 0.000, 1.007)', fill: 'forwards' })


}

watch(checkedindex, (to, from) => {
    isShowPopup.value = false
})

const changeChecked = (index) => {
    checkedindex.value = index
    // change事件
    emit('change', index)
}

function get(data, path, fallback = undefined) {
  if (data == null) return fallback;

  const parts = Array.isArray(path)
    ? path
    : String(path).split('.').filter(Boolean);

  let cur = data;
  for (const key of parts) {
    if (cur == null || !(key in cur)) return fallback;
    cur = cur[key];
  }
  return cur;
}

const getValueByIndexAndNode = (index) => {
    if (props.dataNode) {
        return get(props.data[index], props.dataNode)
    } else {
        return props.data[index]
    }
}
</script>
<style scoped>
@import url("./basic.css");

.combobox {

    min-height: 32px;
    min-width: 150px;
    width: auto;
    display: inline-flex;
    flex-direction: row;
    align-items: center;
    justify-content: flex-start;
    box-sizing: border-box;


}

.combobox:active>.icon {
    transform: translateY(2px);
}

.text {
    flex: 1;
    margin-left: 10px;
    font-size: 14px;
    line-height: 32px;
}

.icon {
    width: 40px;
    height: 32px;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: transform 0.15s ease-out;
}

.popupbg {
    position: fixed;
    top: 0;
    left: 0;
    bottom: 0;
    right: 0;
    width: 100vw;
    height: 100vh;
    z-index: 9;
}

.popup {

    box-shadow: 0px 5px 15px rgba(0, 0, 0, 0.4);
    border-radius: 8px;
    z-index: 10;
    display: flex;
    background-color: rgba(47, 47, 47, 0.980);
    backdrop-filter: blur(10px);
    flex-direction: column;
    padding: 4px;
    box-sizing: border-box;
    width: 150px;
    border: 1px solid rgba(28, 31, 39, 0.678);

}
</style>