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
const tempPath = "./temp.zip"
const extractedPath = "./glove80.uf2"

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

func flash(owner string, repo string, glovePath string) (err error) {
	// Verify that both gloves are connected in bootloader mass storage device mode
	err = verifyGlovesConnected(glovePath)
	if err != nil {
		return
	}

	// Get the ID of the latest built uf2 artifact
	token := os.Getenv("GITHUB_PAT")
	client := github.NewClient(nil).WithAuthToken(token)

	artifacts, _, err := client.Actions.ListArtifacts(context.Background(), owner, repo, nil)

	if err != nil {
		return
	}

	if len(artifacts.Artifacts) < 1 {
		return errors.New("No artifacts to flash")
	}

	slices.SortFunc(artifacts.Artifacts, func(i, j *github.Artifact) int {
		return i.CreatedAt.GetTime().Compare(*j.CreatedAt.GetTime())
	})

	latestArtifact := artifacts.Artifacts[0]

	// Download the latest zipped built uf2 artifact
	artifactUrl, _, err := client.Actions.DownloadArtifact(context.Background(), owner, repo, *latestArtifact.ID, 1)

	if err != nil {
		return
	}

	if err := downloadFile(tempPath, artifactUrl.String()); err != nil {
		return err
	}

	defer os.Remove(tempPath)

	// Unzip the uf2 artifact
	cwd, err := os.Getwd()
	if err != nil {
		return
	}

	if err := unzip(tempPath, cwd); err != nil {
		return err
	}

	// TODO: What if it's not extracted with this name?
	defer os.Remove(extractedPath)

	// Copy the uf2 to each glove
	err = copy(extractedPath, filepath.Join(glovePath, leftGloveFilename, extractedPath))
	if err != nil {
		return
	}

	err = copy(extractedPath, filepath.Join(glovePath, rightGloveFilename, extractedPath))
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
