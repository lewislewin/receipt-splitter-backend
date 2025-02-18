package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"receipt-splitter-backend/auth"
	"receipt-splitter-backend/db"
	"receipt-splitter-backend/helpers"
	"receipt-splitter-backend/models"

	"github.com/gorilla/mux"
	openai "github.com/sashabaranov/go-openai"
	"gorm.io/gorm"
)

var openaiClient *openai.Client

// Initialize OpenAI client
func InitOpenAIClient(apiKey string) {
	openaiClient = openai.NewClient(apiKey)
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

// ParseReceiptHandler processes and parses receipts
func ParseReceiptHandler(w http.ResponseWriter, r *http.Request) {
	type ParseRequest struct {
		Receipt string `json:"receipt"`
	}

	var req ParseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.JSONErrorResponse(w, http.StatusBadRequest, "Invalid input")
		return
	}

	base64Image := req.Receipt
	if strings.HasPrefix(base64Image, "data:image") {
		base64Image = strings.Split(base64Image, ",")[1]
	}

	// Decode Base64 image
	_, err := base64.StdEncoding.DecodeString(base64Image)
	if err != nil {
		helpers.JSONErrorResponse(w, http.StatusBadRequest, "Invalid Base64 image data")
		return
	}

	// Extract text using Google Vision API
	extractedText, err := callGoogleVisionAPI(base64Image)
	if err != nil {
		helpers.JSONErrorResponse(w, http.StatusBadRequest, "Error parsing Base64 data")
		return
	}

	// Call OpenAI for parsing the extracted text
	structuredData, err := callOpenAIForParsing(extractedText)
	if err != nil {
		helpers.JSONErrorResponse(w, http.StatusInternalServerError, "Failed to parse receipt with OpenAI API: "+err.Error())
		return
	}

	// Respond with the structured data
	helpers.JSONResponse(w, http.StatusOK, structuredData)
}

func CreateReceiptHandler(w http.ResponseWriter, r *http.Request) {
	// Get the user ID from the context
	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		helpers.JSONErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Decode request body into a receipt object
	var receiptInput struct {
		Name      string               `json:"name"`
		Reason    string               `json:"reason"`
		MonzoID   string               `json:"monzo_id"`
		Items     []models.ReceiptItem `json:"items"`
		Modifiers []models.Modifier    `json:"modifiers"`
	}
	if err := json.NewDecoder(r.Body).Decode(&receiptInput); err != nil {
		helpers.JSONErrorResponse(w, http.StatusBadRequest, "Invalid input")
		return
	}

	// Create a new receipt
	receipt := models.Receipt{
		UserID:    userID,
		Name:      receiptInput.Name,
		Reason:    receiptInput.Reason,
		MonzoID:   receiptInput.MonzoID,
		Items:     receiptInput.Items,
		Modifiers: receiptInput.Modifiers,
	}

	// Save the receipt to the database
	if err := db.DB.Create(&receipt).Error; err != nil {
		helpers.JSONErrorResponse(w, http.StatusInternalServerError, "Failed to store receipt")
		return
	}

	// Respond with the created receipt
	helpers.JSONResponse(w, http.StatusCreated, map[string]interface{}{
		"id":         receipt.ID,
		"user_id":    receipt.UserID,
		"name":       receipt.Name,
		"reason":     receipt.Reason,
		"monzo_id":   receipt.MonzoID,
		"items":      receipt.Items,
		"modifiers":  receipt.Modifiers,
		"created_at": receipt.CreatedAt,
	})
}

func GetAllReceiptsHandler(w http.ResponseWriter, r *http.Request) {
	// Get user ID from the context
	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		helpers.JSONErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Fetch all receipts for the user with associated items and modifiers
	var receipts []models.Receipt
	if err := db.DB.Preload("Items").Preload("Modifiers").Where("user_id = ?", userID).Find(&receipts).Error; err != nil {
		helpers.JSONErrorResponse(w, http.StatusInternalServerError, "Failed to fetch receipts")
		return
	}

	// Format receipts for the response
	var formattedReceipts []map[string]interface{}
	for _, receipt := range receipts {
		formattedReceipts = append(formattedReceipts, map[string]interface{}{
			"id":         receipt.ID,
			"user_id":    receipt.UserID,
			"name":       receipt.Name,
			"reason":     receipt.Reason,
			"monzo_id":   receipt.MonzoID,
			"items":      receipt.Items,
			"modifiers":  receipt.Modifiers,
			"created_at": receipt.CreatedAt,
		})
	}

	// Respond with the list of receipts
	helpers.JSONResponse(w, http.StatusOK, formattedReceipts)
}

func GetReceiptByIDHandler(w http.ResponseWriter, r *http.Request) {
	// Get the receipt ID from the URL
	id := mux.Vars(r)["id"]

	// Fetch the receipt, including associated items and modifiers
	var receipt models.Receipt
	if err := db.DB.Preload("Items").Preload("Modifiers").First(&receipt, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			helpers.JSONErrorResponse(w, http.StatusNotFound, "Receipt not found")
			return
		}
		helpers.JSONErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve receipt")
		return
	}

	// Construct the receipt data manually
	receiptData := map[string]interface{}{
		"id":         receipt.ID,
		"user_id":    receipt.UserID,
		"name":       receipt.Name,
		"reason":     receipt.Reason,
		"monzo_id":   receipt.MonzoID,
		"items":      receipt.Items,
		"modifiers":  receipt.Modifiers,
		"created_at": receipt.CreatedAt,
	}

	// Respond with the receipt
	helpers.JSONResponse(w, http.StatusOK, receiptData)
}
