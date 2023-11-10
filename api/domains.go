package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	defaultLimit = 500

	pathDomainRecords       = "%s/v1/domains/%s/records?limit=%d&offset=%d"
	pathDomainRecordsAdd    = "%s/v1/domains/%s/records"
	pathDomainRecordsUpdate = "%s/v1/domains/%s/records/%s/%s"
	pathDomainRecordsByType = "%s/v1/domains/%s/records/%s"
	pathDomains             = "%s/v1/domains/%s"

	//to use v2 api
	shoppers               = "%s/v1/shoppers/%s?includes=customerId"
	pathDomainsNameServers = "%s/v2/customers/%s/domains/%s/nameServers"
)

// GetDomains fetches the details for the provided domain
func (c *Client) GetDomains(customerID string) ([]Domain, error) {
	domainURL := fmt.Sprintf(pathDomains, c.baseURL, "")
	req, err := http.NewRequest(http.MethodGet, domainURL, nil)

	if err != nil {
		return nil, err
	}

	var d []Domain
	if err := c.execute(customerID, req, &d); err != nil {
		return nil, err
	}

	return d, nil
}

// GetDomain fetches the details for the provided domain
func (c *Client) GetDomain(customerID, domain string) (*Domain, error) {
	domainURL := fmt.Sprintf(pathDomains, c.baseURL, domain)
	req, err := http.NewRequest(http.MethodGet, domainURL, nil)

	if err != nil {
		return nil, err
	}

	d := new(Domain)
	for {
		if err := c.execute(customerID, req, &d); err != nil {
			return nil, err
		}

		if !strings.Contains(d.Status, "PENDING") {
			break
		}

		time.Sleep(3 * time.Second)
	}

	return d, nil
}

// UpdateNSDomain ...
func (c *Client) UpdateNSDomain(ns []string, customerID, domain string) error {
	t := &struct {
		NameServers []string `json:"nameServers"`
	}{
		NameServers: ns,
	}

	msg, err := json.Marshal(t)
	if err != nil {
		return err
	}

	buffer := bytes.NewBuffer(msg)

	domainURL := fmt.Sprintf(pathDomains, c.baseURL, domain)
	req, err := http.NewRequest(http.MethodPatch, domainURL, buffer)

	if err != nil {
		return err
	}

	if err = c.execute(customerID, req, nil); err != nil {
		return err
	}

	return nil
}

// GetDomainRecords fetches all existing records for the provided domain
func (c *Client) GetDomainRecords(customerID, domain string) ([]*DomainRecord, error) {
	offset := 1
	records := make([]*DomainRecord, 0)
	for {
		page := make([]*DomainRecord, 0)
		domainURL := fmt.Sprintf(pathDomainRecords, c.baseURL, domain, defaultLimit, offset)
		req, err := http.NewRequest(http.MethodGet, domainURL, nil)

		if err != nil {
			return nil, err
		}

		if err := c.execute(customerID, req, &page); err != nil {
			return nil, err
		}
		if len(page) == 0 {
			break
		}
		offset += 1
		records = append(records, page...)
	}

	return records, nil
}

// AddDomainRecords adds records without affecting existing ones on the provided domain
func (c *Client) AddDomainRecords(customerID, domain string, records []*DomainRecord) error {
	for t := range supportedTypes {
		typeRecords := c.domainRecordsOfType(t, records)
		if IsDisallowed(t, typeRecords) {
			continue
		}

		msg, err := json.Marshal(typeRecords)
		if err != nil {
			return err
		}

		buffer := bytes.NewBuffer(msg)
		domainURL := fmt.Sprintf(pathDomainRecordsAdd, c.baseURL, domain)
		log.Println(domainURL)
		log.Println(buffer)

		// set method to patch to only add records
		// for more info check: https://developer.godaddy.com/doc/endpoint/domains#/v1/recordAdd
		req, err := http.NewRequest(http.MethodPatch, domainURL, buffer)
		if err != nil {
			return err
		}

		if err := c.execute(customerID, req, nil); err != nil {
			return err
		}
	}

	return nil
}

// ReplaceDomainRecords overwrites all existing records with the ones provided
func (c *Client) ReplaceDomainRecords(customerID, domain string, records []*DomainRecord) error {
	for t := range supportedTypes {
		typeRecords := c.domainRecordsOfType(t, records)
		if IsDisallowed(t, typeRecords) {
			continue
		}

		msg, err := json.Marshal(typeRecords)
		if err != nil {
			return err
		}

		domainURL := fmt.Sprintf(pathDomainRecordsByType, c.baseURL, domain, t)
		buffer := bytes.NewBuffer(msg)

		log.Println(domainURL)
		log.Println(buffer)

		// set method to put to replace all existing records
		// for more info check: https://developer.godaddy.com/doc/endpoint/domains#/v1/recordReplaceType
		req, err := http.NewRequest(http.MethodPut, domainURL, buffer)
		if err != nil {
			return err
		}

		if err := c.execute(customerID, req, nil); err != nil {
			return err
		}
	}

	return nil
}

