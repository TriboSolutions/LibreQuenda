package opendata

import (
	"encoding/xml"
	"io"
	"sort"
	"strings"
)

// A small RDF graph, enough for a DCAT catalog: resources named by IRI, each
// with its types and properties, written out as RDF/XML or Turtle.

type object struct {
	iri      string // a resource, or…
	value    string // …a literal
	lang     string // its language, for text
	datatype string // its type (full IRI), for dates
}

type property struct {
	predicate string // prefixed: dct:title
	object    object
}

type resource struct {
	iri        string
	types      []string // prefixed: dcat:Dataset
	properties []property
}

func (r *resource) add(predicate string, o object) *resource {
	r.properties = append(r.properties, property{predicate, o})
	return r
}

func (r *resource) link(predicate, iri string) *resource {
	return r.add(predicate, object{iri: iri})
}

func (r *resource) text(predicate, value, lang string) *resource {
	return r.add(predicate, object{value: value, lang: lang})
}

func (r *resource) literal(predicate, value string) *resource {
	return r.add(predicate, object{value: value})
}

func (r *resource) typed(predicate, value, datatype string) *resource {
	return r.add(predicate, object{value: value, datatype: datatype})
}

type graph struct {
	resources []*resource
}

func (g *graph) resource(iri string, types ...string) *resource {
	var r = &resource{iri: iri, types: types}
	g.resources = append(g.resources, r)
	return r
}

const xsdDate = "http://www.w3.org/2001/XMLSchema#date"

var namespaces = map[string]string{
	"dcat":   "http://www.w3.org/ns/dcat#",
	"dct":    "http://purl.org/dc/terms/",
	"dcatap": "http://data.europa.eu/r5r/",
	"foaf":   "http://xmlns.com/foaf/0.1/",
	"rdf":    "http://www.w3.org/1999/02/22-rdf-syntax-ns#",
	"vcard":  "http://www.w3.org/2006/vcard/ns#",
	"xsd":    "http://www.w3.org/2001/XMLSchema#",
}

func prefixes() []string {
	var names = make([]string, 0, len(namespaces))
	for name := range namespaces {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func escapeXML(value string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(value))
	return b.String()
}

// WriteRDFXML writes the graph flat: one element per resource, linked by rdf:resource.
func (g *graph) WriteRDFXML(w io.Writer) error {
	var b strings.Builder
	b.WriteString("<?xml version=\"1.0\" encoding=\"utf-8\"?>\n<rdf:RDF")
	for _, name := range prefixes() {
		b.WriteString("\n    xmlns:" + name + "=\"" + namespaces[name] + "\"")
	}
	b.WriteString(">\n")
	for _, r := range g.resources {
		var element = "rdf:Description"
		var extraTypes = r.types
		if len(r.types) > 0 {
			element, extraTypes = r.types[0], r.types[1:]
		}
		b.WriteString("  <" + element + " rdf:about=\"" + escapeXML(r.iri) + "\">\n")
		for _, t := range extraTypes {
			b.WriteString("    <rdf:type rdf:resource=\"" + escapeXML(expand(t)) + "\"/>\n")
		}
		for _, p := range r.properties {
			b.WriteString("    <" + p.predicate)
			switch {
			case p.object.iri != "":
				b.WriteString(" rdf:resource=\"" + escapeXML(p.object.iri) + "\"/>\n")
				continue
			case p.object.lang != "":
				b.WriteString(" xml:lang=\"" + p.object.lang + "\"")
			case p.object.datatype != "":
				b.WriteString(" rdf:datatype=\"" + escapeXML(p.object.datatype) + "\"")
			}
			b.WriteString(">" + escapeXML(p.object.value) + "</" + p.predicate + ">\n")
		}
		b.WriteString("  </" + element + ">\n")
	}
	b.WriteString("</rdf:RDF>\n")
	var _, err = io.WriteString(w, b.String())
	return err
}

// expand: dcat:Dataset → http://www.w3.org/ns/dcat#Dataset
func expand(prefixed string) string {
	var prefix, local, _ = strings.Cut(prefixed, ":")
	return namespaces[prefix] + local
}

var turtleString = strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`, "\r", `\r`, "\t", `\t`)

func turtleIRI(iri string) string {
	return "<" + strings.NewReplacer(">", "%3E", "<", "%3C", " ", "%20", `"`, "%22", "\\", "%5C").Replace(iri) + ">"
}

// WriteTurtle writes the graph with the same prefixes.
func (g *graph) WriteTurtle(w io.Writer) error {
	var b strings.Builder
	for _, name := range prefixes() {
		b.WriteString("@prefix " + name + ": <" + namespaces[name] + "> .\n")
	}
	for _, r := range g.resources {
		b.WriteString("\n" + turtleIRI(r.iri))
		var lines []string
		if len(r.types) > 0 {
			lines = append(lines, "a "+strings.Join(r.types, ", "))
		}
		for _, p := range r.properties {
			var o string
			switch {
			case p.object.iri != "":
				o = turtleIRI(p.object.iri)
			case p.object.lang != "":
				o = `"` + turtleString.Replace(p.object.value) + `"@` + p.object.lang
			case p.object.datatype != "":
				o = `"` + turtleString.Replace(p.object.value) + `"^^` + turtleIRI(p.object.datatype)
			default:
				o = `"` + turtleString.Replace(p.object.value) + `"`
			}
			lines = append(lines, p.predicate+" "+o)
		}
		b.WriteString("\n    " + strings.Join(lines, " ;\n    ") + " .\n")
	}
	var _, err = io.WriteString(w, b.String())
	return err
}
