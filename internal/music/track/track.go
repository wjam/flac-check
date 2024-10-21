package track

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/go-flac/flacpicture/v2"
	"github.com/go-flac/flacvorbis/v2"
	"github.com/go-flac/go-flac/v2"
	errors2 "github.com/wjam/flac-check/internal/errors"
	"github.com/wjam/flac-check/internal/log"
	"github.com/wjam/flac-check/internal/util"
)

func NewTrack(path string) (*Track, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	f, err := flac.ParseBytes(bytes.NewReader(content))
	if err != nil {
		return nil, err
	}

	comment, ci, err := util.ExtractCommentFromFlacFile(f)
	if err != nil {
		return nil, err
	}

	pic, pi, err := extractPicture(f)
	if err != nil {
		return nil, err
	}

	tags := map[string][]string{}
	for _, c := range comment.Comments {
		parts := strings.SplitN(c, "=", 2)
		tags[parts[0]] = append(tags[parts[0]], parts[1])
	}

	return &Track{
		fileName:      path,
		flac:          f,
		comment:       comment,
		commentOffset: ci,
		picture:       pic,
		pictureOffset: pi,
		tags:          tags,
		newTags:       map[string][]string{},
	}, nil
}

type Track struct {
	fileName      string
	flac          *flac.File
	comment       *flacvorbis.MetaDataBlockVorbisComment
	commentOffset *int
	picture       *flacpicture.MetadataBlockPicture
	pictureOffset *int
	tags          map[string][]string
	newTags       map[string][]string
	newPicture    *flacpicture.MetadataBlockPicture
}

func (t *Track) SetPicture(pic []byte, url string) error {
	mime := http.DetectContentType(pic)
	if mime != "image/jpeg" && mime != "image/png" {
		return fmt.Errorf("invalid picture type: %s", mime)
	}
	img, err := flacpicture.NewFromImageData(flacpicture.PictureTypeFrontCover, url, pic, mime)
	if err != nil {
		return err
	}

	t.newPicture = img

	return nil
}

func (t *Track) HasPicture() bool {
	return t.picture != nil
}

func (t *Track) HasLyrics() bool {
	if _, ok := t.TagOk("LYRICS"); ok {
		return true
	}
	if _, ok := t.TagOk("UNSYNCEDLYRICS"); ok {
		return true
	}

	return false
}

func (t *Track) SetUnsyncedLyrics(ctx context.Context, lyrics string, isInternational bool) {
	if _, ok := t.TagOk("LYRICS"); ok {
		panic("check if track already has lyrics")
	}
	lyrics = tidyUpLyrics(ctx, lyrics, isInternational)
	if lyrics == "" {
		return
	}

	t.newTags["UNSYNCEDLYRICS"] = []string{lyrics}
}

func (t *Track) SetSyncedLyrics(ctx context.Context, lyrics string, isInternational bool) {
	if _, ok := t.TagOk("UNSYNCEDLYRICS"); ok {
		panic("check if track already has lyrics")
	}
	lyrics = tidyUpLyrics(ctx, lyrics, isInternational)
	if lyrics == "" {
		return
	}

	t.newTags["LYRICS"] = []string{lyrics}
}

func (t *Track) ValidateTags() error {
	var errs []error
	for _, tag := range []string{"ARTIST", "TRACKNUMBER", "TRACKTOTAL", "ALBUM", "TITLE", "ARTISTSORT"} {
		if values := t.Tag(tag); len(values) != 1 {
			errs = append(errs, errors2.ErrNotSingleTagValue{
				Tag:    tag,
				Values: values,
			})
		}
	}

	if t.HasPicture() {
		mime := http.DetectContentType(t.picture.ImageData)
		if mime != "image/jpeg" && mime != "image/png" {
			errs = append(errs, fmt.Errorf("invalid picture type: %s", mime))
		} else if t.picture.MIME != mime {
			errs = append(errs, fmt.Errorf("incorrect picture type %s - should be %s", mime, t.picture.MIME))
		}
	}

	return errors.Join(errs...)
}

func (t *Track) Tag(key string) []string {
	if v, ok := t.newTags[key]; ok {
		return v
	}
	return t.tags[key]
}

func (t *Track) TagOk(key string) ([]string, bool) {
	if v, ok := t.newTags[key]; ok {
		return v, true
	}
	v, ok := t.tags[key]
	return v, ok
}

