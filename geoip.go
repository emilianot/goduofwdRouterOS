/*
Based (copied) on code from the goFwd project
Credits to jftuga (John Taylor) for the original code.
https://github.com/jftuga/gofwd
*/
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// This is the format returned by: https://ipinfo.io/w.x.y.z/json
type ipInfoResult struct {
	IP       string
	Hostname string
	City     string
	Region   string
	Country  string
	Loc      string
	Lon      float64
	Lat      float64
	Postal   string
	Org      string
	Timezone string
	Distance float64
	Updated  int64
	ErrMsg   error
}

var localGeoIP ipInfoResult
var geoCache = make(map[string]ipInfoResult)

func IPisInGeoList(ip string, geos []TGeo) bool {
	if len(geos) == 0 {
		return true
	}

	remoteGeoIP, err := getIPInfo(ip)
	if err != nil {
		fmt.Printf("ERROR: Failed to get remote IP info: %s\n", err.Error())
		return false
	}

	fmt.Printf("%+v", remoteGeoIP)

	for _, geo := range geos {
		if validateGeoIP(remoteGeoIP, geo) {
			return true
		}
	}

	return false
}

/*
getIpInfo issues a web query to ipinfo.io
The JSON result is converted to an ipInfoResult struct
Args:

	ip: an IPv4 address

Returns:

	an ipInfoResult struct containing the information returned by the service
*/
func getIPInfo(ip string) (ipInfoResult, error) {
	var obj ipInfoResult

	if len(ip) == 0 {
		if localGeoIP.Updated > (time.Now().Unix() - config.IPInfo.CacheTime) {
			return localGeoIP, nil
		}
	} else {
		if geoCache[ip].Updated > (time.Now().Unix() - config.IPInfo.CacheTime) {
			return geoCache[ip], nil
		}
	}

	url := "https://ipinfo.io/" + iif(len(ip) == 0, "json", ip+"/json").(string) + iif(len(config.IPInfo.Token) > 0, "?token="+config.IPInfo.Token, "").(string)
	// #nosec G107 -- ip has been validated
	resp, err := http.Get(url)
	if err != nil {
		return obj, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return obj, err
	}

	if strings.Contains(string(body), "Rate limit exceeded") {
		err := fmt.Errorf("IPInfo Rate limit exceeded")
		empty := ipInfoResult{}
		return empty, err
	}

	err = json.Unmarshal(body, &obj)
	if err != nil {
		empty := ipInfoResult{}
		return empty, err
	}

	// save to to cache
	obj.Lat, obj.Lon = latlon2coord(obj.Loc)

	if len(ip) == 0 && obj.Lon == 0 && obj.Lat == 0 {
		err := fmt.Errorf("failed to get local IP info")
		return obj, err
	}

	obj.Updated = time.Now().Unix()
	if len(ip) == 0 {
		localGeoIP = obj
	} else if config.IPInfo.CacheMaxCount > 0 {
		geoCache[ip] = obj
		if len(geoCache) > config.IPInfo.CacheMaxCount {
			// remove the oldest entry
			oldest := time.Now().Unix()
			var oldestKey string
			for k, v := range geoCache {
				if v.Updated < oldest {
					oldest = v.Updated
					oldestKey = k
				}
			}
			delete(geoCache, oldestKey)
		}
	}

	return obj, nil
}

