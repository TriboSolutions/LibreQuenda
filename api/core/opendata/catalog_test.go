package opendata

import (
	"bytes"
	"database/sql"
	"encoding/xml"
	"fmt"
	"io"
	"librequenda/model"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func sampleCatalog() *graph {
	return buildCatalog(catalogInput{
		council: model.GetPublishedCatalogRow{
			ID:               uuid.New(),
			Slug:             "coruna",
			Name:             `Concello da Coruña & "amigos"`,
			Timezone:         "Europe/Madrid",
			ContactEmail:     sql.NullString{String: "datos@coruna.gal", Valid: true},
			ContactPhone:     sql.NullString{String: "981 18 42 00", Valid: true},
			MunicipalityCode: sql.NullString{String: "15030", Valid: true},
			Dir3Code:         "L01150301",
			IssuedAt:         sql.NullTime{Time: time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC), Valid: true},
			ModifiedAt:       time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC),
		},
		appName:  "LibreQuenda",
		webURL:   "https://reservas.example",
		apiURL:   "https://reservas.example/api/tenants/coruna/open-data",
		keywords: []string{"Pádel", "Tenis"},
		useStart: time.Date(2024, 11, 1, 0, 0, 0, 0, time.UTC),
		useEnd:   time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC),
		translate: func(key, language string, args ...any) string {
			return fmt.Sprintf("%s[%s] %v", key, language, args)
		},
	})
}

func TestCatalogRDFXMLIsWellFormed(t *testing.T) {
	var out bytes.Buffer
	if err := sampleCatalog().WriteRDFXML(&out); err != nil {
		t.Fatal(err)
	}
	var decoder = xml.NewDecoder(bytes.NewReader(out.Bytes()))
	var elements = map[string]int{}
	for {
		var token, err = decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("not well-formed XML: %v\n%s", err, out.String())
		}
		if start, ok := token.(xml.StartElement); ok {
			elements[start.Name.Space+start.Name.Local]++
		}
	}
	for element, want := range map[string]int{
		"http://www.w3.org/ns/dcat#Catalog":      1,
		"http://www.w3.org/ns/dcat#Dataset":      2,
		"http://www.w3.org/ns/dcat#Distribution": 4,
		"http://www.w3.org/2006/vcard/ns#Kind":   1,
		"http://purl.org/dc/terms/PeriodOfTime":  1,
	} {
		if elements[element] != want {
			t.Errorf("%s: %d elements, want %d", element, elements[element], want)
		}
	}
	for _, want := range []string{
		`<dct:publisher rdf:resource="http://datos.gob.es/recurso/sector-publico/org/Organismo/L01150301"/>`,
		`<dcat:themeTaxonomy rdf:resource="http://datos.gob.es/kos/sector-publico/sector"/>`,
		`<dct:language rdf:resource="http://publications.europa.eu/resource/authority/language/SPA"/>`,
		`<dct:issued rdf:datatype="http://www.w3.org/2001/XMLSchema#date">2026-11-01</dct:issued>`,
		`<dcat:downloadURL rdf:resource="https://reservas.example/api/tenants/coruna/open-data/monthly-use.csv"/>`,
		`&amp; &#34;amigos&#34;`,
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("missing %s", want)
		}
	}
}

func TestCatalogTurtle(t *testing.T) {
	var out bytes.Buffer
	if err := sampleCatalog().WriteTurtle(&out); err != nil {
		t.Fatal(err)
	}
	var ttl = out.String()
	for _, want := range []string{
		"@prefix dcat: <http://www.w3.org/ns/dcat#> .",
		"<https://reservas.example/api/tenants/coruna/open-data/catalog>\n    a dcat:Catalog ;",
		`dct:title "catalog_title[gl] [Concello da Coruña & \"amigos\"]"@gl ;`,
		`dcat:keyword "Pádel"@es ;`,
		`dct:accrualPeriodicity <http://publications.europa.eu/resource/authority/frequency/MONTHLY> ;`,
		`vcard:hasEmail <mailto:datos@coruna.gal> ;`,
		`vcard:hasTelephone <tel:+34981184200> .`,
		`dct:spatial <http://datos.gob.es/recurso/sector-publico/territorio/Provincia/A-Coruna> ;`,
		`dcat:endDate "2026-10-31"^^<http://www.w3.org/2001/XMLSchema#date> .`,
	} {
		if !strings.Contains(ttl, want) {
			t.Errorf("missing %s\n%s", want, ttl)
		}
	}
	// every statement ends: one " ." per resource (catalog, 2 datasets, a period, 4 distributions, contact)
	if got := strings.Count(ttl, " .\n") - strings.Count(ttl, "@prefix"); got != 9 {
		t.Errorf("%d statements, want 9", got)
	}
}

func TestSpatialAndPhone(t *testing.T) {
	for code, want := range map[string]string{"15030": "Provincia/A-Coruna", "36038": "Provincia/Pontevedra", "": "Pais/España", "99001": "Pais/España"} {
		if got := spatial(code); got != territoryBase+want {
			t.Errorf("spatial(%q) = %s", code, got)
		}
	}
	for phone, want := range map[string]string{"981 18 42 00": "tel:+34981184200", "+34 986-000-111": "tel:+34986000111", "": "", "—": ""} {
		if got := telURI(phone); got != want {
			t.Errorf("telURI(%q) = %q, want %q", phone, got, want)
		}
	}
}

func TestCSVCellsDefuseFormulas(t *testing.T) {
	for in, want := range map[any]string{"=SUM(A1)": "'=SUM(A1)", "Pista 1": "Pista 1", int64(3): "3", 2.5: "2.5", nil: "", true: "true"} {
		if got := cell(in); got != want {
			t.Errorf("cell(%v) = %q, want %q", in, got, want)
		}
	}
	if got := cell([]string{"1 09:00-14:00", "2 16:00-21:00"}); got != "1 09:00-14:00; 2 16:00-21:00" {
		t.Errorf("list cell = %q", got)
	}
}