func (t *Track) String() string {
	return filepath.Base(t.fileName)
}

func (t *Track) Save(ctx context.Context, write bool) error {
	if len(t.newTags) == 0 && t.newPicture == nil {
		return nil
	}

	if write {
		return t.saveChanges(ctx)
	}

	t.logChanges(ctx)
	return nil
}

func (t *Track) saveChanges(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	log.Logger(ctx).WarnContext(ctx, "Saving changes to track", t.changesToSlogAttrs()...)

	if err := t.updateFlacWithNewTags(); err != nil {
		return err
	}

	if err := t.updateFlacWithNewPicture(); err != nil {
		return err
	}
	return t.flac.Save(t.fileName)
}

func (t *Track) updateFlacWithNewTags() error {
	if len(t.newTags) == 0 {
		return nil
	}

	comment := t.comment
	if comment == nil {
		comment = &flacvorbis.MetaDataBlockVorbisComment{}
	}

	for k, vs := range t.newTags {
		for _, v := range vs {
			if err := comment.Add(k, v); err != nil {
				return err
			}
		}
	}

	t.saveBlockToFlac(comment, t.commentOffset)

	return nil
}

func (t *Track) updateFlacWithNewPicture() error {
	if t.newPicture == nil {
		return nil
	}

	t.newPicture.Description = ""

	t.saveBlockToFlac(t.newPicture, t.pictureOffset)

	return nil
}

func (t *Track) saveBlockToFlac(block marshalable, offset *int) {
	m := block.Marshal()
	if offset != nil {
		t.flac.Meta[*offset] = &m
		return
	}

	t.flac.Meta = append(t.flac.Meta, &m)
}

func (t *Track) logChanges(ctx context.Context) {
	log.Logger(ctx).WarnContext(ctx, "Updated track", t.changesToSlogAttrs()...)
}

func (t *Track) changesToSlogAttrs() []any {
	var attrs []any
	if len(t.newTags) > 0 {
		var tagAttrs []any
		for k, v := range t.newTags {
			tagAttrs = append(tagAttrs, slog.String(k, strings.Join(v, ",")))
		}
		attrs = append(attrs, slog.Group("tags", tagAttrs...))
	}

	if t.newPicture != nil {
		attrs = append(attrs, slog.Group("picture",
			slog.String("url", t.newPicture.Description),
			slog.String("mime", t.newPicture.MIME),
			slog.Uint64("height", uint64(t.newPicture.Height)),
			slog.Uint64("width", uint64(t.newPicture.Width)),
		))
	}

	return attrs
}

func extractPicture(f *flac.File) (*flacpicture.MetadataBlockPicture, *int, error) {
	for idx, meta := range f.Meta {
		if meta.Type == flac.Picture {
			pic, err := flacpicture.ParseFromMetaDataBlock(*meta)
			if err != nil {
				return nil, nil, err
			}

			if pic.PictureType != flacpicture.PictureTypeFrontCover {
				continue
			}

			return pic, &idx, nil
		}
	}
	return nil, nil, nil
}

func tidyUpLyrics(ctx context.Context, text string, isInternational bool) string {
	// Replace probable marker characters
	text = strings.NewReplacer(
		string('е'), "e",
		string('ﾠ'), " ",
	).Replace(text)

	if !strings.ContainsFunc(text, notEnglishOrEmojiCharacters) || isInternational {
		return text
	}

	unknown := map[string]struct{}{}
	for _, e := range text {
		if notEnglishOrEmojiCharacters(e) {
			unknown[string(e)] = struct{}{}
		}
	}

	nonEnglishChars := util.Keys(unknown)
	slices.Sort(nonEnglishChars)
	log.Logger(ctx).InfoContext(ctx,
		"Skipped lyrics as it wasn't english",
		slog.String("unknown", strings.Join(nonEnglishChars, "")),
		slog.String("lyrics", text),
	)
	return ""
}

// non-marker characters that commonly appear in lyrics that definitely aren't international characters
var emojis = map[rune]struct{}{
	'♪': {},
	'♫': {},
	'♬': {},
	'—': {},
	'–': {},
	'’': {},
}

func notEnglishOrEmojiCharacters(r rune) bool {
	if r <= 255 {
		return false
	}
	if _, ok := emojis[r]; ok {
		return false
	}
	return true
}

type marshalable interface {
	Marshal() flac.MetaDataBlock
}
