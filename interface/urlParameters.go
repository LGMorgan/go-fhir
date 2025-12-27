package fhirInterface

import "net/url"

type UrlParameters struct {
	Id                string
	Name              string
	Address           string
	Role              string
	QualificationCode string
	Active            bool
	GetPages          string
	PageId            string
	BundleType        string
	Count             string
	RevInclude        string
}

func (u UrlParameters) BuildUrlValues() url.Values {
	values := url.Values{}
	if u.Name != "" {
		values.Add("name", u.Name)
	}
	if u.Address != "" {
		values.Add("address-postalcode", u.Address)
	}
	if u.Role != "" {
		values.Add("role", u.Role)
	}
	if u.QualificationCode != "" {
		values.Add("qualification-code", u.QualificationCode)
	}
	// Support v2 pagination via /_page?id=...
	if u.Id != "" {
		// FHIR search uses `_id` for resource ids
		values.Add("_id", u.Id)
	}
	if u.Active {
		values.Add("active", "true")
	}
	if u.GetPages != "" {
		values.Add("_getpages", u.GetPages)
	}
	if u.PageId != "" {
		values.Add("_pageId", u.PageId)
	}
	if u.BundleType != "" {
		values.Add("_bundletype", u.BundleType)
	}
	if u.Count != "" {
		values.Add("_count", u.Count)
	}
	if u.RevInclude != "" {
		values.Add("_revinclude", u.RevInclude)
	}
	return values
}

func (u UrlParameters) Intersection(u_cur UrlParameters) UrlParameters {
	if u_cur.Active {
		u.Active = u_cur.Active
	}
	if u_cur.QualificationCode != "" {
		u.QualificationCode = u_cur.QualificationCode
	}
	return u
}

func (u UrlParameters) Union(u_cur UrlParameters) UrlParameters {
	if u_cur.Name != "" {
		u.Name = u.Name + "," + u_cur.Name
	}
	if u_cur.Address != "" {
		u.Address = u.Address + "," + u_cur.Address
	}
	if u_cur.QualificationCode != "" {
		u.QualificationCode = u.QualificationCode + "," + u_cur.QualificationCode
	}
	return u
}

type FhirName struct {
	Value string
}

func (f FhirName) Contains() struct {
	Value func(v string) UrlParameters
} {
	return struct {
		Value func(v string) UrlParameters
	}{
		Value: func(v string) UrlParameters {
			return UrlParameters{
				Name: v,
			}
		},
	}
}

type FhirAddress struct {
	Value string
}

func (f FhirAddress) Contains() struct {
	Value func(v string) UrlParameters
} {
	return struct {
		Value func(v string) UrlParameters
	}{
		Value: func(v string) UrlParameters {
			return UrlParameters{
				Address: v,
			}
		},
	}
}

type FhirRole struct {
	Value string
}

func (f FhirRole) Contains() struct {
	Value func(v string) UrlParameters
} {
	return struct {
		Value func(v string) UrlParameters
	}{
		Value: func(v string) UrlParameters {
			return UrlParameters{
				Role: v,
			}
		},
	}
}

type FhirQualificationCode struct {
	Value string
}

func (f FhirQualificationCode) Contains() struct {
	Value func(v string) UrlParameters
} {
	return struct {
		Value func(v string) UrlParameters
	}{
		Value: func(v string) UrlParameters {
			return UrlParameters{
				QualificationCode: v,
			}
		},
	}
}

type FhirActive struct {
	Value bool
}

func (f FhirActive) IsActive() UrlParameters {
	return UrlParameters{
		Active: true,
	}
}
