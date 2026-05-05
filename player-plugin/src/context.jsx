// 顶部导入（需确保环境中已加载 GSAP 和 figma-squircle）
import gsap from "gsap";
import { VoxBackendStates } from "./store";
import { GetIo } from "./ws";
import PlaylistOverlay from './overlay';
import { GetPlaylists, GetSongById } from './db';
const { useEffect, useRef, useState, useCallback, useLayoutEffect } = React;
const { createPortal } = ReactDOM;
// Jotai
const { atom, useAtom, useAtomValue, useSetAtom } = Jotai;

const svg = `<svg width="100%" height="100%" preserveAspectRatio="xMidYMid slice" xmlns="http://www.w3.org/2000/svg">
     <defs>
                <mask id="hole">
                    <rect width="100%" height="100%" fill="rgba(255,255,255,0.3)" />
                    <rect fill="white" id="ssma" y="0%" opacity="0" style=" width:100%; height:100%" />
                    <use href="#icon" x="1.7vh" y="0%" width="2.3vh" height="100%" fill="black" style="transform-origin: center bottom; overflow: visible;"/>
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


const playlistSvg = {
    outline: `<svg xmlns="http://www.w3.org/2000/svg" width="64" height="64" viewBox="0 0 64 64" fill="none">
<path d="M64 36.8C64 47.275 64 52.5125 61.6422 56.36C60.323 58.5129 58.5129 60.323 56.36 61.6422C52.5125 64 47.275 64 36.8 64H27.2C16.725 64 11.4875 64 7.64002 61.6422C5.48714 60.323 3.67705 58.5129 2.35776 56.36C0 52.5125 0 47.275 0 36.8V27.2C0 16.725 0 11.4875 2.35776 7.64002C3.67705 5.48714 5.48714 3.67705 7.64002 2.35776C11.4875 0 16.725 0 27.2 0H36.8C47.275 0 52.5125 0 56.36 2.35776C58.5129 3.67705 60.323 5.48714 61.6422 7.64002C64 11.4875 64 16.725 64 27.2V36.8ZM14.4805 41.0977C13.6602 41.0977 12.957 41.3841 12.3711 41.957C11.7982 42.5299 11.5117 43.2266 11.5117 44.0469C11.5117 44.8672 11.7982 45.5638 12.3711 46.1367C12.957 46.7227 13.6602 47.0156 14.4805 47.0156C15.2878 47.0156 15.9844 46.7227 16.5703 46.1367C17.1562 45.5638 17.4492 44.8672 17.4492 44.0469C17.4492 43.2266 17.1562 42.5299 16.5703 41.957C15.9844 41.3841 15.2878 41.0977 14.4805 41.0977ZM23.9922 41.9766C23.4062 41.9766 22.9115 42.1784 22.5078 42.582C22.1172 42.9857 21.9219 43.474 21.9219 44.0469C21.9219 44.6328 22.1172 45.1276 22.5078 45.5312C22.9115 45.9219 23.4062 46.1172 23.9922 46.1172H50.418C50.9909 46.1172 51.4792 45.9219 51.8828 45.5312C52.2865 45.1276 52.4883 44.6328 52.4883 44.0469C52.4883 43.474 52.2865 42.9857 51.8828 42.582C51.4792 42.1784 50.9909 41.9766 50.418 41.9766H23.9922ZM14.4805 28.9492C13.6602 28.9492 12.957 29.2422 12.3711 29.8281C11.7982 30.4141 11.5117 31.1107 11.5117 31.918C11.5117 32.7253 11.7982 33.4219 12.3711 34.0078C12.957 34.5938 13.6602 34.8867 14.4805 34.8867C15.2878 34.8867 15.9844 34.5938 16.5703 34.0078C17.1562 33.4219 17.4492 32.7253 17.4492 31.918C17.4492 31.1107 17.1562 30.4141 16.5703 29.8281C15.9844 29.2422 15.2878 28.9492 14.4805 28.9492ZM23.9922 29.8672C23.4062 29.8672 22.9115 30.069 22.5078 30.4727C22.1172 30.8632 21.9219 31.3451 21.9219 31.918C21.9219 32.4909 22.1172 32.9792 22.5078 33.3828C22.9115 33.7865 23.4062 33.9883 23.9922 33.9883H50.418C50.9909 33.9883 51.4792 33.793 51.8828 33.4023C52.2865 32.9987 52.4883 32.5039 52.4883 31.918C52.4883 31.3451 52.2865 30.8632 51.8828 30.4727C51.4792 30.069 50.9909 29.8672 50.418 29.8672H23.9922ZM14.4805 16.8398C13.6602 16.8399 12.957 17.1263 12.3711 17.6992C11.7982 18.2721 11.5117 18.9688 11.5117 19.7891C11.5117 20.6094 11.7982 21.306 12.3711 21.8789C12.957 22.4518 13.6602 22.7383 14.4805 22.7383C15.2878 22.7383 15.9844 22.4518 16.5703 21.8789C17.1562 21.306 17.4492 20.6094 17.4492 19.7891C17.4492 18.9688 17.1562 18.2721 16.5703 17.6992C15.9844 17.1263 15.2878 16.8398 14.4805 16.8398ZM23.9922 17.7188C23.4062 17.7188 22.9115 17.9206 22.5078 18.3242C22.1172 18.7279 21.9219 19.2161 21.9219 19.7891C21.9219 20.375 22.1172 20.8698 22.5078 21.2734C22.9115 21.6641 23.4062 21.8594 23.9922 21.8594H50.418C50.9909 21.8594 51.4792 21.6641 51.8828 21.2734C52.2865 20.8698 52.4883 20.375 52.4883 19.7891C52.4883 19.2161 52.2865 18.7279 51.8828 18.3242C51.4792 17.9206 50.9909 17.7188 50.418 17.7188H23.9922Z" fill="currentColor"/>
</svg>`,
    inline: `<svg width="64" height="64" viewBox="0 0 64 64" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M23.9922 21.8594C23.4062 21.8594 22.9115 21.6641 22.5078 21.2734C22.1172 20.8698 21.9219 20.375 21.9219 19.7891C21.9219 19.2161 22.1172 18.7279 22.5078 18.3242C22.9115 17.9206 23.4062 17.7188 23.9922 17.7188H50.418C50.9909 17.7188 51.4792 17.9206 51.8828 18.3242C52.2865 18.7279 52.4883 19.2161 52.4883 19.7891C52.4883 20.375 52.2865 20.8698 51.8828 21.2734C51.4792 21.6641 50.9909 21.8594 50.418 21.8594H23.9922ZM23.9922 33.9883C23.4062 33.9883 22.9115 33.7865 22.5078 33.3828C22.1172 32.9792 21.9219 32.4909 21.9219 31.918C21.9219 31.3451 22.1172 30.8633 22.5078 30.4727C22.9115 30.069 23.4062 29.8672 23.9922 29.8672H50.418C50.9909 29.8672 51.4792 30.069 51.8828 30.4727C52.2865 30.8633 52.4883 31.3451 52.4883 31.918C52.4883 32.5039 52.2865 32.9987 51.8828 33.4023C51.4792 33.793 50.9909 33.9883 50.418 33.9883H23.9922ZM23.9922 46.1172C23.4062 46.1172 22.9115 45.9219 22.5078 45.5312C22.1172 45.1276 21.9219 44.6328 21.9219 44.0469C21.9219 43.474 22.1172 42.9857 22.5078 42.582C22.9115 42.1784 23.4062 41.9766 23.9922 41.9766H50.418C50.9909 41.9766 51.4792 42.1784 51.8828 42.582C52.2865 42.9857 52.4883 43.474 52.4883 44.0469C52.4883 44.6328 52.2865 45.1276 51.8828 45.5312C51.4792 45.9219 50.9909 46.1172 50.418 46.1172H23.9922ZM14.4805 22.7383C13.6602 22.7383 12.957 22.4518 12.3711 21.8789C11.7982 21.306 11.5117 20.6094 11.5117 19.7891C11.5117 18.9688 11.7982 18.2721 12.3711 17.6992C12.957 17.1263 13.6602 16.8398 14.4805 16.8398C15.2878 16.8398 15.9844 17.1263 16.5703 17.6992C17.1562 18.2721 17.4492 18.9688 17.4492 19.7891C17.4492 20.6094 17.1562 21.306 16.5703 21.8789C15.9844 22.4518 15.2878 22.7383 14.4805 22.7383ZM14.4805 34.8867C13.6602 34.8867 12.957 34.5938 12.3711 34.0078C11.7982 33.4219 11.5117 32.7253 11.5117 31.918C11.5117 31.1107 11.7982 30.4141 12.3711 29.8281C12.957 29.2422 13.6602 28.9492 14.4805 28.9492C15.2878 28.9492 15.9844 29.2422 16.5703 29.8281C17.1562 30.4141 17.4492 31.1107 17.4492 31.918C17.4492 32.7253 17.1562 33.4219 16.5703 34.0078C15.9844 34.5938 15.2878 34.8867 14.4805 34.8867ZM14.4805 47.0156C13.6602 47.0156 12.957 46.7227 12.3711 46.1367C11.7982 45.5638 11.5117 44.8672 11.5117 44.0469C11.5117 43.2266 11.7982 42.5299 12.3711 41.957C12.957 41.3841 13.6602 41.0977 14.4805 41.0977C15.2878 41.0977 15.9844 41.3841 16.5703 41.957C17.1562 42.5299 17.4492 43.2266 17.4492 44.0469C17.4492 44.8672 17.1562 45.5638 16.5703 46.1367C15.9844 46.7227 15.2878 47.0156 14.4805 47.0156Z" fill="currentColor"></path></svg>`
}
function createTopic() {
    const div = document.createElement('span');
    div.innerHTML = "自动过渡";
    div.style.width = "100px";
    div.style.fontSize = "max(1., 0.8em)";
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
    div.style.animation = "shine-move 2s ease infinite,fadeIn 1s ease-out";
    // 阴影
    div.style.textShadow = "0 0 3px rgba(255,255,255,0.5)";

    div.style.mixBlendMode = "plus-lighter";
    // padding 0
    div.style.padding = "0";
    return div;
}
function MainContext() {
    const [container, setContainer] = useState(null);
    const hasFoundContainer = useRef(false);
    const [vocalGain, setVocalGain] = useAtom(VoxBackendStates.VocalGain);
    const [isShowEleInPlayer, setIsShowEleInPlayer] = useAtom(VoxBackendStates.ShowEleInPlayer);
    const [isShowPlaylist, setIsShowPlaylist] = useAtom(VoxBackendStates.IsShowPlaylist);

    const appleRef = useRef(null);
    const iconRef = useRef(null);
    const ssmaRef = useRef(null);
    const [isShowCtrl, setIsShowCtrl] = useState(false);
    const [lyricContainer, setLyricContainer] = useState(null);
    const playlistOverlayRef = useRef(null);
    const svgInjectedRef = useRef(false);
    const isShowPlaylistRef = useRef(isShowPlaylist);
    isShowPlaylistRef.current = isShowPlaylist;
    const playlistBtnRef = useRef(null);
    const observerRef = useRef(null);

    // 核心引用：音量值、拖拽累计值、拖拽状态
    const vRef = useRef(vocalGain);
    const logicalVRef = useRef(vocalGain);
    const logicalHRef = useRef(0);
    const isClickRef = useRef(false);
    const stateRef = useRef({
        baseScale: 1, stretchY: 0, stretchX: 0,
        offsetY: 0, followX: 0, followY: 0,
    });



    const [masterVolume, setMasterVolume] = useAtom(VoxBackendStates.MasterVolume);
    useEffect(() => {
        GetIo()?.emit("volume", masterVolume)
    }, [masterVolume]);

    const [vocalGainRamp, setVocalGainRamp] = useAtom(VoxBackendStates.VocalGainRamp);
    useEffect(() => {

        GetIo()?.emit("vocal-gain-ramp", vocalGainRamp)
    }, [vocalGainRamp]);

    const [crossfade, setCrossfade] = useAtom(VoxBackendStates.Crossfade);
    const [isCrossfade] = useAtom(VoxBackendStates.IsCrossfade);
    useEffect(() => {
        GetIo()?.emit("crossfade", isCrossfade ? crossfade : 0)
    }, [crossfade, isCrossfade]);

    //dsp
    const [dspMode, setDspMode] = useAtom(VoxBackendStates.DSPMode);
    useEffect(() => {
        GetIo()?.emit("dsp", dspMode)
    }, [dspMode]);

    const [needShufflePlay, setNeedShufflePlay] = useAtom(VoxBackendStates.NeedShufflePlay);
    const [nowPlayListName] = useAtom(VoxBackendStates.NowPlayListName);
    const [nowPlayList] = useAtom(VoxBackendStates.NowPlayList);
    const [currentTrackId] = useAtom(VoxBackendStates.CurrentTrackId);

    useEffect(() => {
        if (!container) return;
        const controls = findElementsByClassContains(document.getElementById('root'), 'bottomControls');
        if (controls.length === 0) return;
        const btn = controls[0].querySelector('button');
        if (!btn) return;
        playlistBtnRef.current = btn;

        btn.innerHTML = isShowPlaylist ? playlistSvg.outline : playlistSvg.inline;
        btn.setAttribute('data-vox-pl', isShowPlaylist ? 'outline' : 'inline');

        const observer = new MutationObserver(() => {
            if (!playlistBtnRef.current) return;
            const key = isShowPlaylistRef.current ? 'outline' : 'inline';
            if (playlistBtnRef.current.getAttribute('data-vox-pl') !== key) {
                playlistBtnRef.current.innerHTML = playlistSvg[key];
                playlistBtnRef.current.setAttribute('data-vox-pl', key);
            }
        });
        observer.observe(btn, { childList: true, characterData: true, subtree: true });
        observerRef.current = observer;

        const onClick = () => {
            setIsShowPlaylist(!isShowPlaylistRef.current);
        };
        btn.addEventListener('click', onClick);

        return () => {
            observer.disconnect();
            observerRef.current = null;
            btn.removeEventListener('click', onClick);
            playlistBtnRef.current = null;
        };
    }, [container]);

    useEffect(() => {
        const btn = playlistBtnRef.current;
        if (!btn) return;
        btn.innerHTML = isShowPlaylist ? playlistSvg.outline : playlistSvg.inline;
        btn.setAttribute('data-vox-pl', isShowPlaylist ? 'outline' : 'inline');
    }, [isShowPlaylist, container]);

    useEffect(() => {
        if (!container) return;
        const lyric = findElementsByClassContains(document.getElementById('root'), '_lyric_')[0];
        if (!lyric) return;
        lyric.style.position = 'relative';
        if (!lyricContainer) {
            setLyricContainer(lyric);
            window.voxPlaylist = { getLayer: () => playlistOverlayRef.current };
        }
        if (isShowPlaylist) {
            lyric.style.maskImage = 'linear-gradient(#0000,#000 0% 90%,#0000)';
            lyric.style.webkitMaskImage = 'linear-gradient(#0000,#000 0% 90%,#0000)';
            //mix-blend-mode: normal
            lyric.style.mixBlendMode = 'normal';
        } else {
            lyric.style.maskImage = '';
            lyric.style.webkitMaskImage = '';
            lyric.style.mixBlendMode = 'plus-lighter';
        }
        const player = findElementsByClassContains(lyric, 'amll-lyric-player')[0];
        if (player) {
            if (isShowPlaylist) {
                player.style.transition = 'opacity 0.35s cubic-bezier(0.4,0,0.2,1), transform 0.35s cubic-bezier(0.4,0,0.2,1)';
                player.style.opacity = '0';
                player.style.transform = 'scale(0.97)';
                player.style.pointerEvents = 'none';
            } else {
                player.style.opacity = '';
                player.style.transform = '';
                player.style.pointerEvents = '';
                const onEnd = () => {
                    player.style.transition = '';
                    player.removeEventListener('transitionend', onEnd);
                };
                player.addEventListener('transitionend', onEnd);
            }
        }
        const apple = appleRef.current;
        if (apple) {
            if (isShowPlaylist) {
                apple.style.transition = 'opacity 0.35s cubic-bezier(0.4,0,0.2,1), transform 0.35s cubic-bezier(0.4,0,0.2,1)';
                apple.style.opacity = '0';
                apple.style.transform = 'scale(0.9)';
                apple.style.pointerEvents = 'none';
            } else {
                apple.style.opacity = '';
                apple.style.transform = '';
                apple.style.pointerEvents = '';
                const onEnd = () => {
                    apple.style.transition = '';
                    apple.removeEventListener('transitionend', onEnd);
                };
                apple.addEventListener('transitionend', onEnd);
            }
        }
    }, [isShowPlaylist, container]);

    useEffect(() => {
        if (!container || !nowPlayListName) return;
        GetPlaylists().then(lists => {
            const pl = lists.find(l => l.name === nowPlayListName);
            if (!pl) return;
            Promise.all(pl.songIds.map(id => GetSongById(id))).then(songs => {
                const tracks = songs.filter(Boolean).map(s => ({
                    id: s.id,
                    duration: s.duration,
                    filePath: s.filePath,
                    songAlbum: s.songAlbum,
                    songArtists: s.songArtists,
                    songName: s.songName,
                }));
                GetIo()?.emit("rm-all-songs");
                GetIo()?.emit("songs", tracks);
            });
        });
    }, [nowPlayListName, container]);

    useEffect(() => {
        if (!needShufflePlay || nowPlayList.length < 2) return;
        GetIo()?.emit("shuffle-upcoming");
        setNeedShufflePlay(false);
    }, [needShufflePlay]);

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
        gsap.to(dom, { duration, height: "11vh", "--smooth-corners": "2, 4", ease });
        gsap.to(icon, { duration, y: "2.8vh", fill: "#000000", ease });
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
        gsap.to(dom, { duration, height: "6vh", "--smooth-corners": "2, 2", ease });
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
        GetIo()?.emit("vocal-gain", actualV);

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

    return (
        <>
            <PlaylistOverlay ref={playlistOverlayRef} lyricContainer={lyricContainer} isShowPlaylist={isShowPlaylist} />
            {createPortal(
                <div
                    ref={appleRef}
                    className="applemusicsingcontrol"
                    style={{
                        position: "fixed",
                        top: isShowCtrl ? "12%" : "120%",
                        right: "3vw",
                        width: "6vh",
                        height: "6vh",
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
            )}
        </>
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