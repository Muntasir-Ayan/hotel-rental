# Hotel Data Fetcher and Inserter

This project is a Go application that fetches hotel data from a third-party API and inserts it into a PostgreSQL database. The application retrieves location data, hotel data, and detailed property information, then stores this information in the database.

## Table of Contents

- [Installation](#installation)
- [Usage](#usage)
- [Configuration](#configuration)
- [Database Schema](#database-schema)
- [API Information](#api-information)
- [Contributing](#contributing)
- [License](#license)

## Installation

1. Clone the repository:
    ```sh
    git clone https://github.com/Muntasir-Ayan/hotel-rental.git
    cd hotel-rental
    ```

2. Install the required dependencies:
    ```sh
    go get -u github.com/lib/pq
    ```

3. Set up your PostgreSQL database and create the necessary tables (see [Database Schema](#database-schema)).

## Usage

1. Update the configuration constants in the `main.go` file:
    ```go
    const (
        dbUser     = "your_db_user"
        dbPassword = "your_db_password"
        dbName     = "your_db_name"
        apiURL     = "https://booking-com18.p.rapidapi.com/web/stays/auto-complete?query=New%20York"
        apiHost    = "booking-com18.p.rapidapi.com"
        apiKey     = "your_api_key"
    )
    ```

2. Run the Docker:
    ```sh
       docker-compose up
    ```
3. Create the Database:
    ```sh
    docker exec -it postgres-container psql -U user hoteldb
    ```
## Database Schema (Create the following table)

```sql
CREATE TABLE locations (
    dest_id TEXT PRIMARY KEY,
    value TEXT,
    dest_type TEXT
);

CREATE TABLE associate_hotel (
    hotel_id VARCHAR(255) PRIMARY KEY,
    hotel_name VARCHAR(255) NOT NULL,
    dest_id VARCHAR(255) NOT NULL,
    hotel_id_url VARCHAR(255) NOT NULL,
    rating FLOAT NOT NULL,
    review_count INT NOT NULL,
    price VARCHAR(255) NOT NULL,
    bedrooms INT NOT NULL,
    bathroom INT NOT NULL,
    location VARCHAR(255) NOT NULL,
    amenities1 VARCHAR(255),
    amenities2 VARCHAR(255),
    amenities3 VARCHAR(255),
    guest_count INT NOT NULL,
    FOREIGN KEY (dest_id) REFERENCES locations(dest_id)
);

CREATE TABLE property_detail (
    hotel_id VARCHAR(255) PRIMARY KEY,
    description TEXT NOT NULL,
    image_url TEXT[],
    type VARCHAR(255) NOT NULL,
    amenities TEXT[],
    FOREIGN KEY (hotel_id) REFERENCES associate_hotel(hotel_id)
);
```
## Run 
```bash
   go run main.go
````

## Configuration

The configuration constants in the `main.go` file need to be updated with your PostgreSQL database credentials and RapidAPI key. These constants include:

- `dbUser`: Your PostgreSQL username.
- `dbPassword`: Your PostgreSQL password.
- `dbName`: Your PostgreSQL database name.
- `apiURL`: The URL for the API endpoint to fetch location data.
- `apiHost`: The `x-rapidapi-host` header value.
- `apiKey`: Your RapidAPI key.



