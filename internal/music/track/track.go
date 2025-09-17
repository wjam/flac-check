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
	"regexp"
	"slices"
	"strconv"
	"strings"

	errors2 "github.com/wjam/flac-check/internal/errors"
	"github.com/wjam/flac-check/internal/log"
	"github.com/wjam/flac-check/internal/music/vorbis"
	"github.com/wjam/flac-check/internal/util"

	"github.com/go-flac/flacpicture/v2"
	"github.com/go-flac/flacvorbis/v2"
	"github.com/go-flac/go-flac/v2"
	"golang.org/x/text/encoding/charmap"
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
	if comment != nil {
		for _, c := range comment.Comments {
			equals := strings.IndexRune(c, '=')
			key := c[:equals]
			value := c[equals+1:]
			tags[key] = append(tags[key], value)
		}
	}

	return &Track{
		fileName:      path,
		flac:          f,
		comment:       comment,
		commentOffset: ci,
		picture:       pic,
		pictureOffset: pi,
		tags:          tags,
		newTags:       map[vorbis.Tag][]string{},
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
	newTags       map[vorbis.Tag][]string
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
	if _, ok := t.TagOk(vorbis.LyricsTag); ok {
		return true
	}
	if _, ok := t.TagOk(vorbis.UnsyncedLyricsTag); ok {
		return true
	}

	return false
}

func (t *Track) HasGenre() bool {
	v, ok := t.TagOk(vorbis.GenreTag)

	return ok && len(v) > 0
}

func (t *Track) SetUnsyncedLyrics(ctx context.Context, lyrics string, isInternational bool) {
	if _, ok := t.TagOk(vorbis.LyricsTag); ok {
		panic("check if track already has lyrics")
	}
	lyrics = tidyUpLyrics(ctx, lyrics, isInternational)
	if lyrics == "" {
		return
	}

	t.newTags[vorbis.UnsyncedLyricsTag] = []string{lyrics}
}

func (t *Track) SetSyncedLyrics(ctx context.Context, lyrics string, isInternational bool) {
	if _, ok := t.TagOk(vorbis.UnsyncedLyricsTag); ok {
		panic("check if track already has lyrics")
	}
	lyrics = tidyUpLyrics(ctx, lyrics, isInternational)
	if lyrics == "" {
		return
	}

	t.newTags[vorbis.LyricsTag] = []string{lyrics}
}

func (t *Track) SetMusicBrainzAlbumID(id string) {
	t.newTags[vorbis.MusicBrainzAlbumIDTag] = []string{id}
}

func (t *Track) SetGenres(genres []string) {
	t.newTags[vorbis.GenreTag] = genres
}

func (t *Track) CorrectTags() {
	//nolint:exhaustive // shorter code rather than covering all scenarios
	for tag, reg := range map[vorbis.Tag]*regexp.Regexp{
		vorbis.MusicBrainzAlbumIDTag:       regexp.MustCompile("^http://musicbrainz.org/release/(.*).html$"),
		vorbis.MusicBrainzAlbumArtistIDTag: regexp.MustCompile("^http://musicbrainz.org/artist/(.*)$"),
		vorbis.MusicBrainzArtistIDTag:      regexp.MustCompile("^http://musicbrainz.org/artist/(.*)$"),
		vorbis.MusicBrainzTrackIDTag:       regexp.MustCompile("^http://musicbrainz.org/track/(.*)$"),
	} {
		for _, value := range t.Tag(tag) {
			if matches := reg.FindStringSubmatch(value); matches != nil {
				t.newTags[tag] = []string{matches[1]}
			}
		}
	}

	//nolint:exhaustive // shorter code rather than covering all scenarios
	for tag, bad := range map[vorbis.Tag][]string{
		vorbis.GenreTag: {"Unknown"},
	} {
		tagValues := t.Tag(tag)
		changed := false
		for _, value := range bad {
			if i := slices.Index(tagValues, value); i != -1 {
				changed = true
				tagValues = append(tagValues[:i], tagValues[i+1:]...)
			}
		}
		if changed {
			t.newTags[tag] = tagValues
		}
	}
}

func (t *Track) ValidateTags() error {
	errs := t.validateExpectedTags()
	errs = append(errs, t.validateTagValues()...)
	errs = append(errs, t.validatePicture()...)

	return errors.Join(errs...)
}

