package models_r4

import (
	"fmt"

	fhirInterface "github.com/LGMorgan/go-fhir/interface"
	parameters_r4 "github.com/LGMorgan/go-fhir/versions/r4/parameters"
)

type PractitionerRole struct {
	Client    fhirInterface.IClient
	Id        string
	Address   fhirInterface.FhirAddress
	Name      fhirInterface.FhirName
	Role      fhirInterface.FhirRole
	Active    fhirInterface.FhirActive
	Reference string
}

func (pr *PractitionerRole) ById(id string) fhirInterface.IParameters {
	fmt.Printf("\t\t--> ById()\n")

	return &parameters_r4.PractitionerRoleParameters{
		Client: pr.Client,
		Uri:    "/PractitionerRole/" + id,
	}
}

func (pr *PractitionerRole) Where(option fhirInterface.UrlParameters) fhirInterface.IParameters {
	fmt.Printf("\t\t--> Where()\n")

	return &parameters_r4.PractitionerRoleParameters{
		Client:     pr.Client,
		Uri:        "/PractitionerRole",
		Parameters: option,
	}
}
