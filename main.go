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

const (
	defaultGloveDirectory   = "/Volumes"
	leftGloveFilename       = "GLV80LHBOOT"
	rightGloveFilename      = "GLV80RHBOOT"
	tempArtifactZipFilename = "temp.zip"
	artifactFilename        = "glove80.uf2"
)

func verifyGlovesConnected(glovePath string) (err error) {
	leftGlovePath := filepath.Join(glovePath, leftGloveFilename)
	lconnected, err := exists(leftGlovePath)
	if err != nil {
		return err
	}

	rightGlovePath := filepath.Join(glovePath, rightGloveFilename)
	rconnected, err := exists(rightGlovePath)
	if err != nil {
		return err
	}

	if !lconnected {
		err = errors.Join(err, fmt.Errorf("left glove not connected in bootloader mass storage device mode (%s)", leftGlovePath))
	}

	if !rconnected {
		err = errors.Join(err, fmt.Errorf("right glove not connected in bootloader mass storage device mode (%s)", rightGlovePath))
	}

	return err
}

func getLatestArtifact(client *github.Client, owner string, repo string) (*github.Artifact, error) {
	artifacts, _, err := client.Actions.ListArtifacts(context.Background(), owner, repo, nil)
	if err != nil {
		return nil, err
	}

	if len(artifacts.Artifacts) < 1 {
		return nil, errors.New("no artifacts to flash")
	}

	slices.SortFunc(artifacts.Artifacts, func(i, j *github.Artifact) int {
		return j.CreatedAt.GetTime().Compare(*i.CreatedAt.GetTime())
	})

	latestArtifact := artifacts.Artifacts[0]

	return latestArtifact, nil
}

func downloadArtifact(client *github.Client, owner string, repo string, artifactID int64) error {
	artifactURL, _, err := client.Actions.DownloadArtifact(context.Background(), owner, repo, int64(artifactID), 1)
	if err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	downloadDestination := filepath.Join(cwd, tempArtifactZipFilename)
	if err := downloadFile(downloadDestination, artifactURL.String()); err != nil {
		return err
	}

	defer os.Remove(downloadDestination)

	if err := unzip(downloadDestination, cwd); err != nil {
		return err
	}

	return nil
}

type FlashConfig struct {
	owner     string
	repo      string
	glovePath string
}

func flash(config FlashConfig) (err error) {
	err = verifyGlovesConnected(config.glovePath)
	if err != nil {
		return err
	}

	fmt.Printf("Downloading latest uf2 artifact from %s/%s\n", config.owner, config.repo)

	token := os.Getenv("GITHUB_PAT")
	client := github.NewClient(nil).WithAuthToken(token)

	fmt.Println("Successfully authenticated with GitHub")

	artifact, err := getLatestArtifact(client, config.owner, config.repo)
	if err != nil {
		return fmt.Errorf("error fetching latest uf2 artifact ID: %w", err)
	}

	fmt.Printf("Latest artifact is %d created at %v\n", artifact.GetID(), artifact.GetCreatedAt())

	if artifact.GetExpired() {
		return errors.New("latest artifact is expired")
	}

	if err = downloadArtifact(client, config.owner, config.repo, artifact.GetID()); err != nil {
		return fmt.Errorf("error downloading latest uf2 artifact: %w", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	artifactPath := filepath.Join(cwd, artifactFilename)

	// TODO: What if it's not extracted with this name?
	// Rename the file to a known one after it's extracted?
	defer os.Remove(artifactPath)

	leftGlovePath := filepath.Join(config.glovePath, leftGloveFilename, artifactFilename)
	fmt.Printf("Copying uf2 to left glove at %v\n", leftGlovePath)
	err = copy(artifactPath, leftGlovePath)
	if err != nil {
		return err
	}

	rightGlovePath := filepath.Join(config.glovePath, rightGloveFilename, artifactFilename)
	fmt.Printf("Copying uf2 to right glove at %v\n", rightGlovePath)
	err = copy(artifactPath, rightGlovePath)
	if err != nil {
		return err
	}

	fmt.Println("Success!")
	return nil
}

func main() {
	cmd := &cli.Command{
		Name:  "flash",
		Usage: "Utility for flashing new config to a Glove80 from a ZMK repo",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "directory",
				Usage:       "Specify the directory where the Glove80 bootloader storage directories will appear",
				Value:       defaultGloveDirectory,
				Aliases:     []string{"d"},
				DefaultText: defaultGloveDirectory,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) (err error) {
			if err = godotenv.Load(); err != nil {
				return err
			}

			owner, set := os.LookupEnv("OWNER")
			if !set {
				return errors.New("no OWNER provided in env")
			}

			repo, set := os.LookupEnv("REPO")
			if !set {
				return errors.New("no REPO provided in env")
			}

			_, set = os.LookupEnv("GITHUB_PAT")
			if !set {
				return errors.New("no GITHUB_PAT provided in env")
			}

			err = flash(FlashConfig{owner: owner, repo: repo, glovePath: cmd.String("directory")})
			return err
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
