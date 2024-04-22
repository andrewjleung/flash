package main

import (
	"archive/zip"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

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
