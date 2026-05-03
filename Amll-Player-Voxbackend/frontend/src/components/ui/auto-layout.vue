<template>
  <div class="layout" :style="{ height: h + 'px' }">
    <!-- 向 slot 暴露一个回调函数 onLayout -->
    <slot :onLayout="onLayout"></slot>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue';
const elements = ref([])
const gap = 6
const lastPositionMap = new Map()
// 1. 定义响应式高度
const h = ref(0);
function onLayout(payload) {

  if (!elements.value) return

  // 显示的，添加或更新 根据id 排序
  let index = elements.value.findIndex(item => item.id === payload.id)
  if (index === -1) {
    elements.value.push(payload)
  } else {
    elements.value[index] = payload
  }
  // 根据id 排序
  elements.value.sort((a, b) => a.id - b.id)

  const nowIndex = elements.value.findIndex(item => item.id === payload.id)
  let last = 10
  let positionList = []
  elements.value.forEach((item, index) => {
    //const t = (index - nowIndex + 1) * 20

    positionList.push({
      id: item.id,
      position: last
    })
    last += (item.height + gap) * item.show
  })
  h.value = last
  let animateList = []
  // diff positionList 和 lastPositionList ，找出需要动画的
  positionList.forEach(item => {
    const prev = lastPositionMap.get(item.id)

    if (prev == null) {
      // 🆕 新元素（第一次出现）
      animateList.push({
        id: item.id,
        from: item.position,
        to: item.position,
        type: 'enter',
      })
    } else if (prev !== item.position) {
      // 🔁 位置发生变化
      animateList.push({
        id: item.id,
        from: prev,
        to: item.position,
        type: 'move',
      })
    }
    // 如果 prev === position → 不需要动画
  })
  console.log("animatelist",animateList);
  
  //console.log('animatelist', animateList);
  animateList.forEach(item => {

    const id = elements.value.findIndex(t => t.id === payload.id) // 根据用户点击的id获取元素索引


    const tindex = elements.value.findIndex(t => t.id === item.id)
    const ele = elements.value[tindex].item

    const t = (tindex - nowIndex + 1) * 20 * Number(payload.animate) + 150 * Number(!payload.open)

    // 取消动画
    //ele.getAnimations().forEach(animation => animation.cancel())
    // / animate
    ele.animate([
      { "transform": `translateY(${item.from}px)` },
      { "transform": `translateY(${item.to}px)` }
    ], {
      duration: payload.animate ? 400 : 0,
      easing: 'cubic-bezier(0.190, 1.000, 0.220, 1.000)',
      fill: 'forwards',
      delay: t
    })

  })


  lastPositionMap.clear()
  positionList.forEach(item => {
    lastPositionMap.set(item.id, item.position)
  })

  //console.log(positionList);


}
onMounted(() => {
  console.log('✅ auto-layout 组件已挂载')
  console.log(elements.value);

})
/*const setLayout = (list, options = {
  isinit: true,
  id: 0
}) => {

  let position = elements.value.findIndex(item => item.id === options.id)

  let last = 10
  list.forEach((item, index) => {
    // 取消动画
    item.item.getAnimations().forEach(animation => animation.cancel())


    const t = (index - position + 1) * 20 * Number(!options.isinit)


    //setTimeout(() => {
    //animate


    console.log('设置动画', index);
    item.item.animate([
      {},
      { "transform": `translateY(${last}px)` }
    ], {
      duration: options.isinit ? 0 : 400,
      easing: 'ease-out',
      fill: 'forwards',
      delay: t
    })
    last += item.height + gap
    if (index === list.length - 1) {
      h.value = last
    }
    item.item.setAttribute('data-position', last.toString())


    // }, t)

  })
}*/
</script>

<style scoped>
.layout {
  width: 100%;
  overflow-y: hidden;
  transition: height 400ms cubic-bezier(0.190, 1.000, 0.220, 1.000);
}
</style>
