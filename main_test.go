package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/go-flac/flacpicture/v2"
	"github.com/go-flac/flacvorbis/v2"
	"github.com/go-flac/go-flac/v2"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wjam/flac-check/internal/errors"
	"github.com/wjam/flac-check/internal/util"
	"golang.org/x/tools/txtar"
)

func TestRoot(t *testing.T) {
	tests := []struct {
		name         string
		expectedErrs []error
	}{
		{name: "no-write-flag"},
		{name: "write-flag"},
		{
			name: "missing-artist-tag",
			expectedErrs: []error{
				errors.ErrNotSingleTagValue{
					Tag:    "ARTIST",
					Values: nil,
				},
				errors.ErrNotSingleAlbumArtist{
					Artists:      nil,
					AlbumArtists: nil,
				},
			},
		},
		{
			name: "missing-track-number-tag",
			expectedErrs: []error{
				errors.ErrNotSingleTagValue{
					Tag:    "TRACKNUMBER",
					Values: nil,
				},
			},
		},
		{
			name: "missing-track-total-tag",
			expectedErrs: []error{
				errors.ErrNotSingleTagValue{
					Tag:    "TRACKTOTAL",
					Values: nil,
				},
			},
		},
		{
			name: "missing-album-tag",
			expectedErrs: []error{
				errors.ErrNotSingleTagValue{
					Tag:    "ALBUM",
					Values: nil,
				},
			},
		},
		{
			name: "missing-title-tag",
			expectedErrs: []error{
				errors.ErrNotSingleTagValue{
					Tag:    "TITLE",
					Values: nil,
				},
			},
		},
		{
			name: "missing-artist-sort-tag",
			expectedErrs: []error{
				errors.ErrNotSingleTagValue{
					Tag:    "ARTISTSORT",
					Values: nil,
				},
			},
		},
		{
			name: "mismatched-album-tag",
			expectedErrs: []error{
				errors.ErrNotSingleTagValue{
					Tag:    "ALBUM",
					Values: []string{"album1", "album2"},
				},
			},
		},
		{
			name: "mismatched-date-tag",
			expectedErrs: []error{
				errors.ErrNotSingleTagValue{
					Tag:    "DATE",
					Values: []string{"2024", "2024-01-01"},
				},
			},
		},
		{
			name: "invalid-date-tag",
			expectedErrs: []error{
				errors.ErrInvalidValue{
					Tag:    "DATE",
					Values: []string{"0001-01-01"},
				},
			},
		},
		{name: "consistent-albumartist-tag"},
		{
			name: "inconsistent-albumartist-tag",
			expectedErrs: []error{
				errors.ErrNotSingleAlbumArtist{
					Artists:      []string{"artist1", "artist1 and someone else"},
					AlbumArtists: []string{"artist1", "artist2"},
				},
			},
		},
		{
			name: "inconsistent-artist-tag",
			expectedErrs: []error{
				errors.ErrNotSingleAlbumArtist{
					Artists: []string{"artist1", "artist1 and someone else"},
				},
			},
		},
		{
			name: "missing-picture-and-musicbrainz-tag",
			expectedErrs: []error{
				errors.ErrNotSingleTagValue{
					Tag: "MUSICBRAINZ_ALBUMID",
				},
			},
		},
		{name: "missing-date-tag-skipped"},
		{name: "international-lyrics-for-non-international-artist"},
		{name: "international-lyrics-for-international-artist"},
		{name: "funky-lyric-chars-dropped"},
		{name: "default-log-level"},
		{name: "musicbrainz-release-id-from-disc-id"},
		{name: "picture-from-wikipedia"},
		{
			name: "inconsistent-genre-tag",
			expectedErrs: []error{
				errors.ErrInvalidGenreTag{
					Values: []string{"granite", "rock"},
				},
			},
		},
		{
			name: "inconsistent-genre-tag-with-missing",
			expectedErrs: []error{
				errors.ErrInvalidGenreTag{
					Values: []string{"metal", "rock"},
				},
			},
		},
		{name: "update-genre-tag"},
		{name: "fix-bad-musicbrainz-albumartistid-tag"},
		{name: "fix-bad-musicbrainz-albumid-tag"},
		{name: "fix-bad-musicbrainz-artistid-tag"},
		{name: "fix-bad-musicbrainz-trackid-tag"},
		{name: "missing-musicbrainz-albumid-skipped"},
		{name: "replace-unknown-genre-tag"},
		{name: "remove-unknown-genre-tag"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			err := runMusicTest(t, dir, root(), filepath.Join("testdata", "root", test.name, "cmd.txtar"))

			if len(test.expectedErrs) == 0 {
				assert.NoError(t, err)
			} else {
				for _, expectedErr := range test.expectedErrs {
					assert.ErrorIs(t, err, expectedErr)
				}
			}

			assertMusicContent(t, dir, filepath.Join("testdata", "root", test.name, "expected.txtar"))
		})
	}
}

