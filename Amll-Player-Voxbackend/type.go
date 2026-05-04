package main

type Song struct {
	Duration float32 `json:"duration"`
	//filePath
	FilePath string `json:"filePath"`
	//id
	Id string `json:"id"`
	//songAlbum
	SongAlbum string `json:"songAlbum"`
	//songArtists
	SongArtists string `json:"songArtists"`
	//songName
	SongName string `json:"songName"`
}
