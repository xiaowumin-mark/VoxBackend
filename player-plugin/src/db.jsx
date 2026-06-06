
export const db = extensionContext.playerDB

export const GetSongs =async ()=>{
    const songs = await db.table('songs').toArray()
    return songs
}

export const GetPlaylists = () => db.table('playlists').toArray();

export const GetSongById = (id) => db.table('songs').get(id);

export const ToBackendTrack = (song) => ({
    id: song.id,
    duration: song.duration,
    filePath: song.filePath,
    songAlbum: song.songAlbum,
    songArtists: song.songArtists,
    songName: song.songName,
});

export const GetPlaylistTracks = async (playlistName) => {
    const name = (playlistName ?? '').trim();
    if (!name) return [];

    const lists = await GetPlaylists();
    const playlist = lists.find(pl => pl.name === name);
    if (!playlist || !Array.isArray(playlist.songIds)) return [];

    const songs = await Promise.all(playlist.songIds.map(id => GetSongById(id)));
    return songs.filter(Boolean).map(ToBackendTrack);
};

export const SyncPlaylistToBackend = async (io, playlistName) => {
    if (!io) return { synced: false, count: 0 };

    const tracks = await GetPlaylistTracks(playlistName);
    io.emit("rm-all-songs");
    io.emit("songs", tracks);
    return { synced: true, count: tracks.length };
};
