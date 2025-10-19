package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"myid3/util"

	"github.com/rkoesters/xdg/trash"
)

const TARGET_FORMAT = "mp3"

func printUsage() {
	fmt.Fprint(os.Stderr, "USAGE: myid3 <path>\n")
	fmt.Fprint(os.Stderr, "EXAMPLES:\n")
	fmt.Fprint(os.Stderr, "    myid3 'path/to/Track File.mp3'\n")
	fmt.Fprint(os.Stderr, "    myid3 'path/to/Album Dir/'\n")
}

type song struct {
	number int
	title  string

	path     string
	basename string
}

type metadata struct {
	artist string
	album  string
	genre  string
	year   int
	disk   int

	coverPath string

	songs []song

	parentdir string
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		util.Fatal("Expected path to a song file or album directory")
	}
	path := os.Args[1]

	info, err := os.Stat(path)
	if err != nil {
		util.Fatal("%s", err)
	}

	meta := metadata{}

	if info.IsDir() {
		// Album dir
		meta.album = util.FixTitle(info.Name())
		meta.parentdir = path

		_, parentinfo, err := util.ParentDir(path)
		if err == nil {
			meta.artist = util.FixTitle(parentinfo.Name())
		}

		collectSongs(&meta, path, info)
	} else {
		// Track file
		parentdir, _, err := util.ParentDir(path)
		if err != nil {
			util.Fatal("Unable to get parent dir: %s", err)
		}
		meta.parentdir = parentdir

		number, title := util.SongNumberAndTitle(info.Name())
		meta.songs = append(meta.songs, song{
			number: number,
			title:  title,

			path:     path,
			basename: info.Name(),
		})
	}

	for {
		prompt(&meta)
		if confirm(&meta) {
			break
		}
	}

	convert(&meta)

	fmt.Println("DONE!")
}

func prompt(meta *metadata) {
	fmt.Println("==============================")
	fmt.Println("<Enter> for default value")
	fmt.Println("    '-' for empty value")
	fmt.Println("==============================")
	fmt.Println()

	if len(meta.songs) == 1 {
		song := &meta.songs[0]
		song.number = util.AskInt("Number", song.number)
		song.title = util.AskString("Title", song.title)
	}

	meta.artist = util.AskString("Artist", meta.artist)
	meta.album = util.AskString("Album", meta.album)
	meta.genre = util.AskString("Genre", meta.genre)
	meta.year = util.AskInt("Year", meta.year)
	meta.disk = util.AskInt("Disk", meta.disk)

	for {
		path := util.AskString("Cover image path", meta.coverPath)
		if len(path) == 0 {
			meta.coverPath = path
			break
		}

		_, err := os.Stat(path)
		if err != nil {
			util.Error("No such file, try again: %s", err)
		} else {
			meta.coverPath = path
			break
		}
	}
}

func printFieldStr(name string, str string) {
	if len(str) == 0 {
		str = "-"
	}
	fmt.Printf("%s: %s\n", name, str)
}
func printFieldInt(name string, num int) {
	if num <= 0 {
		fmt.Printf("%s: -\n", name)
	} else {
		fmt.Printf("%s: %d\n", name, num)
	}
}
func confirm(meta *metadata) bool {
	fmt.Println()
	fmt.Println("==============================")
	fmt.Println("Confirm")
	fmt.Println("==============================")
	fmt.Println()

	printFieldStr("Artist", meta.artist)
	printFieldStr("Album", meta.album)
	printFieldStr("Genre", meta.genre)
	printFieldInt("Year", meta.year)
	printFieldInt("Disk", meta.disk)
	printFieldStr("Cover image path", meta.coverPath)

	fmt.Println()

	w := tabwriter.NewWriter(os.Stdout, 1, 1, 1, ' ', 0)
	fmt.Fprintf(w, "Number\tTitle\n")
	for _, s := range meta.songs {
		if s.number > 0 {
			fmt.Fprintf(w, "%d\t", s.number)
		}
		if len(s.title) > 0 {
			fmt.Fprintf(w, "%s\t", s.title)
		}
		fmt.Fprintf(w, "\n")
	}

	w.Flush()

	fmt.Println()

	return util.AskBool("Ok?")
}

