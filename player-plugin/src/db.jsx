
export const db = extensionContext.playerDB

export const GetSongs =async ()=>{
    const songs = await db.table('songs').toArray()
    return songs
}

export const GetPlaylists = () => db.table('playlists').toArray();

export const GetSongById = (id) => db.table('songs').get(id);