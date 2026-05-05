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
}

@keyframes fadeIn {
	from {
		opacity: 0;
	}

	to {
		opacity: 1;
	}
}

.top_box {
	width: 100%;
	height: 10vh;
	display: flex;
	justify-content: space-between;
	align-items: center;
    mix-blend-mode: plus-lighter;
}

.top_box_info {
	display: flex;
	flex-direction: column;

}

.top_box_info_title {
	font-size: max(2.3vh, 1.3em);
	color: #fff;
}

.top_box_info_tip {
	font-size: max(1.9vh, 1em);
	color:  rgba(255, 255, 255, 0.5);
}

.top_box_ctrl {
	display: flex;
	align-items: center;
	gap: 2vh;
}

.top_box_ctrl_btn {
	width: 11vh;
	height: 5.5vh;
	background: rgba(255, 255, 255, 0.3);
	mask-image: paint(smooth-corners);
	--smooth-corners: "2, 2";
	display: flex;
	justify-content: center;
	align-items: center;
	transition: all 0.2s ease-in-out;
	color: white;
}

.tbc_active {
	background: rgba(255, 255, 255, 1);
	color: #222;

}

.playlist {
	display: flex;
	flex-direction: column;

}

.playlist_item {
	display: flex;
	width: 100%;
	height: 10vh;
	align-items: center;
	justify-content: space-between;
	box-sizing: border-box;
	padding-top: 1vh;
}

.playlist_item_img {
	min-width: 9vh !important;
	min-height: 9vh !important;
    max-width: 9vh;
    max-height: 9vh;
	background: #ffffff1f;
	border-radius: 1vh;
}

.playlist_item_info {
	width: 100%;
	display: flex;
	justify-content: space-between;
	align-items: center;
	margin-left: 2vh;
	height: 100%;
    box-sizing: border-box;

	border-bottom: #aaaaaa49 1px solid;
}

.playlist_item_info_box {
	display: flex;
	flex-direction: column;
	justify-content: space-between;
	align-items: flex-start;
	gap: 0.5vh;
}

.playlist_item_info_title {
    mix-blend-mode: plus-lighter;
	font-size: max(2.3vh, 1.3em);
    line-height: 2vh;
	color: #fff;
}

.playlist_item_info_tip {
	font-size: max(2vh, 1em);
	color: rgba(255, 255, 255, 0.5);
    mix-blend-mode: plus-lighter;
}

.playlist_item_endicon {
	width: 5vh;
	height: 5vh;
	mask-image: paint(smooth-corners);
	--smooth-corners: "2";
	display: flex;
	justify-content: center;
	align-items: center;

	svg {
		transform: scale(1.2);
	}
}
        `
    document.head.appendChild(style);

    extensionContext.registerComponent("settings", Setting)
    extensionContext.registerComponent("context", MainContext);

    let handlersRegistered = false;
    ConnectBackend((io) => {
        console.log("Connect Backend");
        extensionContext.jotaiStore.set(VoxBackendStates.WsIsConect, true)

        const store = extensionContext.jotaiStore;

        if (!handlersRegistered) {
            handlersRegistered = true;
            io.on("PausedChanged", (d) => {
                console.log("PausedChanged", d);
            })
            io.on("Event", (d) => {
                console.log(`[${d.type}] `, d.message);
                switch (d.type) {
                    case "crossfade_started":

                        store.set(VoxBackendStates.Crossfadeing, true)
                        break;
                    case "track_changed":
                        store.set(VoxBackendStates.Crossfadeing, false)
                        break;
                }
                const prev = store.get(VoxBackendStates.EventLog);
                store.set(VoxBackendStates.EventLog, [
                    { type: d.type, message: d.message, time: Date.now() },
                    ...prev
                ].slice(0, 50));
            })
            io.on("OnTrackChanged", (d) => {
                console.log("OnTrackChanged", d);
                store.set(VoxBackendStates.CurrentTrackId, d.id);
                extensionContext.playerDB.table('songs').get(d.id).then(song => {
                    if (song) {
                        console.log(song);
                        if (song.lyricFormat == "ttml") {
                            console.log(extensionContext.lyric.parseTTML(song.lyric));
                            store.set(
                                extensionContext.amllStates.musicLyricLinesAtom,
                                extensionContext.lyric.parseTTML(song.lyric).lines
                            )
                        }
                        // cover File类型
                        // 生成链接
                        const cover = URL.createObjectURL(new Blob([song.cover], { type: song.type }));
                        console.log(cover);
                        store.set(
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
            io.on("playlist", (d) => {
                store.set(VoxBackendStates.NowPlayList, d);
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
        })
    }, () => {
        extensionContext.jotaiStore.set(VoxBackendStates.WsIsConect, false)
    });
});
extensionContext.addEventListener('extension-unload', function () {
    console.log('unload');
});

