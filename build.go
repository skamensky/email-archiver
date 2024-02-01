//go:build mage
// +build mage

package main

import (
	"errors"
	"fmt"
	"github.com/magefile/mage/sh"
	"github.com/skamensky/email-archiver/pkg/email"
	"github.com/skamensky/email-archiver/pkg/models"
	"github.com/skamensky/email-archiver/pkg/options"
	"github.com/tkrajina/typescriptify-golang-structs/typescriptify"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// should be run manually when models change and those changes need to be reflected in the frontend
func GenerateTypescriptModels() error {
	converter := typescriptify.New().
		Add(options.Options{}).Add(email.Email{}).Add(models.MailboxRecord{}).Add(models.MailboxEvent{}).AddEnum(models.AllEventTypes)

	err := converter.ConvertToFile(filepath.Join(".", "pkg", "web", "frontend", "src", "goGeneratedModels.ts"))
	if err != nil {
		return fmt.Errorf("could not generate typescript models: %w", err)
	}

	// delete files that match "goGeneratedModels.ts-*.backup in the current directory
	filesToDelete, err := filepath.Glob(filepath.Join(".", "goGeneratedModels.ts-*.backup"))
	if err != nil {
		return fmt.Errorf("could not find files to delete: %w", err)
	}
	for _, file := range filesToDelete {
		err = os.Remove(file)
		if err != nil {
			return fmt.Errorf("could not delete file %s: %w", file, err)
		}
	}

	return nil
}

// runs yarn install and yarn build in the frontend directory
func BuildReactFrontend() error {
	_, err := exec.LookPath("yarn")
	if err != nil {
		return fmt.Errorf("yarn must be installed")
	}

	err = os.Chdir(filepath.Join(".", "pkg", "web", "frontend"))
	if err != nil {
		return errors.Join(errors.New("could not change directory to frontend"), err)
	}
	defer func() {
		err = os.Chdir(filepath.Join("..", "..", ".."))
		if err != nil {
			panic(fmt.Errorf("could not change directory back to root: %w", err))
		}
	}()

	fmt.Println("Running yarn install")
	err = exec.Command("yarn", "install").Run()
	if err != nil {
		return fmt.Errorf("could not run yarn install: %w", err)
	}

	fmt.Println("Running yarn build")
	err = exec.Command("yarn", "build").Run()
	if err != nil {
		return fmt.Errorf("could not run yarn build: %w", err)
	}

	return nil
}

// builds the go app for the current platform or the platform specified by GOOS and GOARCH env vars
func BuildGoApp() error {
	goos := os.Getenv("GOOS")
	goarch := os.Getenv("GOARCH")

	if goos == "" {
		goos = runtime.GOOS
	}
	if goarch == "" {
		goarch = runtime.GOARCH
	}

	output := filepath.Join(".", "build", fmt.Sprintf("email-archiver-%s-%s", goos, goarch))
	if goos == "windows" {
		output += ".exe"
	}
	fmt.Println("Building for", goos, goarch, "to", output)
	// ft5 is a sqlite3 extension that allows for full text search
	err := sh.RunWith(
		map[string]string{
			"GOOS":   goos,
			"GOARCH": goarch,
		}, "go", "build", "--tags", "\"fts5\"", "-o", output, filepath.Join(".", "cmd", "main.go"))
	if err != nil {
		return fmt.Errorf("could not build go app: %w", err)
	}
	return nil
}

// builds the frontend and then the backend (with the frontend embedded)
func Build() error {
	err := BuildReactFrontend()
	if err != nil {
		return err
	}
	return BuildGoApp()
}
