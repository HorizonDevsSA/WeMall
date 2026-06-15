package routing

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

const CouriersGeoKey = "active_couriers"

type GeoIndex struct {
	rdb *redis.Client
}

func NewGeoIndex(rdb *redis.Client) *GeoIndex {
	return &GeoIndex{rdb: rdb}
}

func (g *GeoIndex) UpdateCourierLocation(ctx context.Context, courierID string, lat, lon float64) error {
	_, err := g.rdb.GeoAdd(ctx, CouriersGeoKey, &redis.GeoLocation{
		Longitude: lon,
		Latitude:  lat,
		Name:      courierID,
	}).Result()
	if err != nil {
		return fmt.Errorf("redis geoadd: %w", err)
	}
	return nil
}

func (g *GeoIndex) RemoveCourier(ctx context.Context, courierID string) error {
	_, err := g.rdb.ZRem(ctx, CouriersGeoKey, courierID).Result()
	if err != nil {
		return fmt.Errorf("redis zrem: %w", err)
	}
	return nil
}

func (g *GeoIndex) GetNearbyCouriers(ctx context.Context, lat, lon float64, radiusKm float64) ([]string, error) {
	locations, err := g.rdb.GeoSearch(ctx, CouriersGeoKey, &redis.GeoSearchQuery{
		Longitude:  lon,
		Latitude:   lat,
		Radius:     radiusKm,
		RadiusUnit: "km",
		Sort:       "ASC",
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("redis geosearch: %w", err)
	}

	courierIDs := make([]string, len(locations))
	for i, loc := range locations {
		courierIDs[i] = loc
	}
	return courierIDs, nil
}
