package main

import (
	"flag"
	"fmt"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/malashin/ffinfo"
)

// Flags.
var flagSegmentDuration float64
var flagSkipVisualAdSegment bool
var flagSkipAudioAdSegment bool
var flagSkipSilenceSegment bool
var flagSkipDesyncSegment bool

// Media files.
var visualAdPath string = "./media/1xbet_logo_white_10sec.mov"
var visualAdDuration float64 = 10

var audioAdPath string = "./media/yandex_eda_10sec.mp4"
var audioAdDuration float64 = 10

func main() {
	// Flags.
	flag.Float64Var(&flagSegmentDuration, "d", 120, "Duration of the output segments in seconds, must be at least 11 seconds")
	flag.BoolVar(&flagSkipVisualAdSegment, "skip_visual_ad", false, "Do not create visual add segment")
	flag.BoolVar(&flagSkipAudioAdSegment, "skip_audio_ad", false, "Do not create audio add segment")
	flag.BoolVar(&flagSkipSilenceSegment, "skip_silence", false, "Do not create silence segment")
	flag.BoolVar(&flagSkipDesyncSegment, "skip_desync", false, "Do not create desync segment")
	flag.Usage = func() {
		fmt.Println("Usage: otkhoneypot [options] [file1 file2 ...]")
		flag.PrintDefaults()
	}
	flag.Parse()

	files := flag.Args()
	if len(files) < 1 {
		flag.Usage()
	}

	if flagSegmentDuration < 11 {
		fmt.Printf("Segment duration (%v) is too low, must be at least 11\n", flagSegmentDuration)
		os.Exit(1)
	}

	// Iterate over files.
	for _, filePath := range files {
		// Get ffprobe data about the file.
		f, err := parseFile(filePath)
		if err != nil {
			panic(err)
		}

		//Make honeypot segments.
		if !flagSkipVisualAdSegment {
			err = makeVisualAdSegment(f)
			if err != nil {
				panic(err)
			}
		}

		if !flagSkipAudioAdSegment {
			err = makeAudioAdSegment(f)
			if err != nil {
				panic(err)
			}
		}

		if !flagSkipSilenceSegment {
			err = makeSilenceSegment(f)
			if err != nil {
				panic(err)
			}
		}

		if !flagSkipDesyncSegment {
			err = makeDesyncSegment(f)
			if err != nil {
				panic(err)
			}
		}
	}
}

func parseFile(filePath string) (f *ffinfo.File, err error) {
	f, err = ffinfo.Probe(filePath)
	if err != nil {
		return nil, err
	}

	formatDuration, err := strconv.ParseFloat(f.Format.Duration, 64)
	if err != nil {
		return nil, err
	}

	if formatDuration < flagSegmentDuration {
		return nil, fmt.Errorf("File duration (%v) is smaller then duration flag (%v)", formatDuration, flagSegmentDuration)
	}

	return f, nil
}

func getSegmentStart(f *ffinfo.File) (float64, error) {
	formatDuration, err := strconv.ParseFloat(f.Format.Duration, 64)
	if err != nil {
		return 0, nil
	}

	max := int((formatDuration - math.Mod(formatDuration, flagSegmentDuration)) / flagSegmentDuration)

	min := 0
	if max > 1 {
		min = 1
	}
	if max > 2 {
		max--
	}

	rand.Seed(time.Now().UnixNano())
	random := float64(rand.Intn(max-min) + min)

	return flagSegmentDuration * random, nil
}

