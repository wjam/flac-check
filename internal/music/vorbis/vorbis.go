package vorbis

const (
	AlbumTag          Tag = "ALBUM"
	AlbumArtistTag    Tag = "ALBUMARTIST"
	ArtistTag         Tag = "ARTIST"
	ArtistSortTag     Tag = "ARTISTSORT"
	DateTag           Tag = "DATE"
	DiscNumberTag     Tag = "DISCNUMBER"
	GenreTag          Tag = "GENRE"
	TitleTag          Tag = "TITLE"
	TrackNumberTag    Tag = "TRACKNUMBER"
	TrackTotalTag     Tag = "TRACKTOTAL"
	LyricsTag         Tag = "LYRICS"
	UnsyncedLyricsTag Tag = "UNSYNCEDLYRICS"

	MusicBrainzAlbumIDTag       Tag = "MUSICBRAINZ_ALBUMID"
	MusicBrainzDiscIDTag        Tag = "MUSICBRAINZ_DISCID"
	MusicBrainzAlbumArtistIDTag Tag = "MUSICBRAINZ_ALBUMARTISTID"
	MusicBrainzArtistIDTag      Tag = "MUSICBRAINZ_ARTISTID"
	MusicBrainzTrackIDTag       Tag = "MUSICBRAINZ_TRACKID"
)

type Tag string
