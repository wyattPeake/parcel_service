package main

import (
	"encoding/json"
	"log"
	"net/http"

	"fmt"

	"time"

	"github.com/gorilla/mux"
)

// truck struct
type Truck struct {
	Region   string `json:"region"`
	Capacity int    `json:"capacity"`
}

func newTruck(region string, capacity int) *Truck {
	t := Truck{Region: region, Capacity: capacity}
	return &t
}

var trucks []Truck
var maxCapacity = 10

// creates a slice of trucks in every region
func createTrucks() {

	for i := 0; i < 2; i++ {
		trucks = append(trucks, Truck{"north-east", maxCapacity})
		trucks = append(trucks, Truck{"south-east", maxCapacity})
		trucks = append(trucks, Truck{"south-east", maxCapacity})
	}
}

// finds available trucks in a region
func findTruck(region string) string {
	for i := range trucks {
		if trucks[i].Region == region && trucks[i].Capacity > 0 {
			trucks[i].Capacity--
			return "truck found in " + region
		}
	}
	return "no trucks available"
}

func deliver() {
	for {
		for i := range trucks {
			if trucks[i].Capacity < maxCapacity {
				size := maxCapacity - trucks[i].Capacity
				trucks[i].Capacity += size
				fmt.Println("shipped: " + string(size) + "items, region: " + trucks[i].Region)
			}

		}
		time.Sleep(30 * time.Second)
	}

}

func main() {

	r := mux.NewRouter()
	createTrucks()
	go deliver()

	r.HandleFunc("/shipping", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "apllication/json")
		json.NewEncoder(w).Encode("available")

	})
	r.HandleFunc("/shipping/findtruck/{region}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "apllication/json")

		params := mux.Vars(r)
		response := findTruck(params["region"])
		if response == "no trucks available" {
			w.WriteHeader(503)
		}
		json.NewEncoder(w).Encode(response)

	})
	log.Fatal(http.ListenAndServe("0.0.0.0:8087", r))

}
