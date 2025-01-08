package main

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "io" // Import io package

    _ "github.com/lib/pq"
)

const (
    dbUser     = "user"
    dbPassword = "password"
    dbName     = "hoteldb"  // Updated database name
    apiURL     = "https://booking-com18.p.rapidapi.com/web/stays/auto-complete?query=Bangladesh"
    apiHost    = "booking-com18.p.rapidapi.com"
    apiKey     = "3308d1f999mshd8adb73826c4e7fp10471fjsn438c09b0aac5"
)

// Location represents the structure of a location
type Location struct {
    DestID   string `json:"dest_id"`
    Value    string `json:"value"`
    DestType string `json:"dest_type"`
}

// Function to get data from the API
func getAPIData() ([]Location, error) {
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

    // Parse the JSON response
    var response map[string]interface{}
    if err := json.Unmarshal(body, &response); err != nil {
        return nil, err
    }

    // Extract location data
    data := response["data"].([]interface{})
    locations := []Location{}
    for _, item := range data {
        itemMap := item.(map[string]interface{})
        location := Location{
            DestID:   itemMap["dest_id"].(string),
            Value:    itemMap["value"].(string),
            DestType: itemMap["dest_type"].(string),
        }
        locations = append(locations, location)
    }

    return locations, nil
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
        _, err := db.Exec(`
            INSERT INTO locations (dest_id, value, dest_type) 
            VALUES ($1, $2, $3) 
            ON CONFLICT (dest_id) DO NOTHING`,
            loc.DestID, loc.Value, loc.DestType)
        if err != nil {
            log.Printf("Error inserting location: %s", err)
            continue
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