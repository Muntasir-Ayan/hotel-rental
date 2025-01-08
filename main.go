package main

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "strconv"
    "io"

    _ "github.com/lib/pq"
)

const (
    dbUser     = "user"
    dbPassword = "password"
    dbName     = "hoteldb"
    apiURL     = "https://booking-com18.p.rapidapi.com/web/stays/auto-complete?query=New%20York"
    apiHost    = "booking-com18.p.rapidapi.com"
    apiKey     = "3308d1f999mshd8adb73826c4e7fp10471fjsn438c09b0aac5"
)

type Location struct {
    DestID   string `json:"dest_id"`
    Value    string `json:"value"`
    DestType string `json:"dest_type"`
}

type Hotel struct {
    HotelID   string `json:"hotel_id"`
    HotelName string `json:"hotel_name"`
    DestID    string `json:"dest_id"`
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

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

    var response map[string]interface{}
    if err := json.Unmarshal(body, &response); err != nil {
        return nil, err
    }

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

// Function to get hotel data from the API
func getHotelData(destID, destType string) ([]Hotel, error) {
    url := fmt.Sprintf("https://booking-com18.p.rapidapi.com/web/stays/search?destId=%s&destType=%s&checkIn=2025-01-12&checkOut=2025-01-31", destID, destType)
    req, err := http.NewRequest("GET", url, nil)
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

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

    var response map[string]interface{}
    if err := json.Unmarshal(body, &response); err != nil {
        return nil, err
    }

    data, ok := response["data"].(map[string]interface{})
    if !ok {
        return nil, fmt.Errorf("unexpected type for data field")
    }

    results, ok := data["results"].([]interface{})
    if !ok {
        return nil, fmt.Errorf("unexpected type for results field")
    }

    hotels := []Hotel{}
    for _, item := range results {
        itemMap, ok := item.(map[string]interface{})
        if !ok {
            continue
        }

        basicPropertyData, ok := itemMap["basicPropertyData"].(map[string]interface{})
        if !ok {
            continue
        }

        hotelID, ok := basicPropertyData["id"]
        if !ok {
            continue
        }

        displayName, ok := itemMap["displayName"].(map[string]interface{})
        if !ok {
            continue
        }

        hotelName, ok := displayName["text"]
        if !ok {
            continue
        }

        // Handle float64 case for hotelID
        var hotelIDStr string
        switch v := hotelID.(type) {
        case float64:
            hotelIDStr = strconv.FormatFloat(v, 'f', -1, 64)
        case string:
            hotelIDStr = v
        default:
            return nil, fmt.Errorf("unexpected type for hotelID: %T", v)
        }

        // Handle float64 case for hotelName
        var hotelNameStr string
        switch v := hotelName.(type) {
        case float64:
            hotelNameStr = strconv.FormatFloat(v, 'f', -1, 64)
        case string:
            hotelNameStr = v
        default:
            return nil, fmt.Errorf("unexpected type for hotelName: %T", v)
        }

        hotel := Hotel{
            HotelID:   hotelIDStr,
            HotelName: hotelNameStr,
            DestID:    destID,
        }
        hotels = append(hotels, hotel)
    }

    return hotels, nil
}

// Function to insert hotel data into PostgreSQL
func insertHotelData(hotels []Hotel) error {
    connStr := fmt.Sprintf("postgres://%s:%s@localhost:5432/%s?sslmode=disable", dbUser, dbPassword, dbName)
    db, err := sql.Open("postgres", connStr)
    if err != nil {
        return err
    }
    defer db.Close()

    for _, hotel := range hotels {
        _, err := db.Exec(`
            INSERT INTO associate_hotel (hotel_id, hotel_name, dest_id) 
            VALUES ($1, $2, $3) 
            ON CONFLICT (hotel_id) DO NOTHING`,
            hotel.HotelID, hotel.HotelName, hotel.DestID)
        if err != nil {
            log.Printf("Error inserting hotel: %s", err)
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

    // Fetch and insert hotel data for each location
    for _, loc := range locations {
        hotels, err := getHotelData(loc.DestID, loc.DestType)
        if err != nil {
            log.Printf("Error fetching hotel data for dest_id %s: %s", loc.DestID, err)
            continue
        }

        if err := insertHotelData(hotels); err != nil {
            log.Printf("Error inserting hotel data for dest_id %s: %s", loc.DestID, err)
            continue
        }
    }

    fmt.Println("Data inserted successfully.")
}