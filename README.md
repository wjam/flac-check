# flac-check

Tool for maintaining my music FLAC files:
* make sure all album tracks are consistent
* make sure all tracks have relevant data
* populate missing data if appropriate
* Rate limited access to external APIs to be a good citizen - 1 request per second per hostname

## TODO
* Add alternative lyric sources, such as https://genius.com/developers
* Add a test that pictures with the wrong mime type fail. There is the issue of building the FLAC in the first place, as the library enforces this.

## NOTES

* Lyric file format - https://en.wikipedia.org/wiki/LRC_(file_format)
