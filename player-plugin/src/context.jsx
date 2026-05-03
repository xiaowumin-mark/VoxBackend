// 顶部导入（需确保环境中已加载 GSAP 和 figma-squircle）
import gsap from "gsap";
import { VoxBackendStates } from "./store";
const { useEffect, useRef, useState, useCallback, useLayoutEffect } = React;
const { createPortal } = ReactDOM;
// Jotai
const { atom, useAtom, useAtomValue, useSetAtom } = Jotai;

const svg = `<svg width="100%" height="100%" preserveAspectRatio="xMidYMid slice" xmlns="http://www.w3.org/2000/svg">
     <defs>
                <mask id="hole">
                    <rect width="100%" height="100%" fill="rgba(255,255,255,0.3)" />
                    <rect fill="white" id="ssma" y="0%" opacity="0" style=" width:100%; height:100%" />
                    <use href="#icon" x="1.1em" y="0%" width="1.5em" height="100%" fill="black" style="transform-origin: center bottom; overflow: visible;"/>
                </mask>

                <symbol id="icon" viewBox="0 0 50 52" style="transform-origin: center bottom;">
                    <path fill-rule="evenodd" clip-rule="evenodd"
                        d="M30.2251 6.20795L44.0677 20.4315C44.0677 20.4315 41.1564 22.9131 38.9879 23.6065C37.4879 24.0861 35.178 24.1144 35.178 24.1144L9.96922 48.5612C9.96922 48.5612 9.19329 48.9835 8.57226 49.0057C7.9073 49.0295 7.50677 48.9785 7.04831 48.5612C6.90508 48.4309 6.73082 48.1802 6.73082 48.1802L2.60343 51.5457C2.60343 51.5457 2.21541 51.8124 1.90495 51.7996C1.61601 51.7878 1.20647 51.5457 1.20647 51.5457L0.127005 50.5297C0.127005 50.5297 -0.00121652 50.1934 8.72544e-06 49.9582C0.00156928 49.6586 0.254002 49.2597 0.254002 49.2597L3.49241 44.9418L3.11142 44.6244C3.11142 44.6244 2.43226 43.7548 2.41294 43.1004C2.39228 42.4006 3.11142 41.4494 3.11142 41.4494L26.7327 15.0977C26.7327 15.0977 26.611 12.6243 27.1137 11.0338C27.7708 8.95497 30.2251 6.20795 30.2251 6.20795ZM8.38177 45.7038L31.7491 22.527C31.7491 22.527 30.4689 21.5646 29.7807 20.8125C29.0976 20.0661 28.2567 18.7171 28.2567 18.7171L5.96884 43.4179L8.38177 45.7038Z" />
                    <path
                        d="M46.6077 17.5741L32.8921 3.47753C32.8921 3.47753 36.2726 0.344453 38.9879 0.0486313C41.9035 -0.269014 44.8302 1.00455 47.1157 2.84255C49.4886 4.75095 49.6469 6.86321 49.9731 9.89085C50.3519 13.4071 46.6077 17.5741 46.6077 17.5741Z" />
                </symbol>
            </defs>
            <rect width="100%" height="100%" fill="white" mask="url(#hole)" />
  </svg>`;