func assertMusicContent(t *testing.T, dir, input string) {
	test, err := txtar.ParseFile(input)
	require.NoError(t, err)

	for _, file := range test.Files {
		actual := readFlacFile(t, filepath.Join(dir, file.Name))
		var expected flacFile
		require.NoError(t, json.Unmarshal(file.Data, &expected))

		assert.Equalf(t, expected, actual, "File %s was different", file.Name)
	}
}

func runMusicTest(t *testing.T, dir string, cmd *cobra.Command, input string) error {
	test, err := txtar.ParseFile(input)
	require.NoError(t, err)

	serverRequests := map[string]map[request]string{}

	for _, file := range test.Files {
		match := requestPattern.FindStringSubmatch(file.Name)
		if match == nil {
			continue
		}
		if _, ok := serverRequests[match[2]]; !ok {
			serverRequests[match[2]] = map[request]string{}
		}

		serverRequests[match[2]][request{
			method: match[1],
			path:   match[3],
		}] = string(file.Data)
	}

	var replacements []string
	for name, requests := range serverRequests {
		s := httptest.NewServer(requestHandler{requests})
		t.Cleanup(s.Close)
		replacements = append(replacements, name, s.URL)
	}
	replacement := strings.NewReplacer(replacements...)

	for _, requests := range serverRequests {
		for k, data := range requests {
			requests[k] = replacement.Replace(data)
		}
	}

	var args []string
	comment := strings.TrimSpace(string(test.Comment))
	comment = replacement.Replace(comment)
	for _, l := range strings.Split(comment, "\n") {
		if !strings.HasPrefix(l, "#") {
			args = append(args, strings.Split(l, " ")...)
		}
	}

	var expectedStdout, expectedStderr string
	for _, file := range test.Files {
		data := []byte(replacement.Replace(string(file.Data)))
		if file.Name == "stdout" {
			expectedStdout = string(data)
			continue
		}
		if file.Name == "stderr" {
			expectedStderr = string(data)
			continue
		}

		if match := requestPattern.FindStringSubmatch(file.Name); match != nil {
			continue
		}

		if strings.HasSuffix(file.Name, ".flac") {
			makeFlacFile(t, filepath.Join(dir, file.Name), data)
			continue
		}

		require.NoError(t, os.WriteFile(filepath.Join(dir, file.Name), data, 0644))
	}

	cwd, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, os.Chdir(cwd))
	}()

	assert.NoError(t, os.Chdir(dir))

	var stdout, stderr bytes.Buffer
	cmd.SetArgs(args)
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err = cmd.ExecuteContext(contextFromTesting(t))

	assert.Equal(t, expectedStdout, stdout.String())
	assert.Equal(t, expectedStderr, stderr.String())

	return err
}

type request struct {
	method string
	path   string
}

var requestPattern = regexp.MustCompile("^(GET|POST) (__[^/]*__)(/.*)$")

var _ http.Handler = requestHandler{}

type requestHandler struct {
	requests map[request]string
}

