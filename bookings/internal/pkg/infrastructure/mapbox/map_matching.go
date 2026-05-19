package mapbox

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"

	extdomain "github.com/arngrimur/bilcool_monolith/bookings/pkg/domain"
)

const mapMatchingURL = "https://api.mapbox.com/matching/v5/mapbox/driving"
const maxWaypoints = 100

type Client struct {
	accessToken string
	httpClient  *http.Client
}

func NewClient(accessToken string) *Client {
	return &Client{
		accessToken: accessToken,
		httpClient:  &http.Client{},
	}
}

type matchingResponse struct {
	Code      string `json:"code"`
	Matchings []struct {
		Distance float64 `json:"distance"`
	} `json:"matchings"`
}

// CalculateRoadDistance returns total road distance in meters between the provided GPS points
// using the Mapbox Map Matching API. Falls back to Haversine sum on API failure.
func (c *Client) CalculateRoadDistance(ctx context.Context, points []extdomain.TrackPoint) (int, error) {
	if len(points) < 2 {
		return 0, nil
	}

	totalMeters := 0.0

	// Split into chunks of maxWaypoints, overlapping by 1 point so coverage is complete
	for start := 0; start < len(points)-1; start += maxWaypoints - 1 {
		end := start + maxWaypoints
		if end > len(points) {
			end = len(points)
		}
		chunk := points[start:end]

		meters, err := c.matchSegment(ctx, chunk)
		if err != nil {
			// Fall back to Haversine for this chunk
			meters = haversineChain(chunk)
		}
		totalMeters += meters
	}

	return int(math.Round(totalMeters)), nil
}

func (c *Client) matchSegment(ctx context.Context, points []extdomain.TrackPoint) (float64, error) {
	coords := make([]string, len(points))
	for i, p := range points {
		coords[i] = fmt.Sprintf("%f,%f", p.Lon, p.Lat) // Mapbox expects lon,lat
	}
	coordStr := strings.Join(coords, ";")

	url := fmt.Sprintf("%s/%s?access_token=%s&overview=false&geometries=polyline",
		mapMatchingURL, coordStr, c.accessToken)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("mapbox returned status %d", resp.StatusCode)
	}

	var result matchingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	if result.Code != "Ok" {
		return 0, fmt.Errorf("mapbox code: %s", result.Code)
	}

	total := 0.0
	for _, m := range result.Matchings {
		total += m.Distance
	}
	return total, nil
}

// haversineChain sums straight-line distances between consecutive points (fallback).
func haversineChain(points []extdomain.TrackPoint) float64 {
	total := 0.0
	for i := 1; i < len(points); i++ {
		total += haversineMeters(points[i-1].Lat, points[i-1].Lon, points[i].Lat, points[i].Lon)
	}
	return total
}

func haversineMeters(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371000
	φ1 := lat1 * math.Pi / 180
	φ2 := lat2 * math.Pi / 180
	Δφ := (lat2 - lat1) * math.Pi / 180
	Δλ := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(Δφ/2)*math.Sin(Δφ/2) + math.Cos(φ1)*math.Cos(φ2)*math.Sin(Δλ/2)*math.Sin(Δλ/2)
	return R * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}
