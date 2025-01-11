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
    "github.com/lib/pq"
)

const (
    dbUser     = "user"
    dbPassword = "password"
    dbName     = "hoteldb"
    apiURL     = "https://booking-com18.p.rapidapi.com/web/stays/auto-complete?query=New%20York"
    apiHost    = "booking-com18.p.rapidapi.com"
    apiKey     = "3dab48e211msh05065cf89ab516dp101291jsnc31c829f1dc9"
)

type Location struct {
    DestID   string `json:"dest_id"`
    Value    string `json:"value"`
    DestType string `json:"dest_type"`
}

type Hotel struct {
    HotelID     string  `json:"hotel_id"`
    HotelName   string  `json:"hotel_name"`
    DestID      string  `json:"dest_id"`
    HotelIDUrl  string  `json:"hotel_id_url"`
    Rating      float64 `json:"rating"`
    ReviewCount int     `json:"review_count"`
    Price       string  `json:"price"`
    Bedrooms    int     `json:"bedrooms"`
    Bathroom    int     `json:"bathroom"`
    Location    string  `json:"location"`
}

type PropertyDetail struct {
    HotelID     string   `json:"hotel_id"`
    Description string   `json:"description"`
    ImageURL    []string `json:"image_url"`
    Type        string   `json:"type"`
    Amenities   []string `json:"amenities"`
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

    // Log the response body for debugging
    log.Printf("API Response: %s", body)

    var response map[string]interface{}
    if err := json.Unmarshal(body, &response); err != nil {
        return nil, err
    }

    // Check if the 'data' field exists and is of the expected type
    data, ok := response["data"].([]interface{})
    if !ok {
        return nil, fmt.Errorf("unexpected type for data field or data field is nil")
    }

    locations := []Location{}
    for _, item := range data {
        itemMap, ok := item.(map[string]interface{})
        if !ok {
            continue
        }

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
            log.Printf("Error inserting location %s: %s", loc.DestID, err)
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

    // Log the response body for debugging
    log.Printf("API Response: %s", body)

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

        hotelIDUrl, ok := itemMap["id"].(string) // Assuming 'id' contains the hotel ID URL
        if !ok {
            return nil, fmt.Errorf("unable to extract hotel_id_url")
        }

        // Extract rating
        reviews, ok := basicPropertyData["reviews"].(map[string]interface{})
        if !ok {
            continue
        }

        rating, ok := reviews["totalScore"].(float64)
        if !ok {
            rating = 0 // Default value if not found
        }

        reviewCount, ok := reviews["reviewsCount"].(float64)
        if !ok {
            reviewCount = 0 // Default value if not found
        }

        // Extract price
        priceDisplayInfo, ok := itemMap["priceDisplayInfoIrene"].(map[string]interface{})
        if !ok {
            continue
        }

        displayPrice, ok := priceDisplayInfo["displayPrice"].(map[string]interface{})
        if !ok {
            continue
        }

        amountPerStay, ok := displayPrice["amountPerStay"].(map[string]interface{})
        if !ok {
            continue
        }

        price, ok := amountPerStay["amount"].(string)
        if !ok {
            price = "0" // Default value if not found
        }

        // Extract bedrooms
        matchingUnitConfigurations, ok := itemMap["matchingUnitConfigurations"].(map[string]interface{})
        if !ok {
            continue
        }

        commonConfiguration, ok := matchingUnitConfigurations["commonConfiguration"].(map[string]interface{})
        if !ok {
            continue
        }

        nbAllBeds, ok := commonConfiguration["nbAllBeds"].(float64)
        if !ok {
            nbAllBeds = 0 // Default value if not found
        }

        nbBathrooms, ok := commonConfiguration["nbBathrooms"].(float64)
        if !ok {
            nbBathrooms = 0 // Default value if not found
        }

        // Extract location
        location, ok := itemMap["location"].(map[string]interface{})
        if !ok {
            continue
        }

        displayLocation, ok := location["displayLocation"].(string)
        if !ok {
            displayLocation = "" // Default value if not found
        }

        hotel := Hotel{
            HotelID:     hotelIDStr,
            HotelName:   hotelNameStr,
            DestID:      destID,
            HotelIDUrl:  hotelIDUrl,
            Rating:      rating,
            ReviewCount: int(reviewCount),
            Price:       price,
            Bedrooms:    int(nbAllBeds), // Convert float64 to int
            Bathroom:    int(nbBathrooms), // Convert float64 to int
            Location:    displayLocation,
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
        _, err := db.Exec(
            `INSERT INTO associate_hotel (hotel_id, hotel_name, dest_id, hotel_id_url, rating, review_count, price, bedrooms, bathroom, location) 
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) 
            ON CONFLICT (hotel_id) DO NOTHING`,
            hotel.HotelID, hotel.HotelName, hotel.DestID, hotel.HotelIDUrl, hotel.Rating, hotel.ReviewCount, hotel.Price,
            hotel.Bedrooms, hotel.Bathroom, hotel.Location)
        if err != nil {
            log.Printf("Error inserting hotel: %s", err)
            continue
        }
    }
    return nil
}

// Function to get property description from the API
func getPropertyDescription(hotelID string) (string, error) {
    url := fmt.Sprintf("https://booking-com18.p.rapidapi.com/stays/get-description?hotelId=%s", hotelID)
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return "", err
    }

    req.Header.Add("x-rapidapi-host", apiHost)
    req.Header.Add("x-rapidapi-key", apiKey)

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", err
    }

    // Log the response body for debugging
    log.Printf("API Response: %s", body)

    var response map[string]interface{}
    if err := json.Unmarshal(body, &response); err != nil {
        return "", err
    }

    data, ok := response["data"].([]interface{})
    if !ok {
        return "", fmt.Errorf("unexpected type for data field")
    }

    if len(data) == 0 {
        return "", fmt.Errorf("no description found for hotel ID: %s", hotelID)
    }

    itemMap, ok := data[0].(map[string]interface{})
    if !ok {
        return "", fmt.Errorf("unexpected type for item in data array")
    }

    description, ok := itemMap["description"].(string)
    if !ok {
        return "", fmt.Errorf("description not found for hotel ID: %s", hotelID)
    }

    return description, nil
}