// AddDomainRecords adds records without affecting existing ones on the provided domain
func (c *Client) UpdateDomainRecords(customerID, domain string, records []*DomainRecord) error {
	for _, rec := range records {
		// typeRecords := c.domainRecordsOfType(t, records)
		t := rec.Type
		// if IsDisallowed(t, typeRecords) {
		// 	continue
		// }

		msg, err := json.Marshal([]*DomainRecord{rec})
		if err != nil {
			return err
		}

		buffer := bytes.NewBuffer(msg)
		domainURL := fmt.Sprintf(pathDomainRecordsUpdate, c.baseURL, domain, t, rec.Name)
		log.Println(domainURL)
		log.Println(buffer)

		req, err := http.NewRequest(http.MethodPut, domainURL, buffer)
		if err != nil {
			return err
		}

		if err := c.execute(customerID, req, nil); err != nil {
			return err
		}
	}

	return nil
}

// // GetDomainRecords fetches all existing records for the provided domain
// func (c *Client) GetShoppers(customerID string) (string, error) {

// 	page := make([]*DomainRecord, 0)
// 	domainURL := fmt.Sprintf(shoppers, c.baseURL, c.customerID)
// 	req, err := http.NewRequest(http.MethodGet, domainURL, nil)

// 	if err != nil {
// 		return "", err
// 	}

// 	if err := c.execute(customerID, req, &page); err != nil {
// 		return "", err
// 	}

// 	return "", nil
// }

// func (c *Client) GetPoll(domain string) (string, error) {

// 	page := make([]*DomainRecord, 0)
// 	domainURL := fmt.Sprintf("%s/v2/customers/%s/domains/%s/actions", c.baseURL, "01804e6f-6562-4304-be20-6563ee6d7573", domain)
// 	req, err := http.NewRequest(http.MethodGet, domainURL, nil)

// 	if err != nil {
// 		return "", err
// 	}

// 	//req.Header.Set("x-shopper-id", "467624111")

// 	if err := c.execute("467624111", req, &page); err != nil {
// 		return "", err
// 	}

// 	return "", nil
// }

// // AddNSRecords adds NS records
// func (c *Client) UpdateDomainInfo(domain string, ns []string) error {
// 	t := &struct {
// 		NameServers []string `json:"nameServers"`
// 	}{
// 		NameServers: ns,
// 	}

// 	msg, err := json.Marshal(t)
// 	if err != nil {
// 		return err
// 	}

// 	buffer := bytes.NewBuffer(msg)
// 	domainURL := fmt.Sprintf(pathDomainsNameServers, c.baseURL, "01804e6f-6562-4304-be20-6563ee6d7573" /*c.customerID*/, domain)

// 	// set method to patch to only add records
// 	// for more info check: https://developer.godaddy.com/doc/endpoint/domains#/v1/recordAdd
// 	req, err := http.NewRequest(http.MethodPut, domainURL, buffer)
// 	if err != nil {
// 		return err
// 	}

// 	req.Header.Set("domain", domain)
// 	if err = c.execute(c.customerID, req, nil); err != nil {
// 		return err
// 	}

// 	return err
// }

// // AddNSRecords adds NS records
// func (c *Client) AddNSRecords(domain string, records []string) error {
// 	t := &struct {
// 		NameServers []string `json:nameServers`
// 	}{
// 		NameServers: records,
// 	}

// 	msg, err := json.Marshal(t)
// 	if err != nil {
// 		return err
// 	}

// 	buffer := bytes.NewBuffer(msg)
// 	domainURL := fmt.Sprintf(pathDomainsNameServers, c.baseURL, c.customerID, domain)
// 	log.Println(domainURL)
// 	log.Println(buffer)

// 	// set method to patch to only add records
// 	// for more info check: https://developer.godaddy.com/doc/endpoint/domains#/v1/recordAdd
// 	_, err = http.NewRequest(http.MethodPut, domainURL, buffer)

// 	// if err := c.execute(customerID, req, nil); err != nil {
// 	// 	return err
// 	// }

// 	return err
// }

func (c *Client) domainRecordsOfType(t string, records []*DomainRecord) []*DomainRecord {
	typeRecords := make([]*DomainRecord, 0)

	for _, record := range records {
		if strings.EqualFold(record.Type, t) {
			typeRecords = append(typeRecords, record)
		}
	}

	return typeRecords
}
