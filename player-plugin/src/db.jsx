

export const db = extensionContext.playerDB

export const GetSongs =async ()=>{
    const songs = await db.table('songs').toArray()
    return songs
}