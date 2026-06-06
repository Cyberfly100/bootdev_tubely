package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os/exec"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
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

func generatePresigedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(s3Client)
	presignGetObjectParams := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}
	presignGetObjectOutput, err := presignClient.PresignGetObject(context.Background(), presignGetObjectParams, s3.WithPresignExpires(expireTime))
	if err != nil {
		return "", err
	}
	return presignGetObjectOutput.URL, nil
}

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
	if video.VideoURL == nil {
		return video, nil
	}
	urlComponents := strings.Split(*video.VideoURL, ",")
	if len(urlComponents) != 2 {
		err := cfg.db.DeleteVideo(video.ID)
		if err != nil {
			return video, fmt.Errorf("couldn't delete video with invalid URL from database: %w", err)
		}
		return video, fmt.Errorf("invalid video URL format")
	}
	bucket := urlComponents[0]
	key := urlComponents[1]

	signedURL, err := generatePresigedURL(cfg.s3Client, bucket, key, 15*time.Minute)
	if err != nil {
		return video, fmt.Errorf("failed to generate signed URL: %w", err)
	}
	video.VideoURL = &signedURL
	return video, nil
}