func pushTagStr(args *[]string, tag string, value string) {
	if len(value) > 0 {
		*args = append(*args, "-metadata", fmt.Sprintf("%s=%s", tag, value))
	}
}
func pushTagInt(args *[]string, tag string, value int) {
	if value > 0 {
		*args = append(*args, "-metadata", fmt.Sprintf("%s=%d", tag, value))
	}
}

// Use `-codec copy` if file's format is the same as TARGET_FORMAT
func pushCodec(args *[]string, songPath string) {
	format := util.RunCmdWithOutput(
		"ffprobe",
		"-loglevel", "quiet",
		"-print_format", "flat",
		"-show_format",
		songPath,
	)

	if strings.Contains(format, TARGET_FORMAT) {
		// Format is the same
		*args = append(*args, "-codec", "copy")
	} else {
		// Some different format
		*args = append(
			*args,
			"-c:a", "libmp3lame",
			"-b:a", "320k",
		)
	}
}
func pushCover(args *[]string, coverPath string) {
	if len(coverPath) > 0 {
		*args = append(
			*args,
			"-i", coverPath,
			"-map", "0:0",
			"-map", "1:0",
		)
	} else {
		// Remove current cover if any
		*args = append(
			*args,
			"-map", "0:a",
			"-map_metadata", "-1",
		)
	}
}
func convert(meta *metadata) {
	// Create a temporary dir where all the converted files will be stored
	targetDir := filepath.Join(meta.parentdir, "converted")
	os.MkdirAll(targetDir, os.ModePerm)

	for idx, s := range meta.songs {
		args := make([]string, 0, 16)

		// Inputs

		args = append(
			args,
			"-loglevel", "repeat+warning",
			"-i", s.path,
		)

		pushCover(&args, meta.coverPath)

		// Output options

		args = append(args, "-id3v2_version", "3")

		pushTagInt(&args, "track", s.number)
		pushTagStr(&args, "title", s.title)
		pushTagStr(&args, "artist", meta.artist)
		pushTagStr(&args, "album", meta.album)
		pushTagStr(&args, "genre", meta.genre)
		pushTagInt(&args, "year", meta.year)
		pushTagInt(&args, "disk", meta.disk)

		pushCodec(&args, s.path)

		targetBasename := util.SetExt(s.basename, TARGET_FORMAT)
		targetFilepath := filepath.Join(targetDir, targetBasename)
		args = append(args, targetFilepath) // push target file

		fmt.Println("Converting song %d/%d...", idx+1, len(meta.songs))

		if !util.RunCmd("ffmpeg", args...) {
			util.Fatal("Song conversion failed, stop")
		}

		// Trash the original song file
		trash.Trash(s.path)
		// Move the converted song file back to the albums dir
		os.Rename(targetFilepath, s.path)
		// Trash the temp directory (just in case)
		trash.Trash(targetDir)
	}
}

func collectSongs(meta *metadata, path string, info os.FileInfo) {
	entries, err := os.ReadDir(path)
	if err != nil {
		util.Fatal("Unable to read albums dir: %s", err)
	}

	for _, e := range entries {
		thisPath := filepath.Join(path, e.Name())

		if e.IsDir() {
			util.Warn("'%s' is a directory, skip", thisPath)
			continue
		}

		if e.Name() == "cover.png" || e.Name() == "cover.jpg" || e.Name() == "cover.webp" {
			meta.coverPath = thisPath
			continue
		}

		number, title := util.SongNumberAndTitle(e.Name())

		meta.songs = append(meta.songs, song{
			number: number,
			title:  title,

			path:     thisPath,
			basename: e.Name(),
		})
	}
}
