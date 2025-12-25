package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	fhir "github.com/LGMorgan/go-fhir"
	fhirInterface "github.com/LGMorgan/go-fhir/interface"
	models_r4 "github.com/LGMorgan/go-fhir/versions/r4/models"
	"github.com/joho/godotenv"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func main() {
	log.Println("FetchAllPhysiotherapists")

	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println("🤯 Error loading .env file")
	}
	apiKey := os.Getenv("ESANTE_API_KEY")

	clientFhir := fhir.New("https://gateway.api.esante.gouv.fr/fhir/v2", "ESANTE-API-KEY", apiKey, fhir.R4)

	// LIMIT 50
	clientFhir.SetEntryLimit(50)
	clientFhir.SetTimeout(30)

	bundleRes := clientFhir.
		Search(fhirInterface.ORGANIZATION).
		Where(models_r4.Organization{}.
			Address.Contains().Value("974")).
		Or(models_r4.Organization{}.
			Address.Contains().Value("976")).
		RevInclude("PractitionerRole:organization").
		ReturnBundle().Execute()

	res, ok := bundleRes.(*models_r4.BundleResult)
	if !ok {
		log.Println("error bundleRes type assertion")
		return
	}

	log.Println("✅ Found ", len(res.Entry), " entries in Mayotte or Reunion")

	for {
		// Step 1: Build organization map from entries
		orgMap := make(map[string]*models_r4.Entry)
		practitionerRoles := []models_r4.Entry{}

		for i, e := range res.Entry {
			if e.GetResourceType() == "Organization" {
				orgMap[e.GetId()] = &res.Entry[i]
			} else if e.GetResourceType() == "PractitionerRole" {
				practitionerRoles = append(practitionerRoles, e)
			}
		}

		log.Println("📊 Organizations: ", len(orgMap), " | PractitionerRoles: ", len(practitionerRoles))

		// Step 2: Process PractitionerRole entries
		for _, prEntry := range practitionerRoles {
			practitionerId := prEntry.GetPractitionerReference()
			orgId := prEntry.GetOrganizationReference()

			// Get the organization info
			org := orgMap[orgId]
			if org == nil {
				log.Printf("⚠️  Organization %s not found for PractitionerRole %s\n", orgId, prEntry.GetId())
				continue
			}

			//log.Printf("\n✅ Found: %s works at %s\n", practitionerId, org.Resource.Name)
			//log.Printf("   Address: %v\n", org.Resource.Address)

			// Step 3: Fetch Practitioner to check qualification-code = 70
			practitionerRaw := clientFhir.
				Search(fhirInterface.PRACTITIONER).
				ById(practitionerId).
				ReturnRaw().
				Execute()

			var practitioner map[string]interface{}
			err := json.Unmarshal(practitionerRaw.([]byte), &practitioner)
			if err != nil {
				log.Printf("❌ Error parsing Practitioner %s: %v\n", practitionerId, err)
				continue
			}

			// Check qualification array for code = "70"
			if !hasQualificationCode(practitioner, "70") {
				//log.Printf("❌ %s is NOT a physiotherapist (no code 70)\n", practitionerId)
				continue
			}
			/*
				log.Printf("✅ %s is a physiotherapist (code 70)\n", practitionerId)
				log.Printf("   PractitionerRole ID: %s\n", prEntry.GetId())
				log.Printf("   Organization: %s (%s)\n", org.Resource.Name, orgId)
				if len(org.Resource.Address) > 0 {
					addr := org.Resource.Address[0]
					log.Printf("   Location: %s %s %s\n", addr.PostalCode, addr.City, strings.Join(addr.Line, ", "))
				}*/

			// Extract data from practitioner and organization
			lastname := ToTile(extractLastnameFromJson(practitionerRaw.([]byte)))
			firstname := ToTile(extractFirstnameFromJson(practitionerRaw.([]byte)))
			rpps := extractRppsFromJson(practitionerRaw.([]byte))
			email := strings.ToLower(extractEmailFromJson(practitionerRaw.([]byte)))
			phone := strings.ReplaceAll(extractPhoneFromJson(practitionerRaw.([]byte)), " ", "")
			address := extractAddressFromOrganization(org)

			log.Printf("   Name: %s %s\n", firstname, lastname)
			log.Printf("   Phone: %s\n", phone)
			log.Printf("   Email: %s\n", email)
			log.Printf("   RPPS: %s\n", rpps)
			log.Printf("   Organization: %s (%s)\n", org.Resource.Name, orgId)
			log.Printf("   Address: %s\n", address.Address)
			log.Printf("   City: %s\n", address.City)
			log.Printf("   Zipcode: %d\n", address.Zipcode)
			log.Printf("   Department: %s\n", address.Department)

			if rpps == "" {
				continue
			}

		}

		if res.GetNextLink() == "" {
			break
		}
		res = clientFhir.LoadPage().Next(res).Execute().(*models_r4.BundleResult)
	}
}

func hasQualificationCode(practitioner map[string]interface{}, targetCode string) bool {
	if qual, ok := practitioner["qualification"].([]interface{}); ok {
		for _, q := range qual {
			if qMap, ok := q.(map[string]interface{}); ok {
				if codeObj, ok := qMap["code"].(map[string]interface{}); ok {
					if coding, ok := codeObj["coding"].([]interface{}); ok {
						for _, code := range coding {
							if cMap, ok := code.(map[string]interface{}); ok {
								// Check TRE_G15-ProfessionSante system for profession code
								if system, ok := cMap["system"].(string); ok {
									if strings.Contains(system, "TRE-G15-ProfessionSante") {
										if codeVal, ok := cMap["code"].(string); ok && codeVal == targetCode {
											return true
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
	return false
}

func extractRppsFromJson(jsonData []byte) string {
	dec := json.NewDecoder(strings.NewReader(string(jsonData)))
	var rpps string

	for {
		t, err := dec.Token()
		if err != nil {
			break
		}
		if s, ok := t.(string); ok {
			if s == "system" {
				t, err := dec.Token()
				if err != nil {
					break
				}
				// Check for RPPS system
				if systemVal, ok := t.(string); ok && systemVal == "https://rpps.esante.gouv.fr" {
					// Now look for the value field
					for {
						t, err := dec.Token()
						if err != nil {
							break
						}
						if s, ok := t.(string); ok && s == "value" {
							t, err := dec.Token()
							if err != nil {
								break
							}
							rpps = t.(string)
							return rpps
						}
					}
				}
			}
		}
	}

	return rpps
}

func extractFirstnameFromJson(jsonData []byte) string {
	dec := json.NewDecoder(strings.NewReader(string(jsonData)))
	var firstname string

	for {
		t, err := dec.Token()
		if err != nil {
			break
		}
		if s, ok := t.(string); ok {
			if s == "given" {
				_, err := dec.Token()
				if err != nil {
					break
				}
				for {
					t, err := dec.Token()
					if err != nil {
						break
					}
					if _, ok := t.(json.Delim); ok {
						break
					}
					if firstname == "" {
						firstname = t.(string)
					} else {
						firstname += " " + t.(string)
					}
				}
				break
			}
		}
	}

	return firstname
}

func extractLastnameFromJson(jsonData []byte) string {
	dec := json.NewDecoder(bytes.NewReader(jsonData))
	var lastname string

	for {
		t, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return ""
		}

		if key, ok := t.(string); ok && key == "family" {
			t, err = dec.Token()
			if err == io.EOF {
				break
			}
			if err != nil {
				return ""
			}
			lastname = fmt.Sprintf("%v", t)
			break
		}
	}

	return lastname
}

func extractEmailFromJson(jsonData []byte) string {
	dec := json.NewDecoder(bytes.NewReader(jsonData))

	for {
		t, err := dec.Token()
		if err != nil {
			return "" // Return empty string if not found
		}
		if s, ok := t.(string); ok {
			if s == "system" {
				t, err := dec.Token()
				if err != nil {
					return ""
				}
				if systemVal, ok := t.(string); ok && systemVal == "email" {
					for {
						t, err := dec.Token()
						if err != nil {
							return ""
						}
						if s, ok := t.(string); ok && s == "value" {
							t, err := dec.Token()
							if err != nil {
								return ""
							}
							return t.(string)
						}
					}
				}
			}
		}
	}
}

func extractPhoneFromJson(jsonData []byte) string {
	dec := json.NewDecoder(strings.NewReader(string(jsonData)))

	for {
		t, err := dec.Token()
		if err != nil {
			return "" // Return empty string if not found
		}
		if s, ok := t.(string); ok {
			if s == "system" {
				t, err := dec.Token()
				if err != nil {
					return ""
				}
				if systemVal, ok := t.(string); ok && systemVal == "phone" {
					for {
						t, err := dec.Token()
						if err != nil {
							return ""
						}
						if s, ok := t.(string); ok && s == "value" {
							t, err := dec.Token()
							if err != nil {
								return ""
							}
							return t.(string)
						}
					}
				}
			}
		}
	}
}

func ToTile(s string) string {
	return cases.Title(language.Und, cases.NoLower).String(strings.ToLower(s))
}

func extractAddressFromOrganization(org *models_r4.Entry) *Address {
	if org == nil || len(org.Resource.Address) == 0 {
		return nil
	}

	addr := org.Resource.Address[0]
	a := &Address{
		Address: strings.Join(addr.Line, " "),
		City:    addr.City,
	}

	// Convert postal code to int and get department
	if code, err := strconv.Atoi(addr.PostalCode); err == nil {
		a.Zipcode = code
		a.Department = department(code)
	}

	return a
}

type Address struct {
	Address    string
	City       string
	Zipcode    int
	Department string
	Lat        float64
	Lng        float64
}

func department(zipcode int) string {
	// 974 <= La Réunion
	// 976 <= Mayotte

	if zipcode >= 97400 && zipcode <= 97499 {
		return "Reunion"
	}

	if zipcode >= 97600 && zipcode <= 97699 {
		return "Mayotte"
	}

	return ""
}