func (t *Track) validateExpectedTags() []error {
	var errs []error
	for _, tag := range []vorbis.Tag{
		vorbis.ArtistTag,
		vorbis.TrackNumberTag,
		vorbis.TrackTotalTag,
		vorbis.AlbumTag,
		vorbis.TitleTag,
		vorbis.ArtistSortTag,
		vorbis.MusicBrainzAlbumIDTag,
		vorbis.DiscNumberTag,
	} {
		if values := t.Tag(tag); len(values) != 1 {
			if tag == vorbis.MusicBrainzAlbumIDTag &&
				((slices.Contains(t.Tag(vorbis.ArtistTag), "King Size Slim") &&
					slices.Contains(t.Tag(vorbis.AlbumTag), "Live at The Man of Kent Alehouse")) ||
					(slices.Contains(t.Tag(vorbis.ArtistTag), "House of the Rising Sun") && slices.Contains(t.Tag(vorbis.AlbumTag), "Tar Babies"))) {
				// No musicbrainz entries
				continue
			}
			errs = append(errs, errors2.NotSingleTagValueError{
				Tag:    tag,
				Values: values,
			})
		}
	}

	errs = append(errs, t.validateTagIsInt(vorbis.DiscNumberTag)...)
	errs = append(errs, t.validateTagIsInt(vorbis.TrackNumberTag)...)

	return errs
}

func (t *Track) validateTagIsInt(tag vorbis.Tag) []error {
	var errs []error
	if vees, ok := t.TagOk(tag); ok {
		var invalid []string
		for _, v := range vees {
			if _, err := strconv.Atoi(v); err != nil {
				invalid = append(invalid, v)
			}
		}

		if len(invalid) > 0 {
			errs = append(errs, errors2.InvalidIntTagError{
				Tag:    tag,
				Values: invalid,
			})
		}
	}
	return errs
}

func (t *Track) validateTagValues() []error {
	var errs []error
	//nolint:exhaustive // shorter code rather than covering all scenarios
	for tag, reg := range map[vorbis.Tag]*regexp.Regexp{
		vorbis.MusicBrainzAlbumIDTag:       regexp.MustCompile("^[A-Za-z0-9-]+$"),
		vorbis.MusicBrainzAlbumArtistIDTag: regexp.MustCompile("^[A-Za-z0-9-]+$"),
		vorbis.MusicBrainzArtistIDTag:      regexp.MustCompile("^[A-Za-z0-9-]+$"),
		vorbis.MusicBrainzTrackIDTag:       regexp.MustCompile("^[A-Za-z0-9-]+$"),
	} {
		for _, value := range t.Tag(tag) {
			if !reg.MatchString(value) {
				errs = append(errs, errors2.InvalidTagValueError{
					Tag:     tag,
					Pattern: reg.String(),
					Value:   value,
				})
			}
		}
	}
	return errs
}

func (t *Track) validatePicture() []error {
	var errs []error

	if t.HasPicture() {
		mime := http.DetectContentType(t.picture.ImageData)
		if mime != "image/jpeg" && mime != "image/png" {
			errs = append(errs, fmt.Errorf("invalid picture type: %s", mime))
		} else if t.picture.MIME != mime {
			errs = append(errs, fmt.Errorf("incorrect picture type %s - should be %s", mime, t.picture.MIME))
		}
	}

	return errs
}

func (t *Track) Tag(key vorbis.Tag) []string {
	if v, ok := t.newTags[key]; ok {
		return v
	}
	return t.tags[string(key)]
}

func (t *Track) TagOk(key vorbis.Tag) ([]string, bool) {
	if v, ok := t.newTags[key]; ok {
		return v, true
	}
	v, ok := t.tags[string(key)]
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
		if _, ok := t.tags[string(k)]; ok {
			removeComment(comment, string(k))
		}
		for _, v := range vs {
			if err := comment.Add(string(k), v); err != nil {
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
			value := "__TAG_REMOVED__"
			if len(v) > 0 {
				value = strings.Join(v, ",")
			}
			tagAttrs = append(tagAttrs, slog.String(string(k), value))
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

func removeComment(b *flacvorbis.MetaDataBlockVorbisComment, name string) {
	for i := 0; i < len(b.Comments); i++ {
		if strings.HasPrefix(b.Comments[i], fmt.Sprintf("%s=", name)) {
			b.Comments = append(b.Comments[:i], b.Comments[i+1:]...)
			i--
		}
	}
}

func notEnglishOrEmojiCharacters(r rune) bool {
	if _, ok := charmap.ISO8859_1.EncodeRune(r); ok {
		return false
	}

	// non-marker characters that commonly appear in lyrics that definitely aren't international characters
	emojis := map[rune]struct{}{
		'♪': {},
		'♫': {},
		'♬': {},
		'—': {},
		'–': {},
		'’': {},
	}
	if _, ok := emojis[r]; ok {
		return false
	}
	return true
}

type marshalable interface {
	Marshal() flac.MetaDataBlock
}
