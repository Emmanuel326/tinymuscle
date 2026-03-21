package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"go.etcd.io/bbolt"
)

func main() {
	fmt.Println("📝 Adding test tenders to TinyMuscle...")

	// Open the BoltDB database directly
	dbPath := "tinymuscle.db"

	db, err := bbolt.Open(dbPath, 0600, nil)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	// Add test portal
	portalData := map[string]interface{}{
		"id":           "test_portal",
		"name":         "Test Procurement Portal",
		"url":          "https://example.com/tenders",
		"goal":         "Extract all open tenders with title, reference_number, issuing_entity, deadline, estimated_value, source_url",
		"interval_min": 60,
	}

	portalJSON, _ := json.Marshal(portalData)

	err = db.Update(func(tx *bbolt.Tx) error {
		// Create portals bucket if it doesn't exist
		portalsBucket, err := tx.CreateBucketIfNotExists([]byte("portals"))
		if err != nil {
			return err
		}

		// Add portal
		return portalsBucket.Put([]byte("test_portal"), portalJSON)
	})

	if err != nil {
		fmt.Printf("⚠️  Error adding portal: %v\n", err)
	} else {
		fmt.Println("✅ Test portal added")
	}

	// Sample tenders
	testTenders := []struct {
		referenceNumber string
		portalID        string
		title           string
		issuingEntity   string
		deadline        time.Time
		estimatedValue  string
		sourceURL       string
		status          string
		version         int
	}{
		{
			referenceNumber: "UNGM-2026-001",
			portalID:        "test_portal",
			title:           "Supply of IT Equipment for UNDP Kenya",
			issuingEntity:   "UNDP Kenya",
			deadline:        time.Now().Add(14 * 24 * time.Hour),
			estimatedValue:  "KES 5,000,000",
			sourceURL:       "https://www.ungm.org/Notice/12345",
			status:          "new",
			version:         1,
		},
		{
			referenceNumber: "KE-TREASURY-2026-045",
			portalID:        "test_portal",
			title:           "Road Construction Project - Nairobi Bypass",
			issuingEntity:   "Kenya National Treasury",
			deadline:        time.Now().Add(7 * 24 * time.Hour),
			estimatedValue:  "KES 50,000,000",
			sourceURL:       "https://www.treasury.go.ke/tenders/045",
			status:          "new",
			version:         1,
		},
		{
			referenceNumber: "AFDB-2026-089",
			portalID:        "test_portal",
			title:           "Solar Power Installation - Rural Electrification",
			issuingEntity:   "African Development Bank",
			deadline:        time.Now().Add(21 * 24 * time.Hour),
			estimatedValue:  "USD 2,500,000",
			sourceURL:       "https://www.afdb.org/tenders/089",
			status:          "updated",
			version:         2,
		},
		{
			referenceNumber: "PPRA-KE-2026-123",
			portalID:        "test_portal",
			title:           "Office Furniture Supply - Ministry of Interior",
			issuingEntity:   "PPRA Kenya",
			deadline:        time.Now().Add(5 * 24 * time.Hour),
			estimatedValue:  "KES 2,500,000",
			sourceURL:       "https://www.ppra.go.ke/tenders/123",
			status:          "new",
			version:         1,
		},
		{
			referenceNumber: "NBI-2026-078",
			portalID:        "test_portal",
			title:           "ICT Infrastructure Upgrade - Nairobi City County",
			issuingEntity:   "Nairobi City County",
			deadline:        time.Now().Add(10 * 24 * time.Hour),
			estimatedValue:  "KES 15,000,000",
			sourceURL:       "https://nairobi.go.ke/tenders/078",
			status:          "new",
			version:         1,
		},
	}

	// Add tenders
	addedCount := 0
	err = db.Update(func(tx *bbolt.Tx) error {
		// Create tenders bucket if it doesn't exist
		tendersBucket, err := tx.CreateBucketIfNotExists([]byte("tenders"))
		if err != nil {
			return err
		}

		for _, tender := range testTenders {
			// Create tender object
			tenderObj := map[string]interface{}{
				"reference_number": tender.referenceNumber,
				"portal_id":        tender.portalID,
				"title":            tender.title,
				"issuing_entity":   tender.issuingEntity,
				"deadline":         tender.deadline.Format(time.RFC3339),
				"estimated_value":  tender.estimatedValue,
				"source_url":       tender.sourceURL,
				"content_hash":     fmt.Sprintf("%x", tender.referenceNumber),
				"version":          tender.version,
				"last_updated":     time.Now().Format(time.RFC3339),
				"status":           tender.status,
			}

			// Store tender with key format: portalID:referenceNumber
			key := []byte(fmt.Sprintf("%s:%s", tender.portalID, tender.referenceNumber))
			data, err := json.Marshal(tenderObj)
			if err != nil {
				fmt.Printf("❌ Failed to marshal tender %s: %v\n", tender.referenceNumber, err)
				continue
			}

			err = tendersBucket.Put(key, data)
			if err != nil {
				fmt.Printf("❌ Failed to add tender %s: %v\n", tender.referenceNumber, err)
			} else {
				fmt.Printf("✅ Added tender: %s - %s\n", tender.referenceNumber, tender.title)
				addedCount++
			}
		}

		return nil
	})

	if err != nil {
		fmt.Printf("⚠️  Error adding tenders: %v\n", err)
	}

	fmt.Printf("\n🎉 Successfully added %d test tenders!\n", addedCount)
	fmt.Println("\n📊 Now refresh your browser at http://localhost:5173 to see them!")
	fmt.Println("🔴 You should see live tenders appearing in the dashboard!")
}