func validateGeoIP(remoteGeoIP ipInfoResult, data TGeo) bool {
	if data.Country != "" && !(data.Country == strings.ToUpper(remoteGeoIP.Country) || (data.Country[0:1] == "*" && strings.Contains(strings.ToUpper(remoteGeoIP.Country), data.Country[1:]))) {
		return false
	}
	if data.Region != "" && !(data.Region == strings.ToUpper(remoteGeoIP.Region) || (data.Region[0:1] == "*" && strings.Contains(strings.ToUpper(remoteGeoIP.Region), data.Region[1:]))) {
		return false
	}
	if data.City != "" && !(data.City == strings.ToUpper(remoteGeoIP.City) || (data.City[0:1] == "*" && strings.Contains(strings.ToUpper(remoteGeoIP.City), data.City[1:]))) {
		return false
	}
	if data.Postal != "" && !(data.Postal == strings.ToUpper(remoteGeoIP.Postal) || (data.Postal[0:1] == "*" && strings.Contains(strings.ToUpper(remoteGeoIP.Postal), data.Postal[1:]))) {
		return false
	}
	if data.Org != "" && !(data.Org == strings.ToUpper(remoteGeoIP.Org) || (data.Org[0:1] == "*" && strings.Contains(strings.ToUpper(remoteGeoIP.Org), data.Org[1:]))) {
		return false
	}
	if data.Hostname != "" && !(data.Hostname == strings.ToUpper(remoteGeoIP.Hostname) || (data.Hostname[0:1] == "*" && strings.Contains(strings.ToUpper(remoteGeoIP.Hostname), data.Hostname[1:]))) {
		return false
	}
	if data.Distance > 0 && !validateLocation(remoteGeoIP, data.Distance, data.Lat, data.Lon) {
		return false
	}
	return true
}

func validateLocation(remoteGeoIP ipInfoResult, distance float64, lat float64, lon float64) bool {
	if remoteGeoIP.Lat == 0 || remoteGeoIP.Lon == 0 {
		fmt.Println("ERROR: Remote IP info does not have lat,lon")
		return false
	}
	if lat == 0 && lon == 0 {
		if meLat == 0 && meLon == 0 {
			var errLocalIP error
			localGeoIP, errLocalIP = getIPInfo("")
			if errLocalIP != nil {
				fmt.Printf("ERROR: Failed to get local IP info: %s\n", errLocalIP.Error())
				return false
			}
			meLat = localGeoIP.Lat
			meLon = localGeoIP.Lon
		}
		lat = meLat
		lon = meLon
	}

	calculatedDistance := HaversineDistance(lat, lon, remoteGeoIP.Lat, remoteGeoIP.Lon)
	if calculatedDistance > distance {
		fmt.Printf("ERROR: Remote IP %s distance (%f) exceeds the maximum allowed (%f)\n", remoteGeoIP.IP, calculatedDistance, distance)
		return false
	}

	return true
}

func latlon2coord(latlon string) (float64, float64) {
	if len(latlon) == 0 || strings.Contains(latlon, ",") == false {
		return 0, 0
	}
	fmt.Println("latlon2coord: ", latlon)
	slots := strings.Split(latlon, ",")
	lat, _ := strconv.ParseFloat(slots[0], 64)
	lon, _ := strconv.ParseFloat(slots[1], 64)
	return lat, lon
}

// adapted from: https://gist.github.com/cdipaolo/d3f8db3848278b49db68
// haversin(θ) function
func hsin(theta float64) float64 {
	return math.Pow(math.Sin(theta/2), 2)
}

// HaversineDistanceKM returns the distance (in KM) between two points of
//
//	a given longitude and latitude relatively accurately (using a spherical
//	approximation of the Earth) through the Haversin Distance Formula for
//	great arc distance on a sphere with accuracy for small distances
//
// point coordinates are supplied in degrees and converted into rad. in the func
//
// http://en.wikipedia.org/wiki/Haversine_formula
func HaversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	// convert to radians
	// must cast radius as float to multiply later
	var la1, lo1, la2, lo2, r float64

	piRad := math.Pi / 180
	la1 = lat1 * piRad
	lo1 = lon1 * piRad
	la2 = lat2 * piRad
	lo2 = lon2 * piRad

	r = 6378100 // Earth radius in METERS

	// calculate
	h := hsin(la2-la1) + math.Cos(la1)*math.Cos(la2)*hsin(lo2-lo1)

	meters := 2 * r * math.Asin(math.Sqrt(h))
	kilometers := meters / 1000
	return kilometers
}
