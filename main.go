package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
)

const (
	apiType       = "rest"
	listPath      = "/api/v2/packages/%s/versions/%s/%s/operations"
	createPath    = "/api/v3/packages/%s/versions/%s/%s/groups"
	updatePath    = "/api/v3/packages/%s/versions/%s/%s/groups/%s"
	pageSize      = 100
	personalToken = "X-Personal-Access-Token"
)

type Operation struct {
	OperationID string         `json:"operationId"`
	CustomTags  map[string]any `json:"customTags,omitempty"`
	PackageRef  string         `json:"packageRef"`
}

type OperationRef struct {
	OperationID string `json:"operationId"`
}

type ListResponse struct {
	Operations []Operation `json:"operations"`
}

func main() {
	apihubUrl := flag.String("apihubURL", "", "Base URL of the Apihub instance")
	packageID := flag.String("packageId", "", "Package unique identifier (full alias)")
	version := flag.String("version", "", "Package version")
	groupName := flag.String("group", "", "Operation group name")
	apiKey := flag.String("token", "", "Personal API key")
	flag.Parse()

	if *apihubUrl == "" || *packageID == "" || *version == "" || *groupName == "" || *apiKey == "" {
		fmt.Println("Missing required parameters")
		flag.Usage()
		os.Exit(1)
	}

	// List all operations
	operations, err := listOperations(*apihubUrl, *packageID, *version, *apiKey)
	if err != nil {
		fmt.Printf("Error listing operations: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Operations count: %d\n", len(operations))

	// Filter operations by custom tag
	filteredOps := filterOperations(operations)
	fmt.Printf("Found %d fileter operations matching conditions\n", len(filteredOps))

	if len(filteredOps) == 0 {
		fmt.Println("No operations matching criteria found, exiting")
		return
	}

	// Create new group
	if err := createGroup(*apihubUrl, *packageID, *version, *groupName, *apiKey); err != nil {
		fmt.Printf("Error creating group: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Group created successfully")

	// Update group with operations
	if err := updateGroupOperations(*apihubUrl, *packageID, *version, *groupName, filteredOps, *apiKey); err != nil {
		fmt.Printf("Error updating group: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Group updated with %d operations\n", len(filteredOps))
}

func listOperations(apihubUrl, packageID, version, apiKey string) ([]Operation, error) {
	var allOps []Operation
	page := 0

	for {
		path := fmt.Sprintf(listPath, packageID, version, apiType)
		reqURL := fmt.Sprintf("%s%s?skipRefs=true&limit=%d&page=%d", apihubUrl, path, pageSize, page)

		req, err := http.NewRequest("GET", reqURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set(personalToken, apiKey)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
		}

		var listResp ListResponse
		if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
			return nil, err
		}

		allOps = append(allOps, listResp.Operations...)
		if len(listResp.Operations) < pageSize {
			break
		}
		page++
	}
	return allOps, nil
}

func filterOperations(ops []Operation) []Operation {
	customTagKey := "x-abc"
	targetValue := "def"

	var filtered []Operation
	for _, op := range ops {
		val, exists := op.CustomTags[customTagKey]
		if !exists {
			continue
		}

		var found bool
		switch v := val.(type) {
		case string:
			found = (v == targetValue)
		case []string:
			for _, s := range v {
				if s == targetValue {
					found = true
					break
				}
			}
		case []interface{}:
			for _, elem := range v {
				if s, ok := elem.(string); ok && s == targetValue {
					found = true
					break
				}
			}
		}

		if found {
			filtered = append(filtered, op)
		}
	}
	return filtered
}

func createGroup(apihubUrl, packageID, version, groupName, apiKey string) error {
	path := fmt.Sprintf(createPath, packageID, version, apiType)
	reqURL := fmt.Sprintf("%s%s", apihubUrl, path)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	if err := writer.WriteField("groupName", groupName); err != nil {
		return err
	}
	writer.Close()

	req, err := http.NewRequest("POST", reqURL, body)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set(personalToken, apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status: %d, body: %s", resp.StatusCode, string(body))
	}
	return nil
}

func updateGroupOperations(apihubUrl, packageID, version, groupName string, operations []Operation, apiKey string) error {
	path := fmt.Sprintf(updatePath, packageID, version, apiType, url.PathEscape(groupName))
	reqURL := fmt.Sprintf("%s%s", apihubUrl, path)

	// Prepare operations payload
	operationRefs := make([]OperationRef, len(operations))
	for i, op := range operations {
		operationRefs[i] = OperationRef{OperationID: op.OperationID}
	}

	operationsJSON, err := json.Marshal(operationRefs)
	if err != nil {
		return err
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormField("operations")
	if err != nil {
		return err
	}
	part.Write(operationsJSON)
	writer.Close()

	req, err := http.NewRequest("PATCH", reqURL, body)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set(personalToken, apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status: %d, body: %s", resp.StatusCode, string(body))
	}
	return nil
}
