package handlers

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"receipt-splitter-backend/auth"
	"receipt-splitter-backend/db"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	openai "github.com/sashabaranov/go-openai"
)

var openaiClient *openai.Client

// Initialize OpenAI client
func InitOpenAIClient(apiKey string) {
	openaiClient = openai.NewClient(apiKey)
}

// ParseReceiptHandler processes and parses receipts
func ParseReceiptHandler(w http.ResponseWriter, r *http.Request) {
	type ParseRequest struct {
		Receipt string `json:"receipt"`
	}

	var req ParseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	base64Image := req.Receipt
	if strings.HasPrefix(base64Image, "data:image") {
		base64Image = strings.Split(base64Image, ",")[1]
	}

	// Decode Base64 image
	_, err := base64.StdEncoding.DecodeString(base64Image)
	if err != nil {
		http.Error(w, "Invalid Base64 image data", http.StatusBadRequest)
		return
	}

	// Simulated text extraction from Google Vision API (replace with actual call)
	extractedText, err := callGoogleVisionAPI(base64Image)
	if err != nil {
		http.Error(w, "Error parsing b64 data", http.StatusBadRequest)
		return
	}

	// Call OpenAI for parsing
	structuredData, err := callOpenAIForParsing(extractedText)
	if err != nil {
		http.Error(w, "Failed to parse receipt with OpenAI API: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(structuredData)
}

// Call OpenAI to parse extracted text into structured JSON
func callOpenAIForParsing(extractedText string) (map[string]interface{}, error) {
	if openaiClient == nil {
		return nil, errors.New("OpenAI client not initialized")
	}

	resp, err := openaiClient.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT4oLatest, // or GPT-4, if available
			Messages: []openai.ChatCompletionMessage{
				{
					Role: openai.ChatMessageRoleSystem,
					Content: `
You are a highly intelligent receipt parsing assistant. Your task is to analyze the provided receipt text and return a structured JSON object with the following format:
          {
            "name": "Store Name",
            "modifiers": [
              {"type": "Modifier Type", "value": Value, "percentage": PercentageOfOrder (if applicable)}
            ],
            "items": [
              {"item": "Item Name", "price": PricePerItem, "qty": Quantity}
            ]
          }
          Important Considerations:
          Store Name:
          Extract the store's name from the receipt header or footer, wherever applicable.
          Modifiers:
          Include all price-related adjustments as separate entries in the modifiers array. Each modifier should include:
          type: The name of the modifier (e.g., "Service Charge", "Discount").
          value: The absolute value of the modifier (e.g., £10.00 for a discount or service charge).
          percentage: If the modifier is a percentage of the total order, include the percentage. If not, set this field to null.
          Items:
          Each item should include:
          item: The item's name, accurately extracted even if split across multiple lines.
          price: The price per unit of the item. If the price is for multiple units, divide the total price by the quantity to calculate the per-item price. This should not include the currency, just the value.
          qty: The quantity of the item. Ensure the correct quantity, even if quantities are specified on separate lines or implied by additional notes like "x2" or "double."
          Handle cases where:
          The price is listed per line (inclusive or exclusive of totals).
          Adjustments (e.g., additions, subtractions, or discounts) are listed on sublines or as notes.
          Format Adaptation:
          Some receipts might have irregular formats, such as handwritten-style totals, unclear item groupings, or totals including service charges. Adapt accordingly and infer missing information where possible.
          Tax:
          If tax is explicitly mentioned, include it as a modifier in the modifiers array with type: "Tax". Specify the tax value and its percentage of the total (if applicable).
          Error Handling:
          If any field cannot be confidently extracted, provide a null value for that field in the JSON and note the reason in a separate "notes" field.
`,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: fmt.Sprintf("Here is the extracted text from a receipt, ONLY PROVIDE ME THE JSON OBJECT NOTHING ELSE:\n\n %s", extractedText),
				},
			},
		},
	)

	if err != nil {
		return nil, err
	}

	if len(resp.Choices) == 0 {
		return nil, errors.New("no response from OpenAI")
	}

	// Extract the content of the response
	rawContent := resp.Choices[0].Message.Content
	fmt.Println("Raw OpenAI response:", rawContent) // Debugging

	// Clean up the response to extract the JSON
	cleanedContent := strings.TrimSpace(rawContent)
	cleanedContent = strings.TrimPrefix(cleanedContent, "```json")
	cleanedContent = strings.TrimPrefix(cleanedContent, "```")
	cleanedContent = strings.TrimSuffix(cleanedContent, "```")

	// Parse the cleaned JSON content
	var structuredData map[string]interface{}
	err = json.Unmarshal([]byte(cleanedContent), &structuredData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OpenAI response: %v", err)
	}

	return structuredData, nil
}

