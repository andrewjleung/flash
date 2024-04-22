package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"

	"github.com/google/go-github/v61/github"
	"github.com/joho/godotenv"
	"github.com/urfave/cli/v3"
	"golang.org/x/net/context"
)

const defaultGloveDirectory = "/Volumes"
const leftGloveFilename = "GLV80LHBOOT"
const rightGloveFilename = "GLV80RHBOOT"
const tempArtifactZipFilename = "temp.zip"
const artifactFilename = "glove80.uf2"

func verifyGlovesConnected(glovePath string) (err error) {
	lconnected, err := exists(filepath.Join(glovePath, leftGloveFilename))
	if err != nil {
		return
	}

	rconnected, err := exists(filepath.Join(glovePath, rightGloveFilename))
	if err != nil {
		return
	}

	if !lconnected {
		err = errors.Join(err, errors.New("Left glove not connected in bootloader mass storage device mode"))
	}

	if !rconnected {
		err = errors.Join(err, errors.New("Right glove not connected in bootloader mass storage device mode"))
	}

	return
}

func getLatestArtifactId(client *github.Client, owner string, repo string) (int, error) {
	artifacts, _, err := client.Actions.ListArtifacts(context.Background(), owner, repo, nil)

	if err != nil {
		return 0, err
	}

	if len(artifacts.Artifacts) < 1 {
		return 0, errors.New("No artifacts to flash")
	}

	slices.SortFunc(artifacts.Artifacts, func(i, j *github.Artifact) int {
		return i.CreatedAt.GetTime().Compare(*j.CreatedAt.GetTime())
	})

	latestArtifact := artifacts.Artifacts[0]
	return int(latestArtifact.GetID()), nil
}

func downloadArtifact(client *github.Client, owner string, repo string, artifactId int64) error {
	artifactUrl, _, err := client.Actions.DownloadArtifact(context.Background(), owner, repo, int64(artifactId), 1)

	if err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	downloadDestination := filepath.Join(cwd, tempArtifactZipFilename)
	if err := downloadFile(downloadDestination, artifactUrl.String()); err != nil {
		return err
	}

	defer os.Remove(downloadDestination)

	if err := unzip(downloadDestination, cwd); err != nil {
		return err
	}

	return nil
}

func flash(owner string, repo string, glovePath string) (err error) {
	err = verifyGlovesConnected(glovePath)
	if err != nil {
		return
	}

	token := os.Getenv("GITHUB_PAT")
	client := github.NewClient(nil).WithAuthToken(token)

	artifactId, err := getLatestArtifactId(client, owner, repo)
	if err != nil {
		return
	}

	if err = downloadArtifact(client, owner, repo, int64(artifactId)); err != nil {
		return
	}

	cwd, err := os.Getwd()
	if err != nil {
		return
	}

	artifactPath := filepath.Join(cwd, artifactFilename)

	// TODO: What if it's not extracted with this name?
	// Rename the file to a known one after it's extracted?
	defer os.Remove(artifactPath)

	err = copy(artifactPath, filepath.Join(glovePath, leftGloveFilename, artifactFilename))
	if err != nil {
		return
	}

	err = copy(artifactPath, filepath.Join(glovePath, rightGloveFilename, artifactFilename))
	if err != nil {
		return
	}

	return nil
}

func main() {
	directory := defaultGloveDirectory

	cmd := &cli.Command{
		Name:  "flash",
		Usage: "Utility for flashing new config to a Glove80 from a ZMK repo",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "directory",
				Usage:       "Specify the directory where the Glove80 bootloader storage directories will appear, defaults to the `/Volumes` directory",
				Destination: &directory,
				Aliases:     []string{"d"},
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if err := godotenv.Load(); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}

			owner, set := os.LookupEnv("OWNER")
			if !set {
				fmt.Fprintln(os.Stderr, "No OWNER provided in env")
				os.Exit(1)
			}

			repo, set := os.LookupEnv("REPO")
			if !set {
				fmt.Fprintln(os.Stderr, "No REPO provided in env")
				os.Exit(1)
			}

			_, set = os.LookupEnv("GITHUB_PAT")
			if !set {
				fmt.Fprintln(os.Stderr, "No GITHUB_PAT provided in env")
				os.Exit(1)
			}

			err := flash(owner, repo, directory)
			return err
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