func makeVisualAdSegment(f *ffinfo.File) error {
	segmentStart, err := getSegmentStart(f)
	if err != nil {
		return err
	}

	min := 0
	max := int(flagSegmentDuration) - int(visualAdDuration)
	if max > 1 {
		min = 1
	}
	rand.Seed(time.Now().UnixNano())
	rand.Intn(max)
	defectOffset := fmt.Sprintf("%v", math.Max(0, float64(rand.Intn(max-min)+min)))

	fmt.Printf("> Making visual ad segment for \"%v\"\n", filepath.Base(f.Format.Filename))
	fmt.Printf("> segmentStart: %v; segmentDuration: %v; defectOffset: %v\n\n", segmentStart, flagSegmentDuration, defectOffset)

	command := []string{
		"-ss", fmt.Sprintf("%v", segmentStart),
		"-i", f.Format.Filename,
		"-i", visualAdPath,
		"-filter_complex", "[1:v]setpts=PTS+" + defectOffset + "/TB[o],[0:v][o]overlay=enable=gte(t\\," + defectOffset + ")::x=main_w-overlay_w-main_w/20:y=main_h-overlay_h:eof_action=pass,format=yuv420p[v]",
		"-map", "[v]",
		"-vcodec", "libx264",
		"-pix_fmt", "yuv420p",
		"-preset", "medium",
		"-crf", "18",
		"-acodec", "aac",
		"-map", "0:a",
		"-ab", "256k",
		"-ar", "48000",
		"-t", fmt.Sprintf("%v", flagSegmentDuration),
		"-loglevel", "error",
		"-stats",
		"-y",
		"-hide_banner",
		strings.TrimSuffix(f.Format.Filename, filepath.Ext(f.Format.Filename)) + "_visual_ad.mp4",
	}

	fmt.Printf("ffmpeg %v\n\n", strings.Join(command, " "))

	cmd := exec.Command("ffmpeg", command...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func makeAudioAdSegment(f *ffinfo.File) error {
	segmentStart, err := getSegmentStart(f)
	if err != nil {
		return err
	}

	min := 0
	max := int(flagSegmentDuration) - int(audioAdDuration)
	if max > 1 {
		min = 1
	}
	rand.Seed(time.Now().UnixNano())
	rand.Intn(max)
	defectOffset := fmt.Sprintf("%v", math.Max(0, float64(rand.Intn(max-min)+min)))

	fmt.Printf("> Making audio ad segment for \"%v\"\n", filepath.Base(f.Format.Filename))
	fmt.Printf("> segmentStart: %v; segmentDuration: %v; defectOffset: %v\n\n", segmentStart, flagSegmentDuration, defectOffset)

	var filterComplex string
	var audioOutput []string
	var i int
	for _, s := range f.Streams {
		if s.CodecType == "audio" {
			filterComplex += fmt.Sprintf("[0:%v]volume=0.25:enable='between(t,%v,%v)'[a%v],[1:a]volume=1.5[ad%v],[ad%v][a%v]amix=inputs=2:duration=longest:dropout_transition=2[a%v],", s.Index, defectOffset, audioAdDuration, s.Index, s.Index, s.Index, s.Index, s.Index)

			disposition := "none"
			if s.Disposition.Default == 1 {
				disposition = "default"
			}

			audioOutput = append(audioOutput, "-map", fmt.Sprintf("[a%v]", s.Index), "-acodec", "aac", "-ab", "256k", "-ar", "48000", fmt.Sprintf("-metadata:s:a:%v", i), fmt.Sprintf("language=%v", s.Tags.Language), fmt.Sprintf("-disposition:a:%v", i), disposition)

			if s.Tags.HandlerName != "SoundHandler" {
				audioOutput = append(audioOutput, fmt.Sprintf("-metadata:s:a:%v", i), fmt.Sprintf("handler=\"%v\"", s.Tags.HandlerName))
			}
			i++
		}
	}

	filterComplex = strings.TrimSuffix(filterComplex, ",")

	command := []string{
		"-ss", fmt.Sprintf("%v", segmentStart),
		"-i", f.Format.Filename,
		"-itsoffset", defectOffset,
		"-i", audioAdPath,
		"-filter_complex", filterComplex,
		"-map", "0:v",
		"-vcodec", "libx264",
		"-pix_fmt", "yuv420p",
		"-preset", "medium",
		"-crf", "18",
		"-async", "1",
	}

	command = append(command, audioOutput...)

	command = append(
		command, "-t", fmt.Sprintf("%v", flagSegmentDuration),
		"-loglevel", "error",
		"-stats",
		"-y",
		"-hide_banner",
		strings.TrimSuffix(f.Format.Filename, filepath.Ext(f.Format.Filename))+"_audio_ad.mp4",
	)

	fmt.Printf("ffmpeg %v\n\n", strings.Join(command, " "))

	cmd := exec.Command("ffmpeg", command...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func makeSilenceSegment(f *ffinfo.File) error {
	segmentStart, err := getSegmentStart(f)
	if err != nil {
		return err
	}

	min := 0
	max := int(flagSegmentDuration) - int(audioAdDuration)
	if max > 1 {
		min = 1
	}
	rand.Seed(time.Now().UnixNano())
	rand.Intn(max)
	defectOffset := fmt.Sprintf("%v", math.Max(0, float64(rand.Intn(max-min)+min)))

	fmt.Printf("> Making silence segment for \"%v\"\n", filepath.Base(f.Format.Filename))
	fmt.Printf("> segmentStart: %v; segmentDuration: %v; defectOffset: %v\n\n", segmentStart, flagSegmentDuration, defectOffset)

	command := []string{
		"-ss", fmt.Sprintf("%v", segmentStart),
		"-i", f.Format.Filename,
		"-map", "0:v",
		"-vcodec", "libx264",
		"-pix_fmt", "yuv420p",
		"-preset", "medium",
		"-crf", "18",
		"-acodec", "aac",
		"-map", "0:a",
		"-ab", "256k",
		"-ar", "48000",
		"-af", fmt.Sprintf("volume=0:enable='between(t,%v,%v)'", defectOffset, 10),
		"-t", fmt.Sprintf("%v", flagSegmentDuration),
		"-loglevel", "error",
		"-stats",
		"-y",
		"-hide_banner",
		strings.TrimSuffix(f.Format.Filename, filepath.Ext(f.Format.Filename)) + "_silence.mp4",
	}

	fmt.Printf("ffmpeg %v\n\n", strings.Join(command, " "))

	cmd := exec.Command("ffmpeg", command...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func makeDesyncSegment(f *ffinfo.File) error {
	segmentStart, err := getSegmentStart(f)
	if err != nil {
		return err
	}

	fmt.Printf("> Making desynced segment for \"%v\"\n", filepath.Base(f.Format.Filename))
	fmt.Printf("> segmentStart: %v; segmentDuration: %v\n\n", segmentStart, flagSegmentDuration)

	command := []string{
		"-ss", fmt.Sprintf("%v", segmentStart),
		"-i", f.Format.Filename,
		"-ss", fmt.Sprintf("%v", segmentStart),
		"-itsoffset", "1",
		"-i", f.Format.Filename,
		"-map", "0:v",
		"-vcodec", "libx264",
		"-pix_fmt", "yuv420p",
		"-preset", "medium",
		"-crf", "18",
		"-acodec", "aac",
		"-map", "1:a",
		"-ab", "256k",
		"-ar", "48000",
		"-t", fmt.Sprintf("%v", flagSegmentDuration),
		"-loglevel", "error",
		"-stats",
		"-y",
		"-hide_banner",
		strings.TrimSuffix(f.Format.Filename, filepath.Ext(f.Format.Filename)) + "_desync.mp4",
	}

	fmt.Printf("ffmpeg %v\n\n", strings.Join(command, " "))

	cmd := exec.Command("ffmpeg", command...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return err
	}

	return nil
}
