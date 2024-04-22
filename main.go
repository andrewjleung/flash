package main

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"

	"github.com/google/go-github/v61/github"
	"golang.org/x/net/context"

	"github.com/joho/godotenv"
)

const owner = "andrewjleung"
const repo = "zmk-config"

// const leftGloveLocation = "/Volumes/GLV80LHBOOT"
const leftGloveLocation = "/Users/andrewleung/Desktop/fake_glove/GLV80LHBOOT"

// const rightGloveLocation = "/Volumes/GLV80RHBOOT"
const rightGloveLocation = "/Users/andrewleung/Desktop/fake_glove/GLV80RHBOOT"
const tempPath = "./temp.zip"
const extractedPath = "./glove80.uf2"

func downloadFile(filepath string, url string) error {
	// Retrieve the file w/ HTTP.
	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	// Create the file.
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}

	defer out.Close()

	// Copy the file contents to the file.
	_, err = io.Copy(out, resp.Body)

	return err
}

func extract(f *zip.File, dest string) error {
	// Open the unzipped file.
	rc, err := f.Open()
	if err != nil {
		return err
	}

	defer rc.Close()

	path := filepath.Join(dest, f.Name)

	// Assuming the file is not a directory, make any non-existing parent directories.
	os.MkdirAll(filepath.Dir(path), f.Mode())

	// Create the file.
	nf, err := os.Create(path)
	if err != nil {
		return err
	}

	defer nf.Close()

	// Copy the file contents to it.
	_, err = io.Copy(nf, rc)
	if err != nil {
		return err
	}

	return nil
}

func copy(from, to string) error {
	r, err := os.Open(from)
	if err != nil {
		return err
	}

	defer r.Close()

	w, err := os.Create(to)
	if err != nil {
		return err
	}

	defer w.Close()

	if _, err := io.Copy(w, r); err != nil {
		return err
	}

	return nil
}

func unzip(filepath string, dest string) error {
	r, err := zip.OpenReader(filepath)
	if err != nil {
		return err
	}

	defer r.Close()

	for _, f := range r.File {
		if err := extract(f, dest); err != nil {
			return err
		}
	}

	return nil
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func glovesConnected() (lconnected bool, rconnected bool, err error) {
	lconnected, lerr := exists(leftGloveLocation)
	rconnected, rerr := exists(rightGloveLocation)
	err = errors.Join(lerr, rerr)
	return
}

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// Verify that both gloves are connected in bootloader mass storage device mode
	lconnected, rconnected, err := glovesConnected()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if !(lconnected && rconnected) {
		if !lconnected {
			fmt.Println("Left glove not connected in bootloader mass storage device mode")
		}

		if !rconnected {
			fmt.Println("Right glove not connected in bootloader mass storage device mode")
		}

		os.Exit(1)
	}

	// Get the ID of the latest built uf2 artifact
	token := os.Getenv("GITHUB_PAT")
	client := github.NewClient(nil).WithAuthToken(token)

	artifacts, _, err := client.Actions.ListArtifacts(context.Background(), owner, repo, nil)

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if len(artifacts.Artifacts) < 1 {
		fmt.Fprintln(os.Stderr, "error: no artifacts to flash")
		os.Exit(1)
	}

	slices.SortFunc(artifacts.Artifacts, func(i, j *github.Artifact) int {
		return i.CreatedAt.GetTime().Compare(*j.CreatedAt.GetTime())
	})

	latestArtifact := artifacts.Artifacts[0]

	// Download the latest zipped built uf2 artifact
	artifactUrl, _, err := client.Actions.DownloadArtifact(context.Background(), owner, repo, *latestArtifact.ID, 1)

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if err := downloadFile(tempPath, artifactUrl.String()); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	defer os.Remove(tempPath)

	// Unzip the uf2 artifact
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if err := unzip(tempPath, cwd); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// TODO: What if it's not extracted with this name?
	defer os.Remove(extractedPath)

	// Copy the uf2 to each glove
	copy(extractedPath, filepath.Join(leftGloveLocation, extractedPath))
	copy(extractedPath, filepath.Join(rightGloveLocation, extractedPath))
}
