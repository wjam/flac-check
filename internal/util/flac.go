package util

import (
	"github.com/go-flac/flacvorbis/v2"
	"github.com/go-flac/go-flac/v2"
)

func ExtractCommentFromFlacFile(f *flac.File) (*flacvorbis.MetaDataBlockVorbisComment, *int, error) {
	for idx, meta := range f.Meta {
		if meta.Type == flac.VorbisComment {
			cmt, err := flacvorbis.ParseFromMetaDataBlock(*meta)
			if err != nil {
				return nil, nil, err
			}

			return cmt, &idx, nil
		}
	}
	return nil, nil, nil
}
