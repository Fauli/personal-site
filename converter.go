package main

import (
	"bufio"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	inputFile := "posts.txt" // Input file containing post-name: folder mapping
	contentDir := "./content/posts/"

	// Open input file
	file, err := os.Open(inputFile)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue // Skip empty lines or comments
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			fmt.Println("Skipping invalid line:", line)
			continue
		}

		postName := strings.TrimSpace(parts[0])
		folder := strings.TrimSpace(parts[1])
		postPath := filepath.Join(contentDir, postName+".md")
		readmePath := filepath.Join(folder, "README.md")
		attachmentsPath := filepath.Join(folder, ".attachments")
		imagesPath := filepath.Join(folder, "images")
		destAssetsPath := filepath.Join(contentDir, postName)

		// Ensure the post file exists
		if _, err := os.Stat(postPath); os.IsNotExist(err) {
			cmd := exec.Command("hugo", "new", "content/posts/"+postName+".md")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				fmt.Println("Failed to create Hugo post:", err)
				continue
			}
		}

		// Read existing post content and extract header
		existingContent, err := os.ReadFile(postPath)
		if err != nil {
			fmt.Println("Error reading post file:", err)
			continue
		}

		headerRegex := regexp.MustCompile(`(?s)^\+\+\+.*?\+\+\+`)
		header := ""
		matches := headerRegex.Find(existingContent)
		if matches != nil {
			header = string(matches)
		}

		// Read README.md
		readmeContentBytes, err := os.ReadFile(readmePath)
		if err != nil {
			fmt.Println("Warning: No README.md found in", folder)
			readmeContentBytes = []byte("\n")
		}

		// Ensure assets directory exists
		os.MkdirAll(destAssetsPath, os.ModePerm)

		// Copy images and attachments
		if _, err := os.Stat(imagesPath); err == nil {
			copyAndCompressImages(imagesPath, destAssetsPath)
		}
		if _, err := os.Stat(attachmentsPath); err == nil {
			copyDir(attachmentsPath, destAssetsPath)
		}

		// Convert image references in README.md to Hugo format
		readmeContent := string(readmeContentBytes)
		imageRegex := regexp.MustCompile(`!\[(.*?)\]\((.*?)\)`) // Matches ![alt](path)
		readmeContent = imageRegex.ReplaceAllStringFunc(readmeContent, func(match string) string {
			groups := imageRegex.FindStringSubmatch(match)
			if len(groups) != 3 {
				return match // Return unchanged if the pattern doesn't match exactly
			}
			altText := groups[1]
			imagePath := groups[2]
			imageName := filepath.Base(imagePath)
			return fmt.Sprintf(`{{< image src="/public/posts/%s/%s" alt="%s" position="center" style="border-radius: 4px;" >}}`, postName, imageName, altText)
		})

		// Write updated post file, keeping only the header and new content
		outputFile, err := os.Create(postPath)
		if err != nil {
			fmt.Println("Error writing post file:", err)
			continue
		}
		defer outputFile.Close()

		writer := bufio.NewWriter(outputFile)
		if header != "" {
			writer.WriteString(header + "\n\n")
		}
		writer.WriteString(readmeContent + "\n")
		writer.Flush()
	}
}

func copyAndCompressImages(src, dest string) {
	os.MkdirAll(dest, os.ModePerm)
	files, err := os.ReadDir(src)
	if err != nil {
		fmt.Println("Error reading directory:", err)
		return
	}
	for _, file := range files {
		srcPath := filepath.Join(src, file.Name())
		destPath := filepath.Join(dest, file.Name())
		if compressImage(srcPath, destPath) {
			continue
		}
		copyFile(srcPath, destPath)
	}
}

func compressImage(src, dest string) bool {
	file, err := os.Open(src)
	if err != nil {
		return false
	}
	defer file.Close()

	info, _ := file.Stat()
	if info.Size() < 1024*1024 {
		return false
	}

	img, format, err := image.Decode(file)
	if err != nil {
		return false
	}

	destFile, err := os.Create(dest)
	if err != nil {
		return false
	}
	defer destFile.Close()

	if format == "jpeg" {
		fmt.Println("shrinking jpeg")
		jpeg.Encode(destFile, img, &jpeg.Options{Quality: 75})
	} else if format == "png" {
		fmt.Println("Shrinking png")
		png.Encode(destFile, img)
	}

	return true
}

func copyDir(src, dest string) {
	os.MkdirAll(dest, os.ModePerm)
	files, err := os.ReadDir(src)
	if err != nil {
		fmt.Println("Error reading directory:", err)
		return
	}
	for _, file := range files {
		srcPath := filepath.Join(src, file.Name())
		destPath := filepath.Join(dest, file.Name())
		copyFile(srcPath, destPath)
	}
}

func copyFile(src, dest string) {
	srcFile, err := os.Open(src)
	if err != nil {
		fmt.Println("Error opening source file:", err)
		return
	}
	defer srcFile.Close()
	destFile, err := os.Create(dest)
	if err != nil {
		fmt.Println("Error creating destination file:", err)
		return
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		fmt.Println("Error copying file:", err)
	}
}