// Helper functions for API calls
func callGoogleVisionAPI(base64Image string) (string, error) {
	apiKey := os.Getenv("GOOGLE_API_KEY")
	url := "https://vision.googleapis.com/v1/images:annotate?key=" + apiKey

	requestBody := map[string]interface{}{
		"requests": []map[string]interface{}{
			{
				"image": map[string]string{"content": base64Image},
				"features": []map[string]interface{}{
					{"type": "TEXT_DETECTION", "maxResults": 1},
				},
			},
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(url, "application/json", strings.NewReader(string(jsonBody)))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return "", errors.New("Google Vision API error: " + string(body))
	}

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return "", err
	}

	responses, ok := result["responses"].([]interface{})
	if !ok || len(responses) == 0 {
		return "", errors.New("invalid Google Vision API response")
	}

	fullText, _ := responses[0].(map[string]interface{})["fullTextAnnotation"].(map[string]interface{})["text"].(string)
	return fullText, nil
}

func CreateReceiptHandler(w http.ResponseWriter, r *http.Request) {
	// Get the user ID from the context
	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Decode request body into a generic receipt object
	var receipt map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&receipt); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	// Generate a new receipt ID and add a created_at timestamp
	receiptID := uuid.New().String()
	createdAt := time.Now()

	// Store the receipt in the database
	query := `INSERT INTO receipts (id, user_id, receipt_data, created_at) VALUES ($1, $2, $3, $4)`
	receiptJSON, err := json.Marshal(receipt)
	if err != nil {
		http.Error(w, "Failed to serialize receipt", http.StatusInternalServerError)
		return
	}

	_, err = db.DB.Exec(query, receiptID, userID, receiptJSON, createdAt)
	if err != nil {
		http.Error(w, "Failed to store receipt", http.StatusInternalServerError)
		return
	}

	// Prepare the response
	response := map[string]interface{}{
		"id":         receiptID,
		"user_id":    userID,
		"receipt":    receipt,
		"created_at": createdAt,
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func GetAllReceiptsHandler(w http.ResponseWriter, r *http.Request) {
	// Get user ID from the context
	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Query all receipts for the user
	rows, err := db.DB.Query(`SELECT id, receipt_data, created_at FROM receipts WHERE user_id = $1`, userID)
	if err != nil {
		http.Error(w, "Failed to fetch receipts", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var receipts []map[string]interface{}
	for rows.Next() {
		var id string
		var receiptData []byte
		var createdAt time.Time

		err := rows.Scan(&id, &receiptData, &createdAt)
		if err != nil {
			http.Error(w, "Failed to parse receipt data", http.StatusInternalServerError)
			return
		}

		// Decode the receipt_data JSON
		var receipt map[string]interface{}
		err = json.Unmarshal(receiptData, &receipt)
		if err != nil {
			http.Error(w, "Failed to decode receipt JSON", http.StatusInternalServerError)
			return
		}

		// Add metadata to the receipt
		receipt["id"] = id
		receipt["created_at"] = createdAt

		receipts = append(receipts, receipt)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(receipts)
}

func GetReceiptByIDHandler(w http.ResponseWriter, r *http.Request) {
	// Get the receipt ID from the URL
	id := mux.Vars(r)["id"]

	// Query the receipt from the database
	var receiptData []byte
	var createdAt time.Time
	query := `SELECT receipt_data, created_at FROM receipts WHERE id = $1`
	err := db.DB.QueryRow(query, id).Scan(&receiptData, &createdAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Receipt not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to retrieve receipt", http.StatusInternalServerError)
		}
		return
	}

	// Decode the receipt_data JSON
	var receipt map[string]interface{}
	err = json.Unmarshal(receiptData, &receipt)
	if err != nil {
		http.Error(w, "Failed to decode receipt JSON", http.StatusInternalServerError)
		return
	}

	// Add metadata to the receipt
	receipt["id"] = id
	receipt["created_at"] = createdAt

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(receipt)
}
