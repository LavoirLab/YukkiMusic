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

func shrutiAPIKey() string {
	return "ShrutiBots1P9oduKISPWG6uuKbAmg"
}

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
	return shrutiAPIKey() != "" && source == PlatformYouTube
}

// Download fetches the track from the Shruti API and returns the local path.
func (s *ShrutiPlatform) Download(
	ctx context.Context,
	track *state.Track,
	_ *td.Message,
) (string, error) {
	key := shrutiAPIKey()
	if key == "" {
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
		err := s.fetch(ctx, id, mediaType, path, key)
		if err == nil {
			return path, nil
		}
		if isDownloadCancelled(err) || (ctx != nil && ctx.Err() != nil) {
			return "", context.Canceled
		}
		lastErr = err
	}

	return "", lastErr
}

// fetch downloads one file from the Shruti API into path.
func (s *ShrutiPlatform) fetch(
	ctx context.Context,
	id, mediaType, path, key string,
) error {
	apiURL := fmt.Sprintf(
		"%s?url=%s&type=%s&api_key=%s",
		shrutiAPIURL,
		url.QueryEscape(id),
		url.QueryEscape(mediaType),
		url.QueryEscape(key),
	)

	r, err := rc.R().
		SetContext(ctx).
		SetResponseSaveFileName(path).
		Get(apiURL)
	if err != nil {
		_ = os.Remove(path)
		if isDownloadCancelled(err) || (ctx != nil && ctx.Err() != nil) {
			return context.Canceled
		}
		return sanitizeAPIError(
			fmt.Errorf("shruti request failed: %w", err),
			key,
		)
	}

	if r.IsStatusFailure() {
		_ = os.Remove(path)
		return sanitizeAPIError(
			fmt.Errorf("shruti api returned %d", r.StatusCode()),
			key,
		)
	}

	// The API answers with a JSON payload when it refuses the request.
	if strings.Contains(r.Header().Get("Content-Type"), "application/json") {
		_ = os.Remove(path)
		return sanitizeAPIError(
			fmt.Errorf("shruti api error: %s", truncate(r.String(), 200)),
			key,
		)
	}

	if !fileExists(path) {
		_ = os.Remove(path)
		return errors.New("shruti api returned an empty file")
	}

	return nil
}

func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}

// PlatformRiteshyt downloads YouTube tracks through the Riteshyt API
// as a fallback when Shruti is unavailable.
const PlatformRiteshyt state.PlatformName = "Riteshyt"

// riteshytAPIURL is the Riteshyt API download endpoint.
const riteshytAPIURL = "https://api.riteshyt.in/download"

func riteshytAPIKey() string {
	return os.Getenv("RITESHYT_API_KEY")
}

// RiteshytPlatform resolves nothing on its own, it only downloads YouTube
// tracks through the Riteshyt API.
type RiteshytPlatform struct{}

func init() {
	Register(&RiteshytPlatform{})
}

func (r *RiteshytPlatform) Name() state.PlatformName { return PlatformRiteshyt }
func (r *RiteshytPlatform) Priority() int            { return 80 }

func (r *RiteshytPlatform) CanGet(_ string) bool { return false }

func (r *RiteshytPlatform) Get(_ string, _ bool) ([]*state.Track, error) {
	return nil, errors.New("riteshyt platform does not support resolving tracks")
}

func (r *RiteshytPlatform) CanDownload(source state.PlatformName) bool {
	return riteshytAPIKey() != "" && source == PlatformYouTube
}

// Download fetches the track from the Riteshyt API and returns the local path.
func (r *RiteshytPlatform) Download(
	ctx context.Context,
	track *state.Track,
	_ *td.Message,
) (string, error) {
	key := riteshytAPIKey()
	if key == "" {
		return "", errors.New("RITESHYT_API_KEY not configured")
	}
	if track.ID == "" {
		return "", errors.New("empty track id")
	}

	if p := findFile(track); p != "" {
		utils.Log.Debug("Riteshyt: cache hit " + p)
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
		err := r.fetch(ctx, id, mediaType, path, key)
		if err == nil {
			return path, nil
		}
		if isDownloadCancelled(err) || (ctx != nil && ctx.Err() != nil) {
			return "", context.Canceled
		}
		lastErr = err
	}

	return "", lastErr
}

// fetch downloads one file from the Riteshyt API into path.
func (r *RiteshytPlatform) fetch(
	ctx context.Context,
	id, mediaType, path, key string,
) error {
	apiURL := fmt.Sprintf(
		"%s?query=%s&dl_type=%s&api_key=%s",
		riteshytAPIURL,
		url.QueryEscape(id),
		url.QueryEscape(mediaType),
		url.QueryEscape(key),
	)

	resp, err := rc.R().
		SetContext(ctx).
		SetResponseSaveFileName(path).
		Get(apiURL)
	if err != nil {
		_ = os.Remove(path)
		if isDownloadCancelled(err) || (ctx != nil && ctx.Err() != nil) {
			return context.Canceled
		}
		return sanitizeAPIError(
			fmt.Errorf("riteshyt request failed: %w", err),
			key,
		)
	}

	if resp.IsStatusFailure() {
		_ = os.Remove(path)
		return sanitizeAPIError(
			fmt.Errorf("riteshyt api returned %d", resp.StatusCode()),
			key,
		)
	}

	// The API answers with a JSON payload when it refuses the request.
	if strings.Contains(resp.Header().Get("Content-Type"), "application/json") {
		_ = os.Remove(path)
		return sanitizeAPIError(
			fmt.Errorf("riteshyt api error: %s", truncate(resp.String(), 200)),
			key,
		)
	}

	if !fileExists(path) {
		_ = os.Remove(path)
		return errors.New("riteshyt api returned an empty file")
	}

	return nil
}
