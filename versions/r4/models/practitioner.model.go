package models_r4

import (
	fhirInterface "github.com/LGMorgan/go-fhir/interface"
	parameters_r4 "github.com/LGMorgan/go-fhir/versions/r4/parameters"
)

type Practitioner struct {
	Client            fhirInterface.IClient
	Address           fhirInterface.FhirAddress
	Name              fhirInterface.FhirName
	QualificationCode fhirInterface.FhirQualificationCode
	Active            fhirInterface.FhirActive
}

func (p *Practitioner) ById(id string) fhirInterface.IParameters {
	//fmt.Printf("\t\t--> ById()\n")

	// Use search on _id to allow combining with other parameters (e.g., qualification-code)
	return &parameters_r4.PractitionerParameters{
		Client: p.Client,
		Uri:    "/Practitioner",
		Parameters: fhirInterface.UrlParameters{
			SearchId: id,
		},
	}
}

func (p *Practitioner) Where(option fhirInterface.UrlParameters) fhirInterface.IParameters {
	//fmt.Printf("\t\t--> Where()\n")

	return &parameters_r4.PractitionerParameters{
		Client:     p.Client,
		Uri:        "/Practitioner",
		Parameters: option,
	}
}

func (p *Practitioner) RevInclude(query string) fhirInterface.IParameters {
	return &parameters_r4.PractitionerParameters{
		Client:     p.Client,
		Uri:        "/Practitioner",
		Parameters: fhirInterface.UrlParameters{RevInclude: query},
	}
}