function createTopic() {
    const div = document.createElement('span');
    div.innerHTML = "自动过渡";
    div.style.width = "100px";
    div.style.fontSize = "max(1.6vh, 0.8em)";
    div.style.fontWeight = "bold";
    div.style.display = "flex";
    div.style.justifyContent = "center";
    div.style.alignItems = "center";

    div.style.color = "white";
    div.style.backgroundColor = "transparent";
    div.style.padding = "0.5em 1em";
    div.style.borderRadius = "12px";

    // mask 渐变：透明 → 白色 → 透明，控制文字高光区域
    div.style.mask = "linear-gradient(90deg, transparent, white, transparent)";
    div.style.maskSize = "200% 100%";
    div.style.webkitMask = "linear-gradient(90deg, transparent, white, transparent)";
    div.style.webkitMaskSize = "200% 100%";
    div.style.backgroundClip = "text";
    // 动画：移动 mask-position
    div.style.animation = "shine-move 2s ease infinite";
    // 阴影
    div.style.textShadow = "0 0 3px rgba(255,255,255,0.5)";

    div.style.mixBlendMode = "plus-lighter";
    return div;
}
function MainContext() {
    const [container, setContainer] = useState(null);
    const hasFoundContainer = useRef(false);
    const [vocalGain, setVocalGain] = useAtom(VoxBackendStates.VocalGain);
    const [isShowEleInPlayer, setIsShowEleInPlayer] = useAtom(VoxBackendStates.ShowEleInPlayer);
    const appleRef = useRef(null);
    const iconRef = useRef(null);
    const ssmaRef = useRef(null);
    const [isShowCtrl, setIsShowCtrl] = useState(false);
    const svgInjectedRef = useRef(false);

    // 核心引用：音量值、拖拽累计值、拖拽状态
    const vRef = useRef(vocalGain);
    const logicalVRef = useRef(vocalGain);
    const logicalHRef = useRef(0);
    const isClickRef = useRef(false);
    const stateRef = useRef({
        baseScale: 1, stretchY: 0, stretchX: 0,
        offsetY: 0, followX: 0, followY: 0,
    });






    const [crossfadeing, setCrossfadeing] = useAtom(VoxBackendStates.Crossfadeing);
    const [musicQualityAtom, setMusicQualityAtom] = useAtom(extensionContext.amllStates.musicQualityAtom);
    useEffect(() => {
        const plse = findElementsByClassContains(
            document.getElementById('root'), 'progressBarLabels'
        )
        if (plse.length > 0) {
            if (plse[0].children.length == 3) {
                const pl = plse[0].children[1];
                if (pl.children.length > 0 && findElementsByClassContains(pl, 'qualityTag').length > 0) {
                    pl.children[0].style.display = "none";
                    if (crossfadeing) {
                        const div = createTopic();
                        pl.appendChild(div);
                    } else {
                        let delindex = -1
                        for (let i = 0; i < pl.children.length; i++) {


                            if (pl.children[i].classList.length == 0) {
                                // 删除节点
                                delindex = i
                            } else {

                                pl.children[i].style.display = "auto";
                            }

                        }
                        if (delindex != -1) {
                            pl.removeChild(pl.children[delindex])
                        }
                    }
                } else if (pl.children.length == 0) {
                    if (crossfadeing) {
                        const div = createTopic();
                        pl.appendChild(div);
                    }
                } else {
                    pl.innerHTML = "";
                }

            }
        }

    }, [crossfadeing])
    useLayoutEffect(() => {
        requestAnimationFrame(() => {
            if (musicQualityAtom.type != "none") {
                if (crossfadeing) {
                    const plse = findElementsByClassContains(
                        document.getElementById('root'), 'progressBarLabels'
                    )
                    if (plse.length > 0) {
                        if (plse[0].children.length == 3) {
                            const pl = plse[0].children[1];
                            for (let i = 0; i < pl.children.length; i++) {
                                if (pl.children[i].className.length != 0) {
                                    // 删除节点
                                    pl.children[i].style.display = "none";
                                }
                            }
                        }
                    }
                }
            }
        })

    }, [musicQualityAtom])






    const isLyricPageOpenedAtom = useAtomValue(extensionContext.amllStates.isLyricPageOpenedAtom);
    useEffect(() => {
        if (hasFoundContainer.current) return;
        setContainer(document.getElementById('root'));
        hasFoundContainer.current = true;
    }, []);

    useEffect(() => {
        setIsShowCtrl(isLyricPageOpenedAtom);
    }, [isLyricPageOpenedAtom]);

    // 渲染变换（使用 GSAP 更新 DOM 位置/缩放）
    const render = useCallback(() => {
        const dom = appleRef.current;
        if (!dom) return;
        const state = stateRef.current;
        const baseWidth = parseFloat(getComputedStyle(dom).width);
        const scaleX_val = state.baseScale * (1 + state.stretchX);
        const scaleY_val = state.baseScale * (1 + state.stretchY);
        const centerOffsetX = (baseWidth * state.baseScale * state.stretchX) / 2;
        gsap.set(dom, {
            scaleX: scaleX_val, scaleY: scaleY_val,
            x: centerOffsetX + state.followX,
            y: state.offsetY + state.followY,
        });
    }, []);

    const setOpenState = (animate = true) => {
        const dom = appleRef.current;
        const icon = iconRef.current;
        const ssma = ssmaRef.current;
        if (!dom || !icon || !ssma) return;
        const duration = animate ? 0.8 : 0;
        const ease = "elastic.out(1, 0.75)";
        gsap.killTweensOf([dom, icon, ssma]);
        gsap.to(dom, { duration, height: "8em", "--smooth-corners": "2, 4", ease });
        gsap.to(icon, { duration, y: "2.1em", fill: "#000000", ease });
        gsap.to(ssma, { duration, opacity: 1, ease });
    };

    const setClosedState = (animate = true) => {
        const dom = appleRef.current;
        const icon = iconRef.current;
        const ssma = ssmaRef.current;
        if (!dom || !icon || !ssma) return;
        const duration = animate ? 0.8 : 0;
        const ease = "elastic.out(1, 0.6)";
        gsap.killTweensOf([dom, icon, ssma]);
        gsap.to(dom, { duration, height: "4em", "--smooth-corners": "2, 2", ease });
        gsap.to(icon, { duration, y: "0.1em", fill: "#ffffff", ease });
        gsap.to(ssma, { duration, opacity: 0, ease });
    };

    const renderRef = useRef(render);
    renderRef.current = render;

    // 核心更新逻辑：根据垂直音量值 + 水平滑动值更新 UI 和全局状态
    const updateEffect = useCallback((valVertical, cumHorizontal) => {
        const dom = appleRef.current;
        const ssma = ssmaRef.current;
        if (!dom || !ssma) return;

        const state = stateRef.current;
        let actualV;
        if (valVertical > 1) {
            actualV = 1;
            state.stretchY = (valVertical - 1) * 0.15;
        } else if (valVertical < 0) {
            actualV = 0;
            state.stretchY = Math.abs(valVertical) * 0.15;
        } else {
            actualV = valVertical;
            state.stretchY = 0;
        }

        // 同时更新 vRef 和外部 atom，确保同步
        vRef.current = actualV;
        setVocalGain(actualV);

        // 更新遮罩高度
        ssma.style.height = actualV * 100 + "%";
        ssma.style.y = (1 - actualV) * 100 + "%";

        if (valVertical > 1) {
            const baseHeight = parseFloat(getComputedStyle(dom).height);
            state.offsetY = -state.stretchY * baseHeight * state.baseScale;
        } else {
            state.offsetY = 0;
        }

        state.stretchX = Math.abs(cumHorizontal) * 0.05;
        state.followX = cumHorizontal * 10;
        const verticalOverflow = valVertical > 1 ? valVertical - 1 : valVertical < 0 ? valVertical : 0;
        state.followY = -verticalOverflow * 10;
        render();
    }, [render, setVocalGain]);

    const updateEffectRef = useRef(updateEffect);
    updateEffectRef.current = updateEffect;

    // 外部 volume 变化时的同步（非拖拽状态）
    useEffect(() => {
        if (isClickRef.current) return; // 拖拽中由鼠标控制，不干扰
        if (Math.abs(vRef.current - vocalGain) < 0.001) return;

        vRef.current = vocalGain;
        logicalVRef.current = vocalGain;
        logicalHRef.current = 0;

        updateEffectRef.current(vocalGain, 0); // 更新遮罩高度

        if (vocalGain === 1) {
            setClosedState(true);
        } else {
            setOpenState(true);
        }
    }, [vocalGain]);

    // SVG 注入与事件绑定（只执行一次）
    useEffect(() => {
        if (!container || !(container instanceof HTMLElement)) return;
        const dom = appleRef.current;
        if (!dom) return;
        if (svgInjectedRef.current) return;
        svgInjectedRef.current = true;

        dom.innerHTML = svg;
        const icon = dom.querySelector("use[href='#icon']");
        const ssma = dom.querySelector("#ssma");
        if (!icon || !ssma) return;
        iconRef.current = icon;
        ssmaRef.current = ssma;

        // 初始化视觉效果（根据当前音量，vRef 已与 volume 同步）
        // 在 SVG 注入成功后
        const initV = vRef.current;
        if (initV === 1) {
            setClosedState(false); // 无动画直接设为关闭
        } else {
            setOpenState(false);   // 无动画直接设为打开
        }

        // 事件回调
        const onMouseDown = () => {
            isClickRef.current = true;
            // 关键：从当前真实音量开始累计，避免跳跃
            logicalVRef.current = vRef.current;
            logicalHRef.current = 0;

            // 强制展开控件（无论当前音量是1还是其他）
            setOpenState(true);

            // 弹性缩放效果
            gsap.to(stateRef.current, {
                baseScale: 1.4, duration: 0.8, ease: "elastic.out(1, 0.75)",
                onUpdate: () => renderRef.current(),
            });

            // 重置拉伸和跟随偏移
            stateRef.current.stretchX = 0;
            stateRef.current.stretchY = 0;
            stateRef.current.offsetY = 0;
            stateRef.current.followX = 0;
            stateRef.current.followY = 0;
        };

        const onMouseUp = () => {
            isClickRef.current = false;
            logicalHRef.current = 0;

            // 恢复缩放和变换
            gsap.to(stateRef.current, {
                baseScale: 1, stretchY: 0, stretchX: 0,
                offsetY: 0, followX: 0, followY: 0,
                duration: 0.8, ease: "elastic.out(1, 0.6)",
                onUpdate: () => renderRef.current(),
            });

            // 根据最终音量决定UI形态
            if (vRef.current === 1) {
                setClosedState(true);   // 音量=1 → 收拢为圆形
            } else {
                setOpenState(true);     // 音量≠1 → 保持打开状态
            }
        };

        const onMouseMove = (e) => {
            if (!isClickRef.current) return;
            logicalVRef.current -= e.movementY * 0.01;
            logicalHRef.current += e.movementX * 0.01;
            updateEffectRef.current(logicalVRef.current, logicalHRef.current);
        };

        const onBlur = () => {
            isClickRef.current = false;
        };

        dom.addEventListener("mousedown", onMouseDown);
        window.addEventListener("mouseup", onMouseUp);
        window.addEventListener("mousemove", onMouseMove);
        window.addEventListener("blur", onBlur);

        return () => {
            dom.removeEventListener("mousedown", onMouseDown);
            window.removeEventListener("mouseup", onMouseUp);
            window.removeEventListener("mousemove", onMouseMove);
            window.removeEventListener("blur", onBlur);
        };
    }, [container]);

    if (!container || !(container instanceof HTMLElement)) return null;

    return createPortal(
        <div
            ref={appleRef}
            className="applemusicsingcontrol"
            style={{
                position: "fixed",
                top: isShowCtrl ? "12%" : "120%",
                right: "3%",
                width: "4em",
                height: "4em",
                overflow: "hidden",
                transformOrigin: "top right",
                maskImage: "paint(smooth-corners)",
                "--smooth-corners": "2, 2",
                transition: "top .5s cubic-bezier(.25,1,.5,1)",
                mixBlendMode: "plus-lighter",
                backdropFilter: "blur(1.5px)",
                willChange: "transform, top, backdrop-filter",
                visibility: isShowEleInPlayer ? "visible" : "hidden",
            }}
        />,
        container
    );
}


/**
 * 查找根元素下所有 class 字符串中包含指定所有片段的元素
 * @param {ParentNode|Element} root - 查找的根节点（如 document 或某个元素）
 * @param {...string} classParts - 需要包含的片段（多个片段为“且”关系）
 * @returns {Element[]} 匹配的元素数组
 */
function findElementsByClassContains(root, ...classParts) {
    if (!classParts.length) return [];
    const elements = root.querySelectorAll('*');
    const result = [];
    for (const el of elements) {
        // 获取 class 字符串（兼容 SVG 等）
        let className = '';
        if (el.classList) {
            className = Array.from(el.classList).join(' ');
        } else if (el.className) {
            className = typeof el.className === 'string' ? el.className : el.className.baseVal;
        } else {
            continue;
        }
        // 检查是否所有片段都包含在 class 字符串中
        if (classParts.every(part => className.includes(part))) {
            result.push(el);
        }
    }
    return result;
}


export default MainContext;