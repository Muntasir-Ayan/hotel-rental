package main

import (
    "database/sql"
    "fmt"
    "log"
    "net/http"
    "io" // Import io package

    _ "github.com/lib/pq"
    "github.com/tidwall/gjson"
)

const (
    dbUser     = "user"
    dbPassword = "password"
    dbName     = "hoteldb"
    apiHost    = "booking-com18.p.rapidapi.com"
    apiKey     = "3308d1f999mshd8adb73826c4e7fp10471fjsn438c09b0aac5"
)

// Location represents the structure of a location
type Location struct {
    CityName string `json:"city_name"`
    Label    string `json:"label"`
    Name     string `json:"name"`
    APIID    string `json:"id"`
}

// Property represents the structure of a property
type Property struct {
    ID            int    `json:"id"`
    Name          string `json:"name"`
    LocationID    int    `json:"location_id"`
    PropertyAPIID int    `json:"property_api_id"`
}

// Function to get data from the API
func getAPIData() ([]Location, error) {
    apiURL := "https://booking-com18.p.rapidapi.com/stays/auto-complete?query=New%20York"
    req, err := http.NewRequest("GET", apiURL, nil)
    if err != nil {
        return nil, err
    }

    req.Header.Add("x-rapidapi-host", apiHost)
    req.Header.Add("x-rapidapi-key", apiKey)

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    // Read the response body into a byte slice
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

    // Parse the JSON response using gjson
    data := gjson.GetBytes(body, "data")
    locations := []Location{}
    for _, item := range data.Array() {
        location := Location{
            CityName: item.Get("city_name").String(),
            Label:    item.Get("label").String(),
            Name:     item.Get("name").String(),
            APIID:    item.Get("id").String(),
        }
        locations = append(locations, location)
    }

    return locations, nil
}

// Function to get property data for a location from the API
func getPropertyData(locationID string) ([]Property, error) {
    apiURL := fmt.Sprintf("https://booking-com18.p.rapidapi.com/stays/search?locationId=%s&checkinDate=2025-01-07&checkoutDate=2025-01-18&units=metric&temperature=c", locationID)
    req, err := http.NewRequest("GET", apiURL, nil)
    if err != nil {
        return nil, err
    }

    req.Header.Add("x-rapidapi-host", apiHost)
    req.Header.Add("x-rapidapi-key", apiKey)

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    // Read the response body into a byte slice
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

    // Parse the JSON response using gjson
    data := gjson.GetBytes(body, "data")
    properties := []Property{}
    for _, item := range data.Array() {
        property := Property{
            Name:          item.Get("name").String(),
            PropertyAPIID: int(item.Get("id").Int()), // Convert int64 to int
        }
        properties = append(properties, property)
    }

    return properties, nil
}

// Function to insert data into PostgreSQL
func insertLocationData(locations []Location) error {
    connStr := fmt.Sprintf("postgres://%s:%s@localhost:5432/%s?sslmode=disable", dbUser, dbPassword, dbName)
    db, err := sql.Open("postgres", connStr)
    if err != nil {
        return err
    }
    defer db.Close()

    for _, loc := range locations {
        // Check if the location already exists
        var locationID int
        err := db.QueryRow(`
            SELECT id FROM locations WHERE api_id = $1`, loc.APIID).Scan(&locationID)

        if err == sql.ErrNoRows {
            // Insert location data if not found
            err = db.QueryRow(`
                INSERT INTO locations (city_name, label, name, api_id) 
                VALUES ($1, $2, $3, $4) 
                RETURNING id`, 
                loc.CityName, loc.Label, loc.Name, loc.APIID).Scan(&locationID)
            if err != nil {
                log.Printf("Error inserting location: %s", err)
                continue
            }
        } else if err != nil {
            log.Printf("Error checking location: %s", err)
            continue
        }

        // Fetch property data for the inserted location using the api_id as location_id
        properties, err := getPropertyData(loc.APIID)
        if err != nil {
            log.Printf("Error fetching properties for location %s: %s", loc.CityName, err)
            continue
        }

        // Insert property data into property_table
        for _, prop := range properties {
            _, err := db.Exec(`
                INSERT INTO property_table (name, location_id, property_api_id) 
                VALUES ($1, $2, $3) 
                ON CONFLICT (property_api_id) DO NOTHING`,
                prop.Name, locationID, prop.PropertyAPIID)
            if err != nil {
                log.Printf("Error inserting property: %s", err)
                continue
            }
        }
    }
    return nil
}

func main() {
    locations, err := getAPIData()
    if err != nil {
        log.Fatal(err)
    }

    if err := insertLocationData(locations); err != nil {
        log.Fatal(err)
    }

    fmt.Println("Data inserted successfully.")
}
