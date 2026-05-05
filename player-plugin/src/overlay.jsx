const { forwardRef, useState, useEffect } = React;
const { createPortal } = ReactDOM;
const { useAtom } = Jotai;
import { VoxBackendStates } from './store';
import { GetSongById } from './db';

const PlaylistOverlay = forwardRef(({ lyricContainer, isShowPlaylist }, ref) => {
    if (!lyricContainer) return null;

    const [isCrossfade, setCrossfade] = useAtom(VoxBackendStates.IsCrossfade);
    const [nowPlayList] = useAtom(VoxBackendStates.NowPlayList);
    const [nowPlayListName] = useAtom(VoxBackendStates.NowPlayListName);
    const [needShufflePlay, setNeedShufflePlay] = useAtom(VoxBackendStates.NeedShufflePlay);
    const [currentTrackId] = useAtom(VoxBackendStates.CurrentTrackId);

    const currentIdx = nowPlayList.findIndex(t => t.id === currentTrackId);
    const upcomingTracks = currentIdx >= 0 ? nowPlayList.slice(currentIdx + 1) : nowPlayList;

    const [covers, setCovers] = useState({});
    useEffect(() => {
        const ids = upcomingTracks.map(t => t.id).filter(Boolean);
        if (!ids.length) return;
        const prevCovers = covers;
        Promise.all(ids.map(id => GetSongById(id))).then(songs => {
            const map = {};
            songs.filter(Boolean).forEach(s => {
                if (s.cover) {
                    map[s.id] = URL.createObjectURL(new Blob([s.cover], { type: s.type }));
                }
            });
            setCovers(map);
        });
        return () => {
            Object.values(prevCovers).forEach(u => {
                try { URL.revokeObjectURL(u); } catch { }
            });
        };
    }, [nowPlayList, currentTrackId]);

    return createPortal(
        <div
            id="vox-playlist-overlay"
            ref={ref}
            style={{
                position: 'absolute', top: 0, width: '95%', height: '100%',
                zIndex: 999,
                pointerEvents: isShowPlaylist ? 'auto' : 'none',
                opacity: isShowPlaylist ? 1 : 0,
                transform: isShowPlaylist ? 'scale(1)' : 'scale(0.9)',
                transition: 'opacity 0.35s cubic-bezier(0.4,0,0.2,1), transform 0.35s cubic-bezier(0.4,0,0.2,1)',
                mixBlendMode: 'normal',
            }}
        >
            <div class="top_box">
                <div class="top_box_info">
                    <div class="top_box_info_title">继续播放</div>
                    <div class="top_box_info_tip">来自 {nowPlayListName}</div>
                </div>
                <div class="top_box_ctrl">
                    <div className={`top_box_ctrl_btn ${needShufflePlay ? 'tbc_active' : ''}`} onClick={() => setNeedShufflePlay(!needShufflePlay)}>
                        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" width="2.4vh">
                            <path style={{ fill: 'currentColor' }} d="M18.6,6.62C21.58,6.62 24,9 24,12C24,14.96 21.58,17.37 18.6,17.37C17.15,17.37 15.8,16.81 14.78,15.8L12,13.34L9.17,15.85C8.2,16.82 6.84,17.38 5.4,17.38C2.42,17.38 0,14.96 0,12C0,9.04 2.42,6.62 5.4,6.62C6.84,6.62 8.2,7.18 9.22,8.2L12,10.66L14.83,8.15C15.8,7.18 17.16,6.62 18.6,6.62M7.8,14.39L10.5,12L7.84,9.65C7.16,8.97 6.31,8.62 5.4,8.62C3.53,8.62 2,10.13 2,12C2,13.87 3.53,15.38 5.4,15.38C6.31,15.38 7.16,15.03 7.8,14.39M16.2,9.61L13.5,12L16.16,14.35C16.84,15.03 17.7,15.38 18.6,15.38C20.47,15.38 22,13.87 22,12C22,10.13 20.47,8.62 18.6,8.62C17.69,8.62 16.84,8.97 16.2,9.61Z" />
                        </svg>
                    </div>
                    <div className={`top_box_ctrl_btn ${isCrossfade ? 'tbc_active' : ''}`} onClick={() => setCrossfade(!isCrossfade)}>
                        <svg xmlns="http://www.w3.org/2000/svg" width="2.4vh" viewBox="0 0 61 36" fill="none">
                            <path d="M43 0C52.9411 0 61 8.05887 61 18C61 27.9411 52.9411 36 43 36C33.0589 36 25 27.9411 25 18C25 8.05887 33.0589 0 43 0ZM43 11C39.134 11 36 14.134 36 18C36 21.866 39.134 25 43 25C46.866 25 50 21.866 50 18C50 14.134 46.866 11 43 11Z" style={{ fill: 'currentColor' }} />
                            <path d="M18 0C27.9411 0 36 8.05887 36 18C36 27.9411 27.9411 36 18 36C8.05887 36 0 27.9411 0 18C0 8.05887 8.05887 0 18 0ZM18 4C10.268 4 4 10.268 4 18C4 25.732 10.268 32 18 32C25.732 32 32 25.732 32 18C32 10.268 25.732 4 18 4Z" style={{ fill: 'currentColor' }} />
                        </svg>
                    </div>
                </div>
            </div>

            <div class="playlist" style={{ overflowY: 'auto', msOverflowStyle: 'none', scrollbarWidth: 'none', height: '100%',opacity: 0.9, }}>
                <style>{`#vox-playlist-overlay .playlist::-webkit-scrollbar{display:none}`}</style>
                {upcomingTracks.length === 0 ? (
                    <div style={{ color: '#aaa', padding: '4vh', textAlign: 'center', fontSize: 'max(2vh,1em)' }}>
                        没有更多歌曲
                    </div>
                ) : (
                    upcomingTracks.map((track, i) => (
                        <div key={`${track.id}-${i}`} class="playlist_item">
                            <div class="playlist_item_img">
                                {covers[track.id] && <img src={covers[track.id]} alt="" style={{ width: '100%', height: '100%', objectFit: 'cover', borderRadius: '1vh' }} />}
                            </div>
                            <div class="playlist_item_info">
                                <div class="playlist_item_info_box">
                                    <div class="playlist_item_info_title">{track.songName}</div>
                                    <div class="playlist_item_info_tip">{track.songArtists}</div>
                                </div>
                                <div class="playlist_item_endicon">
                                    <svg width="24" height="24" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                                        <rect x="4" y="7" width="17" height="1" rx="0.5" fill="white" fill-opacity="0.5" />
                                        <rect x="4" y="11" width="17" height="1" rx="0.5" fill="white" fill-opacity="0.5" />
                                        <rect x="4" y="15" width="17" height="1" rx="0.5" fill="white" fill-opacity="0.5" />
                                    </svg>

                                </div>
                            </div>
                        </div>
                    ))
                )}
            </div>
        </div>,
        lyricContainer
    );
});

export default PlaylistOverlay;
