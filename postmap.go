package main

import (
	"crypto/sha256"
	"fmt"
	"image/color"
	"strings"

	staticMap "github.com/Luzifer/go-staticmaps"
	"github.com/golang/geo/s2"
	"github.com/pkg/errors"
)

type postMapEnvelope struct {
	Center             postMapPoint   `json:"center"`
	Zoom               int            `json:"zoom"`
	Markers            postMapMarkers `json:"markers"`
	Width              int            `json:"width"`
	Height             int            `json:"height"`
	DisableAttribution bool           `json:"disable_attribution"`
	Overlays           postMapOverlay `json:"overlays"`
	Paths              postMapPaths   `json:"paths"`
}

func (p postMapEnvelope) toGenerateMapConfig() (generateMapConfig, error) {
	result := generateMapConfig{
		Center:             p.Center.getPoint(),
		Zoom:               p.Zoom,
		Width:              p.Width,
		Height:             p.Height,
		DisableAttribution: p.DisableAttribution,
	}

	if p.Width > mapMaxX || p.Height > mapMaxY {
		return generateMapConfig{}, errors.Errorf("map size exceeds allowed bounds of %dx%d", mapMaxX, mapMaxY)
	}

	var err error
	if result.Markers, err = p.Markers.toMarkers(); err != nil {
		return generateMapConfig{}, err
	}

	if result.Overlays, err = p.Overlays.toOverlays(); err != nil {
		return generateMapConfig{}, err
	}

	if result.Paths, err = p.Paths.toPaths(); err != nil {
		return generateMapConfig{}, err
	}

	return result, nil
}

type postMapMarker struct {
	Size  string       `json:"size"`
	Color string       `json:"color"`
	Coord postMapPoint `json:"coord"`
}

func (p postMapMarker) String() string {
	parts := []string{}

	if p.Size != "" {
		parts = append(parts, fmt.Sprintf("size:%s", p.Size))
	}

	if p.Color != "" {
		parts = append(parts, fmt.Sprintf("color:%s", p.Color))
	}

	parts = append(parts, p.Coord.String())
	return strings.Join(parts, "|")
}

type postMapMarkers []postMapMarker

func (p postMapMarkers) toMarkers() ([]marker, error) {
	raw := []string{}
	for _, pm := range p {
		raw = append(raw, pm.String())
	}

	return parseMarkerLocations(raw)
}

type postMapPath struct {
	Weight    float64        `json:"size"`
	Color     string         `json:"color"`
	Positions []postMapPoint `json:"positions"`
}

// func (p postMapPath) String() string {
// 	parts := []string{}

// 	// if p.Size != "" {
// 	// 	parts = append(parts, fmt.Sprintf("size:%s", p.Size))
// 	// }

// 	if p.Color != "" {
// 		parts = append(parts, fmt.Sprintf("color:%s", p.Color))
// 	}

// 	parts = append(parts, p.Coord.String())
// 	return strings.Join(parts, "|")
// }

type postMapPaths []postMapPath

func (ps postMapPaths) toPaths() ([]path, error) {
	result := []path{}
	for _, p := range ps {
		positions := []s2.LatLng{}
		for _, point := range p.Positions {
			positions = append(positions, point.getPoint())
		}

		pt := path{
			positions: positions,
			color:     color.RGBA{0xff, 0, 0, 0xff},
			weight:    5.0,
		}

		if p.Color != "" {
			var err error
			pt.color, err = staticMap.ParseColorString(p.Color)
			if err != nil {
				return nil, errors.Errorf("bad color name %q: %w", p.Color, err)
			}
		}

		if p.Weight != 0 {
			pt.weight = p.Weight
		}

		result = append(result, pt)
	}

	return result, nil
}

type postMapPoint struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

func (p postMapPoint) String() string {
	return fmt.Sprintf("%f,%f", p.Lat, p.Lon)
}

func (p postMapPoint) getPoint() s2.LatLng {
	return s2.LatLngFromDegrees(p.Lat, p.Lon)
}

type postMapOverlay []string

func (p postMapOverlay) toOverlays() ([]*staticMap.TileProvider, error) {
	result := []*staticMap.TileProvider{}
	for _, pat := range p {
		for _, v := range []string{`{0}`, `{1}`, `{2}`} {
			if !strings.Contains(pat, v) {
				return nil, errors.Errorf("placeholder %q not found in pattern %q", v, pat)
			}
		}

		pat = strings.NewReplacer(`{0}`, `%[2]d`, `{1}`, `%[3]d`, `{2}`, `%[4]d`).Replace(pat)

		result = append(result, &staticMap.TileProvider{
			Name:       fmt.Sprintf("%x", sha256.Sum256([]byte(pat))),
			TileSize:   256, //nolint:mnd
			URLPattern: pat,
		})
	}

	return result, nil
}
