// 顶部导入（需确保环境中已加载 GSAP 和 figma-squircle）
import gsap from "gsap"; // 或从 CDN 的 ES module 导入
import { VoxBackendStates } from "./store";
const { useEffect, useRef, useState, useCallback } = React;
const { createPortal } = ReactDOM;
//Jotai
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
               
            </defs>

            <rect width="100%" height="100%" fill="white" mask="url(#hole)" />
  </svg>`;

function MainContext() {
    const [container, setContainer] = useState(null);
    const hasFoundContainer = useRef(false);
    const [volume, setVolume] = useAtom(VoxBackendStates.Volume);

    // 只保留容器 div 的 ref，SVG 内部完全用原生 DOM 管理
    const appleRef = useRef(null);
    const iconRef = useRef(null);
    const ssmaRef = useRef(null);
    const [isShowCtrl, setIsShowCtrl] = useState(false);

    // 标记 SVG 是否已经原生注入过
    const svgInjectedRef = useRef(false);

    const vRef = useRef(1);
    const logicalVRef = useRef(1);
    const logicalHRef = useRef(0);
    const isClickRef = useRef(false);
    const singScaleRef = useRef(null);
    const stateRef = useRef({
        baseScale: 1, stretchY: 0, stretchX: 0,
        offsetY: 0, followX: 0, followY: 0,
    });
    //isLyricPageOpenedAtom
    const isLyricPageOpenedAtom = useAtomValue(extensionContext.amllStates.isLyricPageOpenedAtom);
    useEffect(() => {
        if (hasFoundContainer.current) return;
        setContainer(document.getElementById('root'));
        hasFoundContainer.current = true;
    }, []);

    useEffect(() => { 
        setIsShowCtrl(isLyricPageOpenedAtom);
    }, [isLyricPageOpenedAtom]);

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

    const renderRef = useRef(render);
    renderRef.current = render;

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
        vRef.current = actualV;
        setVolume(actualV);
        
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
    }, [render]);

    const updateEffectRef = useRef(updateEffect);
    updateEffectRef.current = updateEffect;

    // ✅ 关键 effect：只在 container 就绪后跑一次，原生注入 SVG
    useEffect(() => {
        if (!container || !(container instanceof HTMLElement)) return;
        const dom = appleRef.current;
        if (!dom) return;

        // ✅ 防止重复注入
        if (svgInjectedRef.current) return;
        svgInjectedRef.current = true;

        // 原生插入 SVG，完全绕开 React diff
        dom.innerHTML = svg;

        // 现在可以安全查询
        const icon = dom.querySelector("use[href='#icon']");
        const ssma = dom.querySelector("#ssma");
        if (!icon || !ssma) return;

        iconRef.current = icon;
        ssmaRef.current = ssma;

        // 创建动画
        
        updateEffectRef.current(logicalVRef.current, logicalHRef.current); 
        // 初始化
        const v = vRef.current;
        if (v === 1) {
            gsap.set(dom, { height: "4em", "--smooth-corners": "2, 2" });
            gsap.set(icon, { y: "0em", fill: "#ffffff" });
            gsap.set(ssma, { opacity: 0 });
        } else {
            gsap.set(icon, { y: "2.1em", fill: "#000000" });
            gsap.set(ssma, { opacity: 1 });
            gsap.set(dom, { height: "8em", "--smooth-corners": "2, 4" });

        }

        // 事件绑定
        const onMouseDown = () => {
            isClickRef.current = true;
            gsap.to(stateRef.current, {
                baseScale: 1.4, duration: 0.8, ease: "elastic.out(1, 0.75)",
                onUpdate: () => renderRef.current(),
            });
            gsap.to(icon, { duration: 0.8, y: "2.1em", fill: "#000000", ease: "elastic.out(1, 0.75)" });
            gsap.to(ssma, { duration: 0.8, opacity: 1 });
            gsap.to(dom, { height: "8em", duration: 0.8, "--smooth-corners": "2, 4", ease: "elastic.out(1, 0.75)" });
            if (vRef.current === 1) gsap.to(ssma, { duration: 0.5, opacity: 1 });
            stateRef.current.stretchX = 0;
            stateRef.current.stretchY = 0;
            stateRef.current.offsetY = 0;
            stateRef.current.followX = 0;
            stateRef.current.followY = 0;
            logicalHRef.current = 0;
        };

        const onMouseUp = () => {
            isClickRef.current = false;
            logicalVRef.current = vRef.current;
            logicalHRef.current = 0;
            gsap.to(stateRef.current, {
                baseScale: 1, stretchY: 0, stretchX: 0,
                offsetY: 0, followX: 0, followY: 0,
                duration: 0.8, ease: "elastic.out(1, 0.6)",
                onUpdate: () => renderRef.current(),
            });
            if (vRef.current === 1) {
                gsap.to(dom, { height: "4em", duration: 0.8, "--smooth-corners": "2, 2", ease: "elastic.out(1, 0.6)" });
                gsap.to(icon, { y: "0.1em", duration: 0.8, fill: "#ffffff", ease: "elastic.out(1, 0.6)" });
                gsap.to(ssma, { duration: 0.8, opacity: 0 });

            }
        };

        const onMouseMove = (e) => {
            if (!isClickRef.current) return;
            logicalVRef.current -= e.movementY * 0.01;
            logicalHRef.current += e.movementX * 0.01;
            updateEffectRef.current(logicalVRef.current, logicalHRef.current);
        };

        const onBlur = () => { isClickRef.current = false; };

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
                top: isShowCtrl ? "12%" : "120%", right: "3%",
                width: "4em", height: "4em",
                overflow: "hidden",
                transformOrigin: "top right",
                maskImage: "paint(smooth-corners)",
                "--smooth-corners": "2, 2",
                transition: "top .5s cubic-bezier(.25,1,.5,1)",
                "mix-blend-mode": "plus-lighter",
                backdropFilter: "blur(1.5px)",
                willChange: "transform, top,backdrop-filter",
            }}
        // ✅ 不再用 dangerouslySetInnerHTML，内容由原生 DOM 管理
        />,
        container
    );
}

// 辅助函数（保持不变）
function getEleByClassContains(ele, className) {
    if (!ele || !ele.children) return undefined;
    for (let i = 0; i < ele.children.length; i++) {
        const elementClassName = ele.children[i].className?.baseVal || ele.children[i].className;
        if (elementClassName && elementClassName.includes(className)) {
            return ele.children[i];
        }
    }
    return undefined;
}

export default MainContext;