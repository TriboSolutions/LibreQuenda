package opendata

import (
	"librequenda/model"
	"strings"
	"time"
)

// The council's DCAT-AP-ES catalog (https://datosgobes.github.io/DCAT-AP-ES/):
// two datasets, its facilities and their monthly use, each as CSV and JSON.
// datos.gob.es harvests it from catalog.rdf; the publisher is the council's
// DIR3 code, registered there.

const (
	dir3Base      = "http://datos.gob.es/recurso/sector-publico/org/Organismo/"
	sectorBase    = "http://datos.gob.es/kos/sector-publico/sector"
	euLanguage    = "http://publications.europa.eu/resource/authority/language/"
	euFileType    = "http://publications.europa.eu/resource/authority/file-type/"
	euFrequency   = "http://publications.europa.eu/resource/authority/frequency/"
	licenseCCBY40 = "http://publications.europa.eu/resource/authority/licence/CC_BY_4_0"
	ianaMediaType = "http://www.iana.org/assignments/media-types/"
	territoryBase = "http://datos.gob.es/recurso/sector-publico/territorio/"
	availableBase = "http://publications.europa.eu/resource/authority/planned-availability/"
)

// the languages the metadata comes in: the app's, with their EU authority codes
var metadataLanguages = []struct{ tag, eu string }{
	{"es", "SPA"}, {"gl", "GLG"}, {"ca", "CAT"}, {"en", "ENG"},
}

type catalogInput struct {
	council   model.GetPublishedCatalogRow
	appName   string
	webURL    string // the web app, e.g. https://reservas.concello.gal
	apiURL    string // this council's open data, e.g. https://reservas.concello.gal/api/tenants/lalin/open-data
	keywords  []string
	translate func(key, language string, args ...any) string
	useStart  time.Time // the monthly use covers useStart…useEnd (its last day)
	useEnd    time.Time
}

// each dataset comes as CSV and JSON
var formats = []struct{ ext, euType, media string }{
	{"csv", "CSV", "text/csv"},
	{"json", "JSON", "application/json"},
}

type datasetSpec struct {
	id         string // also the file name: facilities.csv, monthly-use.json
	key        string // its texts: <key>_title, <key>_description, <key>_keywords
	frequency  string
	keywords   []string   // besides its own: the facilities' tags
	start, end *time.Time // the period it covers, when it has one
}

// INE province codes (the first two digits of a municipality's) and their
// territory IRIs at datos.gob.es
var provinces = map[string]string{
	"01": "Alava", "02": "Albacete", "03": "Alicante", "04": "Almeria", "05": "Avila", "06": "Badajoz",
	"07": "Illes-Balears", "08": "Barcelona", "09": "Burgos", "10": "Caceres", "11": "Cadiz", "12": "Castellon",
	"13": "Ciudad-Real", "14": "Cordoba", "15": "A-Coruna", "16": "Cuenca", "17": "Girona", "18": "Granada",
	"19": "Guadalajara", "20": "Guipuzcoa", "21": "Huelva", "22": "Huesca", "23": "Jaen", "24": "Leon",
	"25": "Lleida", "26": "La-Rioja", "27": "Lugo", "28": "Madrid", "29": "Malaga", "30": "Murcia",
	"31": "Navarra", "32": "Ourense", "33": "Asturias", "34": "Palencia", "35": "Las-Palmas", "36": "Pontevedra",
	"37": "Salamanca", "38": "Santa-Cruz-Tenerife", "39": "Cantabria", "40": "Segovia", "41": "Sevilla", "42": "Soria",
	"43": "Tarragona", "44": "Teruel", "45": "Toledo", "46": "Valencia", "47": "Valladolid", "48": "Vizcaya",
	"49": "Zamora", "50": "Zaragoza", "51": "Ceuta", "52": "Melilla",
}

// spatial: the council's province, from its INE code; Spain without one.
func spatial(municipalityCode string) string {
	if len(municipalityCode) == 5 {
		if province, ok := provinces[municipalityCode[:2]]; ok {
			return territoryBase + "Provincia/" + province
		}
	}
	return territoryBase + "Pais/España"
}

func buildCatalog(in catalogInput) *graph {
	var g = &graph{}
	var c = in.council
	var publisher = dir3Base + c.Dir3Code
	var issued = c.ModifiedAt
	if c.IssuedAt.Valid {
		issued = c.IssuedAt.Time
	}
	var date = func(t time.Time) string { return t.Format(time.DateOnly) }
	var homepage = in.webURL + "/" + c.Slug
	var place = spatial(c.MunicipalityCode.String)

	var catalog = g.resource(in.apiURL+"/catalog", "dcat:Catalog")
	for _, l := range metadataLanguages {
		catalog.text("dct:title", in.translate("catalog_title", l.tag, c.Name), l.tag)
	}
	for _, l := range metadataLanguages {
		catalog.text("dct:description", in.translate("catalog_description", l.tag, c.Name, in.appName), l.tag)
	}
	catalog.link("dct:publisher", publisher).
		link("foaf:homepage", homepage).
		link("dcat:themeTaxonomy", sectorBase).
		link("dct:spatial", place).
		typed("dct:issued", date(issued), xsdDate).
		typed("dct:modified", date(c.ModifiedAt), xsdDate).
		link("dct:license", licenseCCBY40)
	for _, l := range metadataLanguages {
		catalog.link("dct:language", euLanguage+l.eu)
	}

	// the council itself, as residents reach it
	var contact = in.apiURL + "/catalog#contact"

	var datasets = []datasetSpec{
		{id: "facilities", key: "facilities", frequency: "IRREG", keywords: in.keywords},
		{id: "monthly-use", key: "use", frequency: "MONTHLY", start: &in.useStart, end: &in.useEnd},
	}
	for _, spec := range datasets {
		catalog.link("dcat:dataset", in.apiURL+"/dataset/"+spec.id)
	}

	for _, spec := range datasets {
		var iri = in.apiURL + "/dataset/" + spec.id
		var dataset = g.resource(iri, "dcat:Dataset").literal("dct:identifier", iri)
		for _, l := range metadataLanguages {
			dataset.text("dct:title", in.translate(spec.key+"_title", l.tag, c.Name), l.tag)
		}
		for _, l := range metadataLanguages {
			dataset.text("dct:description", in.translate(spec.key+"_description", l.tag, c.Name), l.tag)
		}
		dataset.link("dct:publisher", publisher).
			link("dcat:theme", sectorBase+"/deporte").
			link("dcat:theme", sectorBase+"/sociedad-bienestar").
			link("dcat:landingPage", homepage).
			link("dct:spatial", place).
			link("dct:accrualPeriodicity", euFrequency+spec.frequency).
			typed("dct:issued", date(issued), xsdDate).
			typed("dct:modified", date(c.ModifiedAt), xsdDate)
		for _, l := range metadataLanguages {
			dataset.link("dct:language", euLanguage+l.eu)
		}
		for _, l := range metadataLanguages {
			for _, keyword := range strings.Split(in.translate(spec.key+"_keywords", l.tag), ",") {
				if keyword = strings.TrimSpace(keyword); keyword != "" {
					dataset.text("dcat:keyword", keyword, l.tag)
				}
			}
		}
		for _, keyword := range spec.keywords {
			dataset.text("dcat:keyword", keyword, "es")
		}
		dataset.link("dcat:contactPoint", contact)
		if spec.start != nil {
			var period = iri + "#period"
			dataset.link("dct:temporal", period)
			g.resource(period, "dct:PeriodOfTime").
				typed("dcat:startDate", date(*spec.start), xsdDate).
				typed("dcat:endDate", date(*spec.end), xsdDate)
		}
		for _, format := range formats {
			dataset.link("dcat:distribution", iri+"/"+format.ext)
		}
	}

	for _, spec := range datasets {
		var iri = in.apiURL + "/dataset/" + spec.id
		for _, format := range formats {
			var file = in.apiURL + "/" + spec.id + "." + format.ext
			var distribution = g.resource(iri+"/"+format.ext, "dcat:Distribution")
			for _, l := range metadataLanguages {
				distribution.text("dct:title", in.translate(spec.key+"_title", l.tag, c.Name)+" ("+format.euType+")", l.tag)
			}
			for _, l := range metadataLanguages {
				distribution.text("dct:description", in.translate(spec.key+"_description", l.tag, c.Name), l.tag)
			}
			distribution.link("dcat:accessURL", file).
				link("dcat:downloadURL", file).
				link("dct:format", euFileType+format.euType).
				link("dcat:mediaType", ianaMediaType+format.media).
				link("dct:license", licenseCCBY40).
				link("dcatap:availability", availableBase+"AVAILABLE")
		}
	}

	var point = g.resource(contact, "vcard:Kind")
	for _, l := range metadataLanguages {
		point.text("vcard:fn", c.Name, l.tag)
	}
	for _, l := range metadataLanguages {
		point.text("vcard:organization-name", c.Name, l.tag)
	}
	point.link("vcard:hasUID", publisher).link("vcard:hasURL", homepage)
	if c.ContactEmail.Valid && c.ContactEmail.String != "" {
		point.link("vcard:hasEmail", "mailto:"+c.ContactEmail.String)
	}
	if phone := telURI(c.ContactPhone.String); phone != "" {
		point.link("vcard:hasTelephone", phone)
	}
	return g
}

// telURI: a phone number as tel:+34-981000000, keeping only + and digits; "" without digits.
func telURI(phone string) string {
	var digits strings.Builder
	for i, r := range strings.TrimSpace(phone) {
		if (r >= '0' && r <= '9') || (r == '+' && i == 0) {
			digits.WriteRune(r)
		}
	}
	var number = digits.String()
	if strings.Trim(number, "+") == "" {
		return ""
	}
	if !strings.HasPrefix(number, "+") {
		number = "+34" + number // a Spanish council's number, written without the prefix
	}
	return "tel:" + number
}
