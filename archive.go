package nsfw

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"

	tg "github.com/galeone/tfgo"
	"github.com/sirupsen/logrus"
)

type Path string

func GetLocalModelPath() (Path, error) {
	cached, err := getLatestCached(DefaultCachePath)
	if err != nil {
		return "", err
	}
	return cached.getModelPath(), nil
}

func GetLatestModelPath() (Path, error) {
	// 0) Hard override via env
	if p := envStr("NSFW_MODEL_PATH", ""); p != "" {
		if looksLikeSavedModel(p) {
			logrus.Infof("Using NSFW model at NSFW_MODEL_PATH=%s", p)
			return Path(p), nil
		}
		return "", fmt.Errorf("NSFW_MODEL_PATH set but not a SavedModel: %s", p)
	}

	// 1) Prefer cache first (avoid network if possible)
	cached, cachedErr := getLatestCached(DefaultCachePath)
	if cachedErr == nil && looksLikeSavedModel(cached.getModelPath().String()) && !envBool("NSFW_MODEL_FORCE_UPDATE") {
		logrus.Infof("Using cached NSFW model '%s'", cached.TagName)
		return cached.getModelPath(), nil
	}

	// 2) Try remote (or fallback if remote disabled/fails)
	logrus.Info("Resolving latest NSFW release (remote or fallback)")
	latest, _ := getLatestReleaseInfo() // never fatal; returns fallback on problems

	// If we had a cache, only update if latest is newer; otherwise keep cache
	if cachedErr == nil && !latest.isNewer(cached) && looksLikeSavedModel(cached.getModelPath().String()) && !envBool("NSFW_MODEL_FORCE_UPDATE") {
		logrus.Info("Cached model is up to date or remote version unknown; sticking with cache")
		return cached.getModelPath(), nil
	}

	// 3) Download + unpack chosen release (latest or fallback)
	logrus.Infof("Preparing model '%s'", latest.TagName)
	latest.LocalPath = filepath.Join(DefaultCachePath, latest.getTagPath())
	if err := os.MkdirAll(latest.LocalPath, 0o770); err != nil {
		return "", err
	}

	if err := latest.download(latest.getZipPath()); err != nil {
		// If download failed but we had a valid cached model, return cache
		if cachedErr == nil && looksLikeSavedModel(cached.getModelPath().String()) {
			logrus.Warnf("Download failed (%v); using cached model '%s'", err, cached.TagName)
			return cached.getModelPath(), nil
		}
		return "", fmt.Errorf("failed to download NSFW model: %w", err)
	}

	if err := latest.saveMeta(latest.getMetaPath()); err != nil {
		logrus.Warnf("Failed to write meta.json: %v", err)
	}

	logrus.Info("Unzipping model")
	if err := latest.unpack(); err != nil {
		return "", err
	}

	mp := latest.getModelPath()
	if !looksLikeSavedModel(mp.String()) {
		return "", fmt.Errorf("unzipped model at %s does not look like a TensorFlow SavedModel", mp)
	}

	logrus.Info("Model cached successfully")
	return mp, nil
}

func (i releaseInfo) unpack() error {
	if err := i.unzip(); err != nil {
		return err
	}
	if err := i.cleanup(); err != nil {
		logrus.Errorf("Unable to cleanup model file at '%s'", i.getZipPath().String())
	}
	return nil
}

func (i releaseInfo) unzip() error {
	r, err := zip.OpenReader(string(i.getZipPath()))
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		filePath := filepath.Join(i.getModelFolder().String(), f.Name)

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(filePath, os.ModePerm); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func (i releaseInfo) cleanup() error {
	return os.Remove(i.getZipPath().String())
}

func (p Path) GetModel() *tg.Model {
	return tg.LoadModel(string(p), []string{"serve"}, nil)
}

func (p Path) String() string {
	return string(p)
}
