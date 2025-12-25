package models_r4

type Entry struct {
	Resource struct {
		ResourceType string `json:"resourceType"`
		Id           string `json:"id"`
		// Organization fields
		Address []struct {
			PostalCode string   `json:"postalCode"`
			City       string   `json:"city"`
			Line       []string `json:"line"`
		} `json:"address"`
		Name string `json:"name"`
		// PractitionerRole fields
		Practitioner struct {
			Reference string `json:"reference"`
		} `json:"practitioner"`
		Organization struct {
			Reference string `json:"reference"`
		} `json:"organization"`
	} `json:"resource"`
}

func (e *Entry) GetId() string {
	return e.Resource.Id
}

func (e *Entry) GetResourceType() string {
	return e.Resource.ResourceType
}

func (e *Entry) GetPractitionerReference() string {
	if e.Resource.Practitioner.Reference == "" {
		return ""
	}
	return e.Resource.Practitioner.Reference[13:]
}

func (e *Entry) GetOrganizationReference() string {
	if e.Resource.Organization.Reference == "" {
		return ""
	}
	return e.Resource.Organization.Reference[13:]
}

func (e *Entry) GetAll() map[string]interface{} {
	result := make(map[string]interface{})
	result["id"] = e.Resource.Id
	result["resourceType"] = e.Resource.ResourceType
	result["address"] = e.Resource.Address
	result["name"] = e.Resource.Name
	result["practitionerReference"] = e.GetPractitionerReference()
	result["organizationReference"] = e.GetOrganizationReference()
	return result
}
