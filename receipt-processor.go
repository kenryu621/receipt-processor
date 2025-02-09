package main

import (
	"encoding/json"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type Receipt struct {
	Retailer     string `json:"retailer"`
	PurchaseDate string `json:"purchaseDate"`
	PurchaseTime string `json:"purchaseTime"`
	Total        string `json:"total"`
	Items        []Item `json:"items"`
}

type Item struct {
	ShortDescription string `json:"shortDescription"`
	Price            string `json:"price"`
}

type ProcessedReceipt struct {
	ID     string
	Points int
}

var receiptStore = make(map[string]ProcessedReceipt)

func calculatePoints(receipt Receipt) int {
	totalPoints := 0
	// One point for every alphanumeric character in the retailer name.
	for _, char := range receipt.Retailer {
		if unicode.IsLetter(char) || unicode.IsDigit(char) {
			totalPoints++
		}
	}

	total, err := strconv.ParseFloat(receipt.Total, 64)
	if err == nil {

		// 50 points if the total is a round dollar amount with no cents.
		if total == math.Trunc(total) {
			totalPoints += 50
		}

		// 25 points if the total is a multiple of 0.25.
		if math.Mod(total, 0.25) == 0 {
			totalPoints += 25
		}
	}

	// 5 points for every two items on the receipt.
	numItems := len(receipt.Items)
	totalPoints += (numItems / 2) * 5

	// If the trimmed length of the item description is a multiple of 3, multiply the price by 0.2 and round up to the nearest integer. The result is the number of points earned.
	for _, item := range receipt.Items {
		description := strings.TrimSpace(item.ShortDescription)
		if len(description)%3 == 0 {
			price, err := strconv.ParseFloat(item.Price, 64)
			if err == nil {
				points := int(math.Ceil(price * 0.2))
				totalPoints += points
			}
		}
	}

	// 6 points if the day in the purchase date is odd.
	purchaseDate, err := time.Parse("2006-01-02", receipt.PurchaseDate)
	if err == nil && purchaseDate.Day()%2 == 1 {
		totalPoints += 6
	}

	// 10 points if the time of purchase is after 2:00pm and before 4:00pm
	purchaseTime, err := time.Parse("15:04", receipt.PurchaseTime)
	if err == nil {
		hour, minute, _ := purchaseTime.Clock()
		totalMinutes := hour*60 + minute
		if totalMinutes >= 14*60 && totalMinutes < 16*60 {
			totalPoints += 10
		}
	}

	return totalPoints
}

func processReceiptHandler(w http.ResponseWriter, r *http.Request) {
	var receipt Receipt
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&receipt); err != nil {
		http.Error(w, "The receipt is invalid.", http.StatusBadRequest)
		return
	}

	points := calculatePoints(receipt)
	id := uuid.New().String()
	receiptStore[id] = ProcessedReceipt{
		ID:     id,
		Points: points,
	}

	response := map[string]string{"id": id}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getPointsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	receipt, exists := receiptStore[id]
	if !exists {
		http.Error(w, "No receipt found for that ID.", http.StatusNotFound)
		return
	}

	response := map[string]int{"points": receipt.Points}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	router := mux.NewRouter()

	router.HandleFunc("/receipts/process", processReceiptHandler).Methods("POST")
	router.HandleFunc("/receipts/{id}/points", getPointsHandler).Methods("GET")

	log.Println("Server is running on port 8087...")
	log.Fatal(http.ListenAndServe(":8087", router))
}
