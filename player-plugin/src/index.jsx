import Setting from './setting';
import MainContext from './context';
import LoadCssHoudini from './smooth-corners';
import { ConnectBackend } from './ws';
import { VoxBackendStates } from './store';
import { GetSongs } from './db';
LoadCssHoudini();
extensionContext.addEventListener('extension-load', function () {
    console.log('load');
    // 创建一个style 到meta
    const style = document.createElement('style');
    style.textContent = `
    @keyframes shine-move {
         0% {
            mask-position: 100% 0;
            -webkit-mask-position: 100% 0;
        }
        
        100% {
            mask-position: -100% 0;
            -webkit-mask-position: -100% 0;
        }
    @keyframes fadeIn {
            from { opacity: 0; }
            to { opacity: 1; }
        }    
        `
    document.head.appendChild(style);

    extensionContext.registerComponent("settings", Setting)
    extensionContext.registerComponent("context", MainContext);

    let handlersRegistered = false;
    ConnectBackend((io) => {
        console.log("Connect Backend");
        extensionContext.jotaiStore.set(VoxBackendStates.WsIsConect, true)

        if (!handlersRegistered) {
            handlersRegistered = true;
            io.on("PausedChanged", (d) => {
                console.log("PausedChanged", d);
            })
            io.on("Event", (d) => {
                console.log(`[${d.type}] `, d.message);
                switch (d.type) {
                    case "crossfade_started":

                        extensionContext.jotaiStore.set(VoxBackendStates.Crossfadeing, true)
                        break;
                    case "track_changed":
                        extensionContext.jotaiStore.set(VoxBackendStates.Crossfadeing, false)
                        break;
                }
            })
            io.on("OnTrackChanged", (d) => {
                console.log("OnTrackChanged", d);
                extensionContext.playerDB.table('songs').get(d.id).then(song => {
                    if (song) {
                        console.log(song);
                        if (song.lyricFormat == "ttml") {
                            console.log(extensionContext.lyric.parseTTML(song.lyric));
                            extensionContext.jotaiStore.set(
                                extensionContext.amllStates.isLyricPageOpenedAtom,
                                true
                            )
                            extensionContext.jotaiStore.set(
                                extensionContext.amllStates.musicLyricLinesAtom,
                                extensionContext.lyric.parseTTML(song.lyric).lines
                            )
                        }
                        // cover File类型
                        // 生成链接
                        const cover = URL.createObjectURL(new Blob([song.cover], { type: song.type }));
                        console.log(cover);
                        extensionContext.jotaiStore.set(
                            extensionContext.amllStates.musicCoverAtom,
                            cover
                        )
                    }
                })
            })
            io.on("OnState", (d) => {
                // console.log("OnState", d);
            })
            io.on("OnVolumeChanged", (d) => {
                console.log("OnVolumeChanged", d);
            })
            io.on("OnVocalChanged", (d) => {
                console.log("OnVocalChanged", d);
                // id 查找歌曲
            })
            io.on("cfg", (d) => {
                console.log("cfg", d);
                extensionContext.jotaiStore.set(VoxBackendStates.MasterVolume, d.MasterVolume)
                extensionContext.jotaiStore.set(VoxBackendStates.Crossfade, d.Crossfade)
                extensionContext.jotaiStore.set(VoxBackendStates.DSPMode, d.DSPMode)
                extensionContext.jotaiStore.set(VoxBackendStates.VocalGain, d.VocalGain)
                extensionContext.jotaiStore.set(VoxBackendStates.VocalGainRamp, d.VocalGainRamp)
            })
        }

        GetSongs().then(songs => {
            io.emit("rm-all-songs")
            io.emit("songs", songs.map(song => {
                return {
                    id: song.id,
                    duration: song.duration,
                    filePath: song.filePath,
                    songAlbum: song.songAlbum,
                    songArtists: song.songArtists,
                    songName: song.songName
                }
            }))
            io.emit("need-cfg")
        })
    }, () => {
        extensionContext.jotaiStore.set(VoxBackendStates.WsIsConect, false)
    });
});
extensionContext.addEventListener('extension-unload', function () {
    console.log('unload');
});