func (h requestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	v, ok := h.requests[request{r.Method, r.RequestURI}]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	response, err := http.ReadResponse(bufio.NewReader(strings.NewReader(v)), r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer func() {
		if err := response.Body.Close(); err != nil {
			panic(err)
		}
	}()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if strings.HasPrefix(response.Header.Get("Content-Type"), "image/") {
		var err error
		body, err = base64.StdEncoding.DecodeString(string(body))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	for k, v := range response.Header {
		w.Header()[k] = v
	}
	w.WriteHeader(response.StatusCode)
	if _, err := w.Write(body); err != nil {
		panic(err)
	}
}

func contextFromTesting(t *testing.T) context.Context {
	ctx := t.Context()

	d, ok := t.Deadline()
	if !ok {
		return ctx
	}

	gracePeriod := 10 * time.Second
	ctx, cancel := context.WithDeadline(ctx, d.Truncate(gracePeriod))
	t.Cleanup(cancel)

	return ctx
}

type flacFile struct {
	Tags     map[string][]string `json:"tags"`
	Pictures []flacPicture       `json:"pictures"`
}

type flacPicture struct {
	Type string `json:"type"`
	Mime string `json:"mime"`
	Img  string `json:"img"`
}

func readFlacFile(t *testing.T, path string) flacFile {
	f, err := flac.ParseFile(path)
	require.NoError(t, err)

	comment, _, err := util.ExtractCommentFromFlacFile(f)
	require.NoError(t, err)

	tags := map[string][]string{}
	for _, c := range comment.Comments {
		parts := strings.SplitN(c, "=", 2)
		tags[parts[0]] = append(tags[parts[0]], parts[1])
	}

	pics := extractPictures(t, f)

	return flacFile{
		Tags:     tags,
		Pictures: pics,
	}
}

func extractPictures(t *testing.T, f *flac.File) []flacPicture {
	var pics []flacPicture
	for _, meta := range f.Meta {
		if meta.Type == flac.Picture {
			pic, err := flacpicture.ParseFromMetaDataBlock(*meta)
			require.NoError(t, err)

			picType, ok := pictureTypeToString[pic.PictureType]
			require.Truef(t, ok, "Unexpected type %v", pic.PictureType)

			pics = append(pics, flacPicture{
				Type: picType,
				Mime: pic.MIME,
				Img:  base64.StdEncoding.EncodeToString(pic.ImageData),
			})
		}
	}
	return pics
}

var stringToPictureType = map[string]flacpicture.PictureType{
	"cover": flacpicture.PictureTypeFrontCover,
	"back":  flacpicture.PictureTypeBackCover,
}
var pictureTypeToString = map[flacpicture.PictureType]string{
	flacpicture.PictureTypeFrontCover: "cover",
	flacpicture.PictureTypeBackCover:  "back",
}

func makeFlacFile(t *testing.T, file string, content []byte) {
	var config flacFile
	require.NoError(t, json.Unmarshal(content, &config))

	blocks := []*flac.MetaDataBlock{
		buildFlacTags(t, config.Tags),
	}
	for _, p := range config.Pictures {
		picType, ok := stringToPictureType[p.Type]
		if !ok {
			t.Fatalf("unknown picture type: %s", p.Type)
		}

		blocks = append(blocks, buildFlacPicture(t, picType, p.Img, p.Mime))
	}

	saveFlacFile(t, file, blocks...)
}

func saveFlacFile(t *testing.T, path string, blocks ...*flac.MetaDataBlock) {
	dir := filepath.Dir(path)
	require.NoError(t, os.MkdirAll(dir, 0755))

	f := flac.File{
		Meta:   blocks,
		Frames: bytes.NewBuffer([]byte{0xFF, 0xF8}),
	}

	require.NoError(t, f.Save(path))
}

func buildFlacTags(t *testing.T, tags map[string][]string) *flac.MetaDataBlock {
	comment := flacvorbis.New()

	for k, vs := range tags {
		for _, v := range vs {
			require.NoError(t, comment.Add(k, v))
		}
	}

	block := comment.Marshal()
	return &block
}

func buildFlacPicture(t *testing.T, picType flacpicture.PictureType, img, mime string) *flac.MetaDataBlock {
	content, err := base64.StdEncoding.DecodeString(img)
	require.NoError(t, err)

	picture, err := flacpicture.NewFromImageData(picType, "", content, mime)
	require.NoError(t, err)

	block := picture.Marshal()

	return &block
}
