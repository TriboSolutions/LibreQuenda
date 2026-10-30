package tenants

import (
	"context"
	"database/sql"
	"errors"
	"librequenda/model"
	"librequenda/shared"
	"librequenda/weather"
	"net/http"
	"time"
)

// WeatherResponse: the forecast of a facility's day, null when there is
// none (disabled, no municipality set, outside the forecast's days, or MeteoGalicia
// unreachable). Source: MeteoGalicia, to be credited wherever it is shown.
type WeatherResponse struct {
	Forecast *weather.DayForecast `json:"forecast"`
	Source   string               `json:"source"`
}

// Weather for a facility on `date` (YYYY-MM-DD in the council's zone; "" = today).
func (s *tenantsService) Weather(ctx context.Context, slug, facility, date string) (*WeatherResponse, *shared.ErrorResponse) {
	var place, err = model.New(s.db).GetFacilityWeatherPlace(ctx, model.GetFacilityWeatherPlaceParams{TenantSlug: slug, FacilitySlug: facility})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.facility_not_found"}
	}
	if err != nil {
		return nil, &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.day_load_failed"}
	}
	var response = &WeatherResponse{Source: "MeteoGalicia"}
	if !place.ShowWeather || !place.MunicipalityCode.Valid {
		return response, nil
	}
	if date == "" {
		var loc, locErr = time.LoadLocation(place.Timezone)
		if locErr != nil {
			loc = time.UTC
		}
		date = time.Now().In(loc).Format(time.DateOnly)
	}
	if _, parseErr := time.Parse(time.DateOnly, date); parseErr != nil {
		return nil, &shared.ErrorResponse{Code: http.StatusBadRequest, Message: "errors.invalid_query_params"}
	}
	if day, ok := weather.Forecast(ctx, place.MunicipalityCode.String, date); ok {
		response.Forecast = &day
	}
	return response, nil
}
