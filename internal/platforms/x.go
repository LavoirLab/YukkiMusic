/*
 * ● YukkiMusic
 * ○ A high-performance engine for streaming music in Telegram voicechats.
 *
 * Copyright (C) 2026 TheTeamVivek
 *
 * This program is free software: you can redistribute it and/or modify it under the
 * terms of the GNU General Public License as published by the Free Software Foundation,
 * either version 3 of the License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful, but WITHOUT ANY
 * WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A
 * PARTICULAR PURPOSE. See the GNU General Public License for more details.
 *
 * Repository: https://github.com/TheTeamVivek/YukkiMusic
 */

package platforms

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	td "github.com/AshokShau/gotdbot"

	state "yukkimusic/internal/core/models"
	"yukkimusic/internal/utils"
)

// PlatformShruti downloads YouTube tracks through the Shruti API.
const PlatformShruti state.PlatformName = "Shruti"

// shrutiAPIURL is the Shruti API download endpoint.
const shrutiAPIURL = "https://api.shrutibots.site/download"

// apiKey is the Shruti API key read from the environment. It is never
// hardcoded so the source stays secret-free.
var apiKey = os.Getenv("SHRUTI_API_KEY")

// ShrutiPlatform resolves nothing on its own, it only downloads YouTube
// tracks through the Shruti API.
type ShrutiPlatform struct{}

func init() {
	Register(&ShrutiPlatform{})
}

func (s *ShrutiPlatform) Name() state.PlatformName { return PlatformShruti }
func (s *ShrutiPlatform) Priority() int            { return 85 }

func (s *ShrutiPlatform) CanGet(_ string) bool { return false }

func (s *ShrutiPlatform) Get(_ string, _ bool) ([]*state.Track, error) {
	return nil, errors.New("shruti platform does not support resolving tracks")
}

func (s *ShrutiPlatform) CanDownload(source state.PlatformName) bool {
	if apiKey == "" {
		return false
	}
	return source == PlatformYouTube
}

// Download fetches the track from the Shruti API and returns the local path.
func (s *ShrutiPlatform) Download(
	ctx context.Context,
	track *state.Track,
	_ *td.Message,
) (string, error) {
	if apiKey == "" {
		return "", errors.New("SHRUTI_API_KEY not configured")
	}
	if track.ID == "" {
		return "", errors.New("empty track id")
	}

	if p := findFile(track); p != "" {
		utils.Log.Debug("Shruti: cache hit " + p)
		return p, nil
	}

	mediaType := "audio"
	ext := ".mp3"
	if track.Video {
		mediaType = "video"
		ext = ".mp4"
	}

	path := getPath(track, ext)

	// The API accepts the bare video id, fall back to the full watch URL
	// in case it only resolves complete links.
	ids := []string{track.ID}
	if track.URL != "" {
		ids = append(ids, track.URL)
	}

	var lastErr error
	for _, id := range ids {
		err := s.fetch(ctx, id, mediaType, path)
		if err == nil {
			return path, nil
		}
		if isDownloadCancelled(err) {
			return "", err
		}
		lastErr = err
	}

	return "", lastErr
}

// fetch downloads one file from the Shruti API into path.
func (s *ShrutiPlatform) fetch(
	ctx context.Context,
	id, mediaType, path string,
) error {
	apiURL := fmt.Sprintf(
		"%s?url=%s&type=%s&api_key=%s",
		shrutiAPIURL,
		url.QueryEscape(id),
		url.QueryEscape(mediaType),
		url.QueryEscape(apiKey),
	)

	r, err := rc.R().
		SetContext(ctx).
		SetResponseSaveFileName(path).
		Get(apiURL)
	if err != nil {
		os.Remove(path)
		if isDownloadCancelled(err) {
			return err
		}
		return sanitizeAPIError(
			fmt.Errorf("shruti request failed: %w", err),
			apiKey,
		)
	}

	if r.IsStatusFailure() {
		os.Remove(path)
		return sanitizeAPIError(
			fmt.Errorf("shruti api returned %d", r.StatusCode()),
			apiKey,
		)
	}

	// The API answers with a JSON payload when it refuses the request.
	if strings.Contains(r.Header().Get("Content-Type"), "application/json") {
		os.Remove(path)
		return sanitizeAPIError(
			fmt.Errorf("shruti api error: %s", truncate(r.String(), 200)),
			apiKey,
		)
	}

	if !fileExists(path) {
		os.Remove(path)
		return errors.New("shruti api returned an empty file")
	}

	return nil
}

// truncate shortens a string to at most n runes.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}