package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
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

			// Step 3: Fetch Practitioner ID with qualification-code = 70
			practitionerRaw := clientFhir.
				Search(fhirInterface.PRACTITIONER).
				ById(practitionerId).
				And(models_r4.Practitioner{}.QualificationCode.Contains().Value("70")).
				ReturnRaw().
				Execute()

			if practitionerRaw == nil {
				continue
			}

			var bundle map[string]interface{}
			err := json.Unmarshal(practitionerRaw.([]byte), &bundle)
			if err != nil {
				log.Printf("❌ Error parsing response for %s: %v\n", practitionerId, err)
				continue
			}

			// Check if the bundle has entries (filter matched)
			entries, ok := bundle["entry"].([]interface{})
			if !ok || len(entries) == 0 {
				// No practitioner with code 70 found
				continue
			}

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
	if org == nil {
		return nil
	}

	// Marshal the resource back to JSON to preserve _line extensions
	jsonData, err := json.Marshal(org.Resource)
	if err != nil {
		log.Println("Error marshaling organization resource:", err)
		return nil
	}

	// Use the robust JSON parser
	addresses, err := extractAddressesFromJson(jsonData)
	if err != nil {
		return nil
	}

	if len(addresses) == 0 {
		return nil
	}

	return &addresses[0]
}

func extractAddressesFromJson(jsonData []byte) ([]Address, error) {
	var result []Address
	var obj map[string]interface{}

	err := json.Unmarshal(jsonData, &obj)
	if err != nil {
		return nil, err
	}

	if addresses, ok := obj["address"].([]interface{}); ok {
		for _, addr := range addresses {
			var a Address
			if addrMap, ok := addr.(map[string]interface{}); ok {
				// Extract line (pre-formatted address)
				if line, ok := addrMap["line"].([]interface{}); ok && len(line) > 0 {
					if line[0] != nil {
						a.Address = strings.TrimSpace(line[0].(string))
					}
				}

				var houseNumber, streetNameType, buildingNumberSuffix, streetNameBase, lieuDit string

				// Extract _line extensions (structured components)
				if line, ok := addrMap["_line"].([]interface{}); ok && len(line) > 0 {
					if lineMap, ok := line[0].(map[string]interface{}); ok {
						if ext, ok := lineMap["extension"].([]interface{}); ok {
							for _, e := range ext {
								if eMap, ok := e.(map[string]interface{}); ok {
									if url, ok := eMap["url"].(string); ok && url == "http://hl7.org/fhir/StructureDefinition/iso21090-ADXP-houseNumber" {
										if value, ok := eMap["valueString"].(string); ok {
											houseNumber = strings.TrimSpace(value)
										}
									}
									if url, ok := eMap["url"].(string); ok && url == "http://hl7.org/fhir/StructureDefinition/iso21090-ADXP-streetNameType" {
										if value, ok := eMap["valueString"].(string); ok {
											streetNameType = strings.TrimSpace(value)
										}
									}
									if url, ok := eMap["url"].(string); ok && url == "http://hl7.org/fhir/StructureDefinition/iso21090-ADXP-buildingNumberSuffix" {
										if value, ok := eMap["valueString"].(string); ok {
											buildingNumberSuffix = strings.TrimSpace(value)
										}
									}
									if url, ok := eMap["url"].(string); ok && url == "http://hl7.org/fhir/StructureDefinition/iso21090-ADXP-streetNameBase" {
										if value, ok := eMap["valueString"].(string); ok {
											streetNameBase = strings.TrimSpace(value)
										}
									}
									if url, ok := eMap["url"].(string); ok && url == "https://interop.esante.gouv.fr/ig/fhir/annuaire/StructureDefinition/as-ext-lieu-dit" {
										if value, ok := eMap["valueString"].(string); ok {
											lieuDit = strings.TrimSpace(value)
										}
									}
									if url, ok := eMap["url"].(string); ok && url == "http://hl7.org/fhir/StructureDefinition/iso21090-ADXP-postBox" {
										if value, ok := eMap["valueString"].(string); ok {
											a.City = strings.TrimSpace(value)
										}
									}
									if url, ok := eMap["url"].(string); ok && url == "http://hl7.org/fhir/us/vr-common-library/StructureDefinition/CityCode" {
										if value, ok := eMap["valueString"].(string); ok {
											a.City = strings.TrimSpace(value)
										}
									}
								}
							}
						}
					}
				}

				if a.City == "" {
					if city, ok := addrMap["city"].(string); ok {
						r, err := regexp.Compile(`\d{5}\s+(.*)`)
						if err != nil {
							return nil, err
						}
						match := r.FindStringSubmatch(city)
						if len(match) > 1 {
							a.City = strings.TrimSpace(match[1])
						} else {
							a.City = strings.TrimSpace(city)
						}
					}
				}
				if postalCode, ok := addrMap["postalCode"].(string); ok {
					if code, err := strconv.Atoi(postalCode); err == nil {
						a.Zipcode = code
						a.Department = department(code)
					}
				}
			}

			result = append(result, a)
		}
	}

	return result, nil
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