// Function to get detailed property information from the API
func getPropertyDetails(hotelID string) (PropertyDetail, error) {
    url := fmt.Sprintf("https://booking-com18.p.rapidapi.com/stays/detail?hotelId=%s&checkinDate=2025-01-11&checkoutDate=2025-01-23&units=metric", hotelID)
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return PropertyDetail{}, err
    }

    req.Header.Add("x-rapidapi-host", apiHost)
    req.Header.Add("x-rapidapi-key", apiKey)

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return PropertyDetail{}, err
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return PropertyDetail{}, err
    }

    // Log the response body for debugging
    log.Printf("API Response: %s", body)

    var response map[string]interface{}
    if err := json.Unmarshal(body, &response); err != nil {
        return PropertyDetail{}, err
    }

    data, ok := response["data"].(map[string]interface{})
    if !ok {
        return PropertyDetail{}, fmt.Errorf("unexpected type for data field")
    }

    // Extract images
    rooms, ok := data["rooms"].(map[string]interface{})
    if !ok {
        return PropertyDetail{}, fmt.Errorf("unexpected type for rooms field")
    }

    images := []string{}
    for _, room := range rooms {
        roomDetails, ok := room.(map[string]interface{})
        if !ok {
            continue
        }

        photos, ok := roomDetails["photos"].([]interface{})
        if !ok {
            continue
        }

        for _, photo := range photos {
            photoDetails, ok := photo.(map[string]interface{})
            if !ok {
                continue
            }

            urlOriginal, ok := photoDetails["url_original"].(string)
            if ok {
                images = append(images, urlOriginal)
            }
        }
    }

    // Extract type
    accommodationTypeName, ok := data["accommodation_type_name"].(string)
    if !ok {
        accommodationTypeName = ""
    }

    // Extract amenities
    facilitiesBlock, ok := data["facilities_block"].(map[string]interface{})
    if !ok {
        return PropertyDetail{}, fmt.Errorf("unexpected type for facilities_block field")
    }

    facilities, ok := facilitiesBlock["facilities"].([]interface{})
    if !ok {
        return PropertyDetail{}, fmt.Errorf("unexpected type for facilities field")
    }

    amenities := []string{}
    for i, facility := range facilities {
        if i >= 3 {
            break
        }
        facilityDetails, ok := facility.(map[string]interface{})
        if !ok {
            continue
        }

        name, ok := facilityDetails["name"].(string)
        if ok {
            amenities = append(amenities, name)
        }
    }

    return PropertyDetail{
        HotelID:   hotelID,
        ImageURL:  images,
        Type:      accommodationTypeName,
        Amenities: amenities,
    }, nil
}

// Function to insert property detail data into PostgreSQL
func insertPropertyDetailData(propertyDetails []PropertyDetail) error {
    connStr := fmt.Sprintf("postgres://%s:%s@localhost:5432/%s?sslmode=disable", dbUser, dbPassword, dbName)
    db, err := sql.Open("postgres", connStr)
    if err != nil {
        return err
    }
    defer db.Close()

    for _, detail := range propertyDetails {
        _, err := db.Exec(`
            INSERT INTO property_detail (hotel_id, description, image_url, type, amenities) 
            VALUES ($1, $2, $3, $4, $5) 
            ON CONFLICT (hotel_id) DO NOTHING`,
            detail.HotelID, detail.Description, pq.Array(detail.ImageURL), detail.Type, pq.Array(detail.Amenities))
        if err != nil {
            log.Printf("Error inserting property detail for hotel_id %s: %s", detail.HotelID, err)
            continue
        }
    }
    return nil
}

func main() {
    locations, err := getAPIData()
    if err != nil {
        log.Fatalf("Error fetching locations: %s", err)
    }

    if err := insertLocationData(locations); err != nil {
        log.Fatalf("Error inserting locations: %s", err)
    }

    // Fetch and insert hotel data for each location
    for _, loc := range locations {
        fmt.Printf("Fetching hotels for destination ID: %s\n", loc.DestID)
        hotels, err := getHotelData(loc.DestID, loc.DestType)
        if err != nil {
            log.Printf("Error fetching hotel data for dest_id %s: %s", loc.DestID, err)
            continue
        }

        if err := insertHotelData(hotels); err != nil {
            log.Printf("Error inserting hotel data for dest_id %s: %s", loc.DestID, err)
            continue
        }

        // Fetch and insert property details for each hotel
        propertyDetails := []PropertyDetail{}
        for _, hotel := range hotels {
            fmt.Printf("Fetching property details for hotel ID: %s\n", hotel.HotelID)
            description, err := getPropertyDescription(hotel.HotelID)
            if err != nil {
                log.Printf("Error fetching property description for hotel_id %s: %s", hotel.HotelID, err)
                continue
            }

            detail, err := getPropertyDetails(hotel.HotelID)
            if err != nil {
                log.Printf("Error fetching property details for hotel_id %s: %s", hotel.HotelID, err)
                continue
            }

            detail.Description = description
            propertyDetails = append(propertyDetails, detail)
        }

        if err := insertPropertyDetailData(propertyDetails); err != nil {
            log.Printf("Error inserting property detail data: %s", err)
            continue
        }
    }

    fmt.Println("Data inserted successfully.")
}