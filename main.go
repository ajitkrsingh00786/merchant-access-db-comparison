package main

import (
	"encoding/csv"
	"fmt"
	"github.com/xuri/excelize/v2"
	"os"
	"sort"
	"strconv"
	"time"
)

func main() {
	compDifferencMam()
	compDiffForMapp()
	compDiffForPc()
}

func compDifferencMam() {
	file, err := os.Open("mam_api.csv")
	if err != nil {
		fmt.Println("Error opening mam_api.csv:", err)
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		fmt.Println("Error reading mam_api.csv:", err)
		return
	}
	fmt.Println("mam_api.csv loaded")

	file2, err := os.Open("mam_prts.csv")
	if err != nil {
		fmt.Println("Error opening mam_prts.csv:", err)
		return
	}
	defer file2.Close()

	reader2 := csv.NewReader(file2)
	records2, err := reader2.ReadAll()
	if err != nil {
		fmt.Println("Error reading mam_prts.csv:", err)
		return
	}
	fmt.Println("mam_prts.csv loaded")

	// Parse created_at field and sort records by created_at in descending order
	sortRecords := func(records [][]string) {
		sort.Slice(records, func(i, j int) bool {
			timeIInt, errI := strconv.ParseInt(records[i][5], 10, 64)
			timeJInt, errJ := strconv.ParseInt(records[j][5], 10, 64)

			if errI != nil || errJ != nil {
				return false
			}
			timeI := time.Unix(timeIInt, 0)
			timeJ := time.Unix(timeJInt, 0)

			return timeI.After(timeJ)
		})
	}
	sortRecords1 := func(records [][]string) {
		sort.Slice(records, func(i, j int) bool {
			timeIInt, errI := strconv.ParseInt(records[i][6], 10, 64)
			timeJInt, errJ := strconv.ParseInt(records[j][6], 10, 64)

			if errI != nil || errJ != nil {
				return false
			}

			timeI := time.Unix(timeIInt, 0)
			timeJ := time.Unix(timeJInt, 0)

			return timeI.After(timeJ)
		})
	}
	sortRecords(records[1:])   // Sort mam_api records (skip header)
	sortRecords1(records2[1:]) // Sort mam_prts records (skip header)

	// Map records2 for faster lookup by ID
	records2Map := make(map[string][]string)
	for _, rec := range records2[1:] { // Skip header
		records2Map[rec[0]] = rec
	}

	// Create a map for records from mam_api.csv for lookup
	recordsMap := make(map[string][]string)
	for _, rec := range records[1:] { // Skip header
		recordsMap[rec[0]] = rec
	}

	// Create an Excel file to save differences
	f := excelize.NewFile()
	sheet := "merchant_access_map"
	f.SetSheetName(f.GetSheetName(0), sheet)
	f.SetCellValue(sheet, "A1", "ID")
	f.SetCellValue(sheet, "B1", "mam_api.csv Record")
	f.SetCellValue(sheet, "C1", "mam_prts.csv Record")
	f.SetCellValue(sheet, "D1", "created_at")
	f.SetCellValue(sheet, "E1", "created_at_data/time")
	f.SetCellValue(sheet, "F1", "Difference Message")
	row := 2

	f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "")
	f.SetCellValue(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("%v", records[0]))
	f.SetCellValue(sheet, fmt.Sprintf("C%d", row), fmt.Sprintf("%v", records2[0]))
	f.SetCellValue(sheet, fmt.Sprintf("D%d", row), "")
	f.SetCellValue(sheet, fmt.Sprintf("E%d", row), "")
	f.SetCellValue(sheet, fmt.Sprintf("F%d", row), "")
	row = 3

	// Compare records and save differences
	for i, record := range records[1:] { // Skip header
		id := record[0]
		otherRecord, exists := records2Map[id]
		if !exists {
			fmt.Printf("ID %s not found in mam_prts.csv\n", id)
			// Log the ID not found case in Excel
			f.SetCellValue(sheet, fmt.Sprintf("A%d", row), id)
			f.SetCellValue(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("%v", record))
			f.SetCellValue(sheet, fmt.Sprintf("C%d", row), "N/A")
			f.SetCellValue(sheet, fmt.Sprintf("D%d", row), record[5])
			f.SetCellValue(sheet, fmt.Sprintf("E%d", row), convertTimestampToIST(record[5]))
			f.SetCellValue(sheet, fmt.Sprintf("F%d", row), "ID not found in mam_prts.csv")
			row++
			continue
		}

		if differenceMessage := getDifferenceMessage(record, otherRecord); differenceMessage != "" {
			fmt.Printf("Difference found at line %d\nmam_api.csv: %v\nmam_prts.csv: %v\n", i+2, record, otherRecord)
			// Log difference in Excel file with a detailed message
			f.SetCellValue(sheet, fmt.Sprintf("A%d", row), id)
			f.SetCellValue(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("%v", record))
			f.SetCellValue(sheet, fmt.Sprintf("C%d", row), fmt.Sprintf("%v", otherRecord))
			f.SetCellValue(sheet, fmt.Sprintf("D%d", row), record[5])
			f.SetCellValue(sheet, fmt.Sprintf("E%d", row), convertTimestampToIST(record[5]))
			f.SetCellValue(sheet, fmt.Sprintf("F%d", row), differenceMessage)
			row++
		}
	}

	// Check for IDs in mam_prts.csv that are not in mam_api.csv
	for id, record := range records2Map {
		if _, exists := recordsMap[id]; !exists {
			fmt.Printf("ID %s not found in mam_api.csv\n", id)
			// Log the ID not found case in Excel
			f.SetCellValue(sheet, fmt.Sprintf("A%d", row), id)
			f.SetCellValue(sheet, fmt.Sprintf("B%d", row), "N/A")
			f.SetCellValue(sheet, fmt.Sprintf("C%d", row), fmt.Sprintf("%v", record))
			f.SetCellValue(sheet, fmt.Sprintf("D%d", row), record[6])
			f.SetCellValue(sheet, fmt.Sprintf("E%d", row), convertTimestampToIST(record[6]))
			f.SetCellValue(sheet, fmt.Sprintf("F%d", row), "ID not found in mam_api.csv")
			row++
		}
	}

	// Save the Excel file
	if err := f.SaveAs("merchant_access_map.xlsx"); err != nil {
		fmt.Println("Error saving Excel file:", err)
		return
	}
}

func getDifferenceMessage(r1, r2 []string) string {
	fields := []struct {
		r1Index int
		r2Index int
		name    string
	}{
		{0, 0, "ID"}, {1, 1, "merchant_id"}, {2, 3, "entity_id"}, {3, 4, "entity_type"}, {4, 2, "entity_owner_id"}, {8, 5, "has_kyc_access"},
	}

	differences := ""
	for _, field := range fields {
		if r1[field.r1Index] != r2[field.r2Index] {
			differences += fmt.Sprintf("%s differs: %v (mam_api.csv) vs %v (mam_prts.csv); ", field.name, r1[field.r1Index], r2[field.r2Index])
		}
	}

	return differences
}

func compDiffForMapp() {
	// Open and read the first CSV file
	file, err := os.Open("mapp_api.csv")
	if err != nil {
		fmt.Println("Error opening mapp_api.csv:", err)
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		fmt.Println("Error reading mapp_api.csv:", err)
		return
	}
	fmt.Println("mapp_api.csv loaded")

	// Open and read the second CSV file
	file2, err := os.Open("mapp_prts.csv")
	if err != nil {
		fmt.Println("Error opening mapp_prts.csv:", err)
		return
	}
	defer file2.Close()

	reader2 := csv.NewReader(file2)
	records2, err := reader2.ReadAll()
	if err != nil {
		fmt.Println("Error reading mapp_prts.csv:", err)
		return
	}
	fmt.Println("mapp_prts.csv loaded")

	sortRecords := func(records [][]string) {
		sort.Slice(records, func(i, j int) bool {
			timeIInt, errI := strconv.ParseInt(records[i][5], 10, 64)
			timeJInt, errJ := strconv.ParseInt(records[j][5], 10, 64)

			if errI != nil || errJ != nil {
				return false
			}

			timeI := time.Unix(timeIInt, 0)
			timeJ := time.Unix(timeJInt, 0)

			return timeI.After(timeJ) // Descending order
		})
	}
	sortRecords1 := func(records [][]string) {
		sort.Slice(records, func(i, j int) bool {
			// Parse the timestamp from string to int64
			timeIInt, errI := strconv.ParseInt(records[i][6], 10, 64)
			timeJInt, errJ := strconv.ParseInt(records[j][6], 10, 64)

			if errI != nil || errJ != nil {
				return false
			}

			timeI := time.Unix(timeIInt, 0)
			timeJ := time.Unix(timeJInt, 0)

			return timeI.After(timeJ) // Descending order
		})
	}
	sortRecords(records[1:])
	sortRecords1(records2[1:])

	// Map records2 for faster lookup by ID
	records2Map := make(map[string][]string)
	for _, rec := range records2[1:] { // Skip header
		records2Map[rec[0]] = rec
	}

	// Create a map for records from mapp_api.csv for lookup
	recordsMap := make(map[string][]string)
	for _, rec := range records[1:] { // Skip header
		recordsMap[rec[0]] = rec
	}

	// Create an Excel file to save differences
	f := excelize.NewFile()
	sheet := "Mapp_Differences"
	f.SetSheetName(f.GetSheetName(0), sheet)
	f.SetCellValue(sheet, "A1", "ID")
	f.SetCellValue(sheet, "B1", "mapp_api.csv Record")
	f.SetCellValue(sheet, "C1", "mapp_prts.csv Record")
	f.SetCellValue(sheet, "D1", "created_At")
	f.SetCellValue(sheet, "E1", "created_At_Date/time")
	f.SetCellValue(sheet, "F1", "Difference Message")
	row := 2

	// Write header row for reference
	f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "")
	f.SetCellValue(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("%v", records[0]))
	f.SetCellValue(sheet, fmt.Sprintf("C%d", row), fmt.Sprintf("%v", records2[0]))
	f.SetCellValue(sheet, fmt.Sprintf("D%d", row), "")
	f.SetCellValue(sheet, fmt.Sprintf("E%d", row), "")
	f.SetCellValue(sheet, fmt.Sprintf("F%d", row), "")
	row++

	// Compare records and save differences
	for i, record := range records[1:] { // Skip header
		id := record[0]
		otherRecord, exists := records2Map[id]
		if !exists {
			fmt.Printf("ID %s not found in mapp_prts.csv\n", id)
			// Log the ID not found case in Excel
			f.SetCellValue(sheet, fmt.Sprintf("A%d", row), id)
			f.SetCellValue(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("%v", record))
			f.SetCellValue(sheet, fmt.Sprintf("C%d", row), "N/A")
			f.SetCellValue(sheet, fmt.Sprintf("D%d", row), record[4])
			f.SetCellValue(sheet, fmt.Sprintf("E%d", row), convertTimestampToIST(record[4]))
			f.SetCellValue(sheet, fmt.Sprintf("F%d", row), "ID not found in mapp_prts.csv")
			row++
			continue
		}

		if differenceMessage := getMappDifferenceMessage(record, otherRecord); differenceMessage != "" {
			fmt.Printf("Difference found at line %d\nmapp_api.csv: %v\nmapp_prts.csv: %v\n", i+2, record, otherRecord)
			// Log difference in Excel file with a detailed message
			f.SetCellValue(sheet, fmt.Sprintf("A%d", row), id)
			f.SetCellValue(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("%v", record))
			f.SetCellValue(sheet, fmt.Sprintf("C%d", row), fmt.Sprintf("%v", otherRecord))
			f.SetCellValue(sheet, fmt.Sprintf("D%d", row), record[4])
			f.SetCellValue(sheet, fmt.Sprintf("E%d", row), convertTimestampToIST(record[4]))
			f.SetCellValue(sheet, fmt.Sprintf("F%d", row), differenceMessage)
			row++
		}
	}

	// Check for IDs in mapp_prts.csv that are not in mapp_api.csv
	for id, record := range records2Map {
		if _, exists := recordsMap[id]; !exists {
			fmt.Printf("ID %s not found in mapp_api.csv\n", id)
			// Log the ID not found case in Excel
			f.SetCellValue(sheet, fmt.Sprintf("A%d", row), id)
			f.SetCellValue(sheet, fmt.Sprintf("B%d", row), "N/A")
			f.SetCellValue(sheet, fmt.Sprintf("C%d", row), fmt.Sprintf("%v", record))
			f.SetCellValue(sheet, fmt.Sprintf("D%d", row), record[4])
			f.SetCellValue(sheet, fmt.Sprintf("E%d", row), convertTimestampToIST(record[4]))
			f.SetCellValue(sheet, fmt.Sprintf("F%d", row), "ID not found in mapp_api.csv")
			row++
		}
	}

	// Save the Excel file
	if err := f.SaveAs("Mapp_Differences.xlsx"); err != nil {
		fmt.Println("Error saving Excel file:", err)
		return
	}
	fmt.Println("Comparison finished and saved in Mapp_Differences.xlsx")
}

// getMappDifferenceMessage returns a string message detailing the differences between two records
func getMappDifferenceMessage(r1, r2 []string) string {
	fields := []struct {
		r1Index int
		r2Index int
		name    string
	}{
		{0, 0, "ID"}, {1, 1, "merchant_id"}, {2, 2, "type"}, {3, 3, "application_id"},
	}

	differences := ""
	for _, field := range fields {
		if r1[field.r1Index] != r2[field.r2Index] {
			differences += fmt.Sprintf("%s differs: %v (mapp_api.csv) vs %v (mapp_prts.csv); ", field.name, r1[field.r1Index], r2[field.r2Index])
		}
	}

	return differences
}

func compDiffForPc() {
	// Open and read the first CSV file
	file, err := os.Open("pc_api.csv")
	if err != nil {
		fmt.Println("Error opening pc_api.csv:", err)
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		fmt.Println("Error reading pc_api.csv:", err)
		return
	}
	fmt.Println("pc_api.csv loaded")

	// Open and read the second CSV file
	file2, err := os.Open("pc_prts.csv")
	if err != nil {
		fmt.Println("Error opening pc_prts.csv:", err)
		return
	}
	defer file2.Close()

	reader2 := csv.NewReader(file2)
	records2, err := reader2.ReadAll()
	if err != nil {
		fmt.Println("Error reading pc_prts.csv:", err)
		return
	}
	fmt.Println("pc_prts.csv loaded")

	sortRecords := func(records [][]string) {
		sort.Slice(records, func(i, j int) bool {
			timeIInt, errI := strconv.ParseInt(records[i][5], 10, 64)
			timeJInt, errJ := strconv.ParseInt(records[j][5], 10, 64)

			if errI != nil || errJ != nil {
				return false
			}
			timeI := time.Unix(timeIInt, 0)
			timeJ := time.Unix(timeJInt, 0)

			return timeI.After(timeJ) // Descending order
		})
	}
	sortRecords1 := func(records [][]string) {
		sort.Slice(records, func(i, j int) bool {
			timeIInt, errI := strconv.ParseInt(records[i][6], 10, 64)
			timeJInt, errJ := strconv.ParseInt(records[j][6], 10, 64)

			if errI != nil || errJ != nil {
				return false // default to false in case of parsing error
			}

			timeI := time.Unix(timeIInt, 0)
			timeJ := time.Unix(timeJInt, 0)

			return timeI.After(timeJ) // Descending order
		})
	}
	sortRecords(records[1:])
	sortRecords1(records2[1:])

	records2Map := make(map[string][]string)
	for _, rec := range records2[1:] { // Skip header
		records2Map[rec[0]] = rec
	}

	recordsMap := make(map[string][]string)
	for _, rec := range records[1:] { // Skip header
		recordsMap[rec[0]] = rec
	}

	// Create an Excel file to save differences
	f := excelize.NewFile()
	sheet := "partner_config_Differences"
	f.SetSheetName(f.GetSheetName(0), sheet)
	f.SetCellValue(sheet, "A1", "ID")
	f.SetCellValue(sheet, "B1", "pc_api.csv Record")
	f.SetCellValue(sheet, "C1", "pc_prts.csv Record")
	f.SetCellValue(sheet, "D1", "created_at")
	f.SetCellValue(sheet, "E1", "created_at_Date/time")
	f.SetCellValue(sheet, "F1", "Difference Message")
	row := 2

	// Write header row for reference
	f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "")
	f.SetCellValue(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("%v", records[0]))
	f.SetCellValue(sheet, fmt.Sprintf("C%d", row), fmt.Sprintf("%v", records2[0]))
	f.SetCellValue(sheet, fmt.Sprintf("D%d", row), "")
	f.SetCellValue(sheet, fmt.Sprintf("E%d", row), "")
	f.SetCellValue(sheet, fmt.Sprintf("F%d", row), "")
	row++

	// Compare records and save differences
	for i, record := range records[1:] { // Skip header
		id := record[0]
		otherRecord, exists := records2Map[id]
		if !exists {
			fmt.Printf("ID %s not found in pc_prts.csv\n", id)
			// Log the ID not found case in Excel
			f.SetCellValue(sheet, fmt.Sprintf("A%d", row), id)
			f.SetCellValue(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("%v", record))
			f.SetCellValue(sheet, fmt.Sprintf("C%d", row), "N/A")
			f.SetCellValue(sheet, fmt.Sprintf("D%d", row), record[18])
			f.SetCellValue(sheet, fmt.Sprintf("E%d", row), convertTimestampToIST(record[18]))
			f.SetCellValue(sheet, fmt.Sprintf("F%d", row), "ID not found in pc_prts.csv")
			row++
			continue
		}

		if differenceMessage := getMappDifferenceMessage(record, otherRecord); differenceMessage != "" {
			fmt.Printf("Difference found at line %d\npc_api.csv: %v\npc_prts.csv: %v\n", i+2, record, otherRecord)
			// Log difference in Excel file with a detailed message
			f.SetCellValue(sheet, fmt.Sprintf("A%d", row), id)
			f.SetCellValue(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("%v", record))
			f.SetCellValue(sheet, fmt.Sprintf("C%d", row), fmt.Sprintf("%v", otherRecord))
			f.SetCellValue(sheet, fmt.Sprintf("D%d", row), record[18])
			f.SetCellValue(sheet, fmt.Sprintf("E%d", row), convertTimestampToIST(record[18]))
			f.SetCellValue(sheet, fmt.Sprintf("F%d", row), differenceMessage)
			row++
		}
	}

	// Check for IDs in mapp_prts.csv that are not in mapp_api.csv
	for id, record := range records2Map {
		if _, exists := recordsMap[id]; !exists {
			fmt.Printf("ID %s not found in pc_api.csv\n", id)
			// Log the ID not found case in Excel
			f.SetCellValue(sheet, fmt.Sprintf("A%d", row), id)
			f.SetCellValue(sheet, fmt.Sprintf("B%d", row), "N/A")
			f.SetCellValue(sheet, fmt.Sprintf("C%d", row), fmt.Sprintf("%v", record))
			f.SetCellValue(sheet, fmt.Sprintf("D%d", row), record[20])
			f.SetCellValue(sheet, fmt.Sprintf("E%d", row), convertTimestampToIST(record[20]))
			f.SetCellValue(sheet, fmt.Sprintf("F%d", row), "ID not found in pc_api.csv")
			row++
		}
	}

	// Save the Excel file
	if err := f.SaveAs("partner_config_Differences.xlsx"); err != nil {
		fmt.Println("Error saving Excel file:", err)
		return
	}
	fmt.Println("Comparison finished and saved in partner_config_Differences.xlsx")
}

// getMappDifferenceMessage returns a string message detailing the differences between two records
func getPcDifferenceMessage(r1, r2 []string) string {
	fields := []struct {
		r1Index int
		r2Index int
		name    string
	}{
		{0, 0, "ID"}, {1, 1, "entity_type"}, {2, 2, "entity_id"}, {3, 3, "origin_type"}, {4, 4, "origin_id"}, {5, 5, "commissions_enabled"},
		{6, 6, "default_payment_methods"}, {7, 7, "default_plan_id"}, {8, 8, "implicit_plan_id"}, {9, 9, "implicit_expiry_at"}, {10, 10, "explicit_plan_id"},
		{11, 11, "explicit_refund_fees"}, {12, 12, "explicit_should_charge"}, {13, 13, "commission_model"}, {14, 14, "settle_to_partner"},
		{15, 15, "tds_percentage"}, {16, 16, "has_gst_certificate"}, {17, 18, "revisit_at"}, {21, 17, "sub_merchant_config"}, {22, 19, "partner_metadata"},
	}

	differences := ""
	for _, field := range fields {
		if r1[field.r1Index] != r2[field.r2Index] {
			differences += fmt.Sprintf("%s differs: %v (pc_api.csv) vs %v (pc_prts.csv); ", field.name, r1[field.r1Index], r2[field.r2Index])
		}
	}

	return differences
}

func convertTimestampToIST(record string) string {
	timestamp, err := strconv.ParseInt(record, 10, 64)
	if err != nil {
		fmt.Println("Error parsing timestamp:", err)
		return ""
	}

	dateTime := time.Unix(timestamp, 0)

	istLocation, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		fmt.Println("Error loading IST location:", err)
		return ""
	}

	dateTimeIST := dateTime.In(istLocation)
	return dateTimeIST.Format("2006-01-02 15:04:05")
}
