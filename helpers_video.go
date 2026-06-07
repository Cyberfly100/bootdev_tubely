package main

import (
	"bytes"
	"encoding/json"
	"math"
	"os/exec"
)

func getVideoAspectRatio(filePath string) (string, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	buffer := bytes.Buffer{}
	cmd.Stdout = &buffer
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	type ffprobeOutput struct {
		Streams []struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"streams"`
	}
	var output ffprobeOutput
	err = json.Unmarshal(buffer.Bytes(), &output)
	if err != nil {
		return "", err
	}
	if len(output.Streams) == 0 {
		return "", nil
	}
	width := output.Streams[0].Width
	height := output.Streams[0].Height

	aspectRatio := float64(width) / float64(height)
	tolerance := 0.03

	switch {
	case math.Abs(aspectRatio-16.0/9.0) < tolerance:
		return "16:9", nil
	case math.Abs(aspectRatio-9.0/16.0) < tolerance:
		return "9:16", nil
	case math.Abs(aspectRatio-4.0/3.0) < tolerance:
		return "4:3", nil
	default:
		return "other", nil
	}
}

func processVideoForFastStart(filePath string) (string, error) {
	outputPath := filePath + ".processing"
	cmd := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", outputPath)
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return outputPath, nil
}
