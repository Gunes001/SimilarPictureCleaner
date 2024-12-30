package main

import (
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/corona10/goimagehash"
)

type ImageInfo struct {
	Path string
	Hash *goimagehash.ImageHash
}

func main() {
	// Define the -d flag
	deleteFlag := flag.Bool("d", false, "Specify whether to delete similar images")
	flag.Parse()

	// Check the number of arguments
	if len(flag.Args()) < 2 {
		fmt.Println("Usage: go run main.go [-d] <directory> <similarity percentage>")
		return
	}

	dir := flag.Arg(0)
	similarityThreshold, err := parsePercentage(flag.Arg(1))
	if err != nil {
		fmt.Println("Invalid similarity percentage:", err)
		return
	}

	images, err := loadImages(dir)
	if err != nil {
		fmt.Println("Error loading images:", err)
		return
	}

	groups := findSimilarImages(images, similarityThreshold)

	// Display similar images
	for _, group := range groups {
		fmt.Println("Similar images:")
		for _, img := range group {
			fmt.Println(img.Path)
		}
		fmt.Println()
	}

	// Delete similar images if the -d flag is set
	if *deleteFlag {
		var totalSaved int64
		for _, group := range groups {
			if len(group) > 1 {
				saved, err := deleteSimilarImages(group)
				if err != nil {
					fmt.Println("Error deleting images:", err)
					continue
				}
				totalSaved += saved
			}
		}
		fmt.Printf("Total space saved: %d bytes\n", totalSaved)
	}
}

func parsePercentage(percentageStr string) (float64, error) {
	var percentage float64
	_, err := fmt.Sscanf(percentageStr, "%f", &percentage)
	if err != nil {
		return 0, err
	}
	if percentage < 0 || percentage > 100 {
		return 0, fmt.Errorf("percentage must be between 0 and 100")
	}
	return percentage / 100, nil
}

func loadImages(dir string) ([]ImageInfo, error) {
	var images []ImageInfo
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && (strings.HasSuffix(strings.ToLower(path), ".jpg") || strings.HasSuffix(strings.ToLower(path), ".jpeg") || strings.HasSuffix(strings.ToLower(path), ".png")) {
			img, err := loadImage(path)
			if err != nil {
				return err
			}
			hash, err := goimagehash.PerceptionHash(img)
			if err != nil {
				return err
			}
			images = append(images, ImageInfo{Path: path, Hash: hash})
		}
		return nil
	})
	return images, err
}

func loadImage(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	if strings.HasSuffix(strings.ToLower(path), ".jpg") || strings.HasSuffix(strings.ToLower(path), ".jpeg") {
		return jpeg.Decode(file)
	} else if strings.HasSuffix(strings.ToLower(path), ".png") {
		return png.Decode(file)
	}
	return nil, fmt.Errorf("unsupported image format")
}

func findSimilarImages(images []ImageInfo, similarityThreshold float64) [][]ImageInfo {
	var groups [][]ImageInfo
	used := make(map[int]bool)

	for i := 0; i < len(images); i++ {
		if used[i] {
			continue
		}
		group := []ImageInfo{images[i]}
		for j := i + 1; j < len(images); j++ {
			if used[j] {
				continue
			}
			distance, err := images[i].Hash.Distance(images[j].Hash)
			if err != nil {
				fmt.Println("Error calculating distance:", err)
				continue
			}
			similarity := 1 - float64(distance)/64.0
			if similarity >= similarityThreshold {
				group = append(group, images[j])
				used[j] = true
			}
		}
		if len(group) > 1 {
			groups = append(groups, group)
		}
	}

	return groups
}

func deleteSimilarImages(group []ImageInfo) (int64, error) {
	sort.Slice(group, func(i, j int) bool {
		distanceI, err := group[i].Hash.Distance(group[0].Hash)
		if err != nil {
			fmt.Println("Error calculating distance:", err)
			return false
		}
		distanceJ, err := group[j].Hash.Distance(group[0].Hash)
		if err != nil {
			fmt.Println("Error calculating distance:", err)
			return false
		}
		return distanceI < distanceJ
	})

	var totalSaved int64
	for i := 1; i < len(group); i++ {
		info, err := os.Stat(group[i].Path)
		if err != nil {
			return 0, err
		}
		totalSaved += info.Size()
		err = os.Remove(group[i].Path)
		if err != nil {
			return 0, err
		}
	}
	return totalSaved, nil
}
