package main

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// func onlyGetPictures(path string) []string {
// 	var pictures []string
// 	files, err := os.ReadDir(path)
// 	discard(err)

// 	for _, file := range files {
// 		if isPicture(file.Name()) {
// 			pictures = append(pictures, file.Name())
// 		}
// 	}
// 	return pictures
// }

// func isPicture(filename string) bool {
// 	ext := strings.ToLower(filepath.Ext(filename))
// 	pictureExtensions := []string{".jpg", ".jpeg", ".png", ".gif", ".webm", ".webp"}
// 	return slices.Contains(pictureExtensions, ext)
// }

func generateUnixFakeTime() int64 {
	threeMonthsAgo := time.Now().AddDate(0, -3, 0).UnixNano() / int64(time.Millisecond)
	oneYearAgo := time.Now().AddDate(-1, 0, 0).UnixNano() / int64(time.Millisecond)

	if threeMonthsAgo > oneYearAgo {
		threeMonthsAgo, oneYearAgo = oneYearAgo, threeMonthsAgo
	}

	return rand.Int63n(oneYearAgo-threeMonthsAgo) + threeMonthsAgo
}

func generateThumbnail(inputFile, outputFile string) error {
	var cmd *exec.Cmd
	inputExt := strings.ToLower(filepath.Ext(inputFile))

	baseArgs := []string{
		"-i", inputFile,
		"-vf", "scale=120:-1",
		"-q:v", "3",
		"-y",
	}

	if inputExt == ".webm" || inputExt == ".gif" {
		baseArgs = append([]string{"-ss", "00:00:01"}, baseArgs...)
		baseArgs = append(baseArgs, "-vframes", "1")
	}

	finalArgs := append(baseArgs, outputFile)

	log.Printf("    FFmpeg command: ffmpeg %s", strings.Join(finalArgs, " "))

	cmd = exec.Command("ffmpeg", finalArgs...)

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe for FFmpeg: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ffmpeg process: %w", err)
	}

	stderrBytes, readErr := io.ReadAll(stderrPipe)
	if readErr != nil {
		log.Printf("Warning: Could not read all FFmpeg stderr: %v", readErr)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("ffmpeg failed for %s: %w. FFmpeg Output: %s", filepath.Base(inputFile), err, strings.TrimSpace(string(stderrBytes)))
	}

	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		return fmt.Errorf("ffmpeg finished but output file '%s' not created. Output: %s", outputFile, strings.TrimSpace(string(stderrBytes)))
	}

	return nil
}
