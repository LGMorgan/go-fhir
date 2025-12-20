package models_r4

import (
	"fmt"
	"net/url"

	fhirInterface "github.com/LGMorgan/go-fhir/interface"
	"github.com/LGMorgan/go-fhir/versions/r4"
)

/*
This BundleResult below is not complete

TODO: Make it complete, and make it for the other models
TODO: Make a parser for the BundleResult (and the other models)
*/
type BundleResult struct {
	Client fhirInterface.IClient
	Id     string `json:"id"`
	Link   []struct {
		Relation string `json:"relation"`
		Url      string `json:"url"`
	} `json:"link"`
	Entry []Entry `json:"entry"`
}

func (b *BundleResult) GetId() string {
	return b.Id
}
func (b *BundleResult) GetNextLink() string {
	for _, link := range b.Link {
		if link.Relation == "next" {
			return link.Url
		}
	}
	return ""
}

func (b *BundleResult) MakeRequestNextPage() (fhirInterface.IRequest, error) {
	nextLink := b.GetNextLink()
	if nextLink == "" {
		return nil, fmt.Errorf("No next link found")
	}
	u, err := url.Parse(nextLink)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	// Esante v2 may return next links like '/_page?id=...'
	if u.Path == "/_page" && q.Get("id") != "" {
		return &r4.Request{
			Client: b.Client,
			Uri:    "/_page",
			Parameters: fhirInterface.UrlParameters{
				Id: q.Get("id"),
			},
			TypeReturned: fhirInterface.BUNDLE,
		}, nil
	}
	// Fallback to HAPI-style pagination with _getpages/_pageId/_bundletype
	return &r4.Request{
		Client: b.Client,
		Uri:    "/",
		Parameters: fhirInterface.UrlParameters{
			GetPages:   q.Get("_getpages"),
			PageId:     q.Get("_pageId"),
			BundleType: q.Get("_bundletype"),
		},
		TypeReturned: fhirInterface.BUNDLE,
	}, nil
}

type Bundle struct {
	Client fhirInterface.IClient
}

func (org *Bundle) ById(id string) fhirInterface.IParameters {
	fmt.Printf("\t\t--> ById()\n")
	return nil
}

func (org *Bundle) Where(option fhirInterface.UrlParameters) fhirInterface.IParameters {
	return nil
}
