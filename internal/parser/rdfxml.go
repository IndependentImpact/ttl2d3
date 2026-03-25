package parser

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// rdfNS is the RDF namespace URI.
const rdfNS = "http://www.w3.org/1999/02/22-rdf-syntax-ns#"

// xmlNS is the XML 1998 namespace URI (used for xml:lang, xml:base, etc.).
const xmlNS = "http://www.w3.org/XML/1998/namespace"

// ParseRDFXML reads RDF/XML input from r, using baseIRI as the document base
// IRI, and returns a [Graph] of triples. It returns a non-nil error if the
// input cannot be parsed.
func ParseRDFXML(r io.Reader, baseIRI string) (*Graph, error) {
	p := &rdfxmlParser{baseIRI: baseIRI}
	if err := p.parse(r); err != nil {
		return nil, fmt.Errorf("rdfxml parse error: %w", err)
	}
	return &Graph{BaseIRI: baseIRI, Triples: p.triples}, nil
}

// rdfxmlParser holds state accumulated while parsing an RDF/XML document.
type rdfxmlParser struct {
	baseIRI  string
	triples  []Triple
	bnodeSeq int
}

func (p *rdfxmlParser) newBNode() string {
	p.bnodeSeq++
	return fmt.Sprintf("N%d", p.bnodeSeq)
}

func (p *rdfxmlParser) emit(s, pred, obj Term) {
	p.triples = append(p.triples, Triple{Subject: s, Predicate: pred, Object: obj})
}

// parse drives top-level token processing.
func (p *rdfxmlParser) parse(r io.Reader) error {
	d := xml.NewDecoder(r)
	for {
		tok, err := d.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		// rdf:RDF is the standard container; honour xml:base on it.
		if se.Name.Space == rdfNS && se.Name.Local == "RDF" {
			for _, a := range se.Attr {
				if a.Name.Space == xmlNS && a.Name.Local == "base" {
					p.baseIRI = a.Value
					break
				}
			}
			return p.parseNodeSequence(d, se.Name)
		}
		// Standalone node element (no rdf:RDF wrapper).
		_, err = p.parseNodeElement(d, se)
		return err
	}
}

// parseNodeSequence reads child node elements until the matching end element.
func (p *rdfxmlParser) parseNodeSequence(d *xml.Decoder, parent xml.Name) error {
	for {
		tok, err := d.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if _, err := p.parseNodeElement(d, t); err != nil {
				return err
			}
		case xml.EndElement:
			if t.Name == parent {
				return nil
			}
		}
	}
}

// parseNodeElement parses an RDF node element and returns its subject term.
func (p *rdfxmlParser) parseNodeElement(d *xml.Decoder, se xml.StartElement) (Term, error) {
	subject := p.subjectTerm(se.Attr)

	// Typed node: element name ≠ rdf:Description → emit rdf:type triple.
	if !(se.Name.Space == rdfNS && se.Name.Local == "Description") && se.Name.Space != "" {
		p.emit(
			subject,
			Term{Kind: TermIRI, Value: rdfNS + "type"},
			Term{Kind: TermIRI, Value: se.Name.Space + se.Name.Local},
		)
	}

	// Process attributes on the node element.
	for _, a := range se.Attr {
		switch {
		case a.Name.Space == rdfNS && a.Name.Local == "type":
			// rdf:type attribute generates an additional type triple.
			p.emit(
				subject,
				Term{Kind: TermIRI, Value: rdfNS + "type"},
				Term{Kind: TermIRI, Value: a.Value},
			)
		case isNodeStructuralAttr(a.Name):
			// about / ID / nodeID already consumed by subjectTerm – skip.
		case a.Name.Space == rdfNS:
			// All other rdf: attributes are structural – skip.
		case a.Name.Space == xmlNS || a.Name.Space == "":
			// XML-namespace and unqualified attributes – skip.
		default:
			// Property attribute: generates a plain literal triple.
			pred := Term{Kind: TermIRI, Value: a.Name.Space + a.Name.Local}
			p.emit(subject, pred, Term{Kind: TermLiteral, Value: a.Value})
		}
	}

	liIndex := 1
	if err := p.parsePropElements(d, se.Name, subject, &liIndex); err != nil {
		return Term{}, err
	}
	return subject, nil
}

// subjectTerm determines the subject Term from a node element's attributes.
func (p *rdfxmlParser) subjectTerm(attrs []xml.Attr) Term {
	for _, a := range attrs {
		if a.Name.Space != rdfNS {
			continue
		}
		switch a.Name.Local {
		case "about":
			return Term{Kind: TermIRI, Value: a.Value}
		case "ID":
			return Term{Kind: TermIRI, Value: stripFragment(p.baseIRI) + "#" + a.Value}
		case "nodeID":
			return Term{Kind: TermBlank, Value: a.Value}
		}
	}
	return Term{Kind: TermBlank, Value: p.newBNode()}
}

// parsePropElements reads property child elements until the matching end element.
func (p *rdfxmlParser) parsePropElements(d *xml.Decoder, parent xml.Name, subject Term, liIndex *int) error {
	for {
		tok, err := d.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if err := p.parsePropElement(d, t, subject, liIndex); err != nil {
				return err
			}
		case xml.EndElement:
			if t.Name == parent {
				return nil
			}
		}
	}
}

// parsePropElement processes a single property element child.
func (p *rdfxmlParser) parsePropElement(d *xml.Decoder, se xml.StartElement, subject Term, liIndex *int) error {
	pred := predTerm(se, liIndex)

	// rdf:resource → IRI object; element must be empty.
	for _, a := range se.Attr {
		if a.Name.Space == rdfNS && a.Name.Local == "resource" {
			p.emit(subject, pred, Term{Kind: TermIRI, Value: a.Value})
			return consumeToEnd(d, se.Name)
		}
	}

	// rdf:nodeID → blank-node object; element must be empty.
	for _, a := range se.Attr {
		if a.Name.Space == rdfNS && a.Name.Local == "nodeID" {
			p.emit(subject, pred, Term{Kind: TermBlank, Value: a.Value})
			return consumeToEnd(d, se.Name)
		}
	}

	// rdf:parseType dispatches to specialised handlers.
	for _, a := range se.Attr {
		if a.Name.Space == rdfNS && a.Name.Local == "parseType" {
			switch a.Value {
			case "Resource":
				bnode := Term{Kind: TermBlank, Value: p.newBNode()}
				p.emit(subject, pred, bnode)
				inner := 1
				return p.parsePropElements(d, se.Name, bnode, &inner)
			case "Literal":
				raw, err := consumeRawXML(d, se.Name)
				if err != nil {
					return err
				}
				p.emit(subject, pred, Term{
					Kind:     TermLiteral,
					Value:    raw,
					Datatype: rdfNS + "XMLLiteral",
				})
				return nil
			case "Collection":
				return p.parseCollection(d, se.Name, subject, pred)
			}
		}
	}

	// Default: text content or a single nested node element.
	return p.parsePropContent(d, se, subject, pred)
}

// parsePropContent reads the content of a plain property element (no
// rdf:resource / rdf:nodeID / rdf:parseType) and emits the appropriate triple.
func (p *rdfxmlParser) parsePropContent(d *xml.Decoder, propElem xml.StartElement, subject Term, pred Term) error {
	lang := ""
	datatype := ""
	for _, a := range propElem.Attr {
		switch {
		case a.Name.Space == xmlNS && a.Name.Local == "lang":
			lang = a.Value
		case a.Name.Space == rdfNS && a.Name.Local == "datatype":
			datatype = a.Value
		}
	}

	var text strings.Builder
	var nodeObj *Term

	for {
		tok, err := d.Token()
		if err == io.EOF {
			return fmt.Errorf("unexpected EOF inside property element <%s>", propElem.Name.Local)
		}
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.CharData:
			text.Write([]byte(t))
		case xml.StartElement:
			subj, err := p.parseNodeElement(d, t)
			if err != nil {
				return err
			}
			nodeObj = &subj
		case xml.EndElement:
			if t.Name == propElem.Name {
				if nodeObj != nil {
					p.emit(subject, pred, *nodeObj)
				} else {
					p.emit(subject, pred, Term{
						Kind:     TermLiteral,
						Value:    text.String(),
						Language: lang,
						Datatype: datatype,
					})
				}
				return nil
			}
		}
	}
}

// parseCollection handles rdf:parseType="Collection", building an RDF list.
func (p *rdfxmlParser) parseCollection(d *xml.Decoder, parent xml.Name, subject Term, pred Term) error {
	var items []Term
	for {
		tok, err := d.Token()
		if err == io.EOF {
			return fmt.Errorf("unexpected EOF inside rdf:parseType=\"Collection\"")
		}
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			item, err := p.parseNodeElement(d, t)
			if err != nil {
				return err
			}
			items = append(items, item)
		case xml.EndElement:
			if t.Name == parent {
				if len(items) == 0 {
					p.emit(subject, pred, Term{Kind: TermIRI, Value: rdfNS + "nil"})
					return nil
				}
				// Allocate list-node blank nodes up front.
				nodes := make([]Term, len(items))
				for i := range nodes {
					nodes[i] = Term{Kind: TermBlank, Value: p.newBNode()}
				}
				p.emit(subject, pred, nodes[0])
				for i, item := range items {
					p.emit(nodes[i], Term{Kind: TermIRI, Value: rdfNS + "first"}, item)
					if i < len(items)-1 {
						p.emit(nodes[i], Term{Kind: TermIRI, Value: rdfNS + "rest"}, nodes[i+1])
					} else {
						p.emit(nodes[i], Term{Kind: TermIRI, Value: rdfNS + "rest"},
							Term{Kind: TermIRI, Value: rdfNS + "nil"})
					}
				}
				return nil
			}
		}
	}
}

// predTerm returns the predicate IRI term for a property element, honouring
// rdf:li container membership shorthand.
func predTerm(se xml.StartElement, liIndex *int) Term {
	if se.Name.Space == rdfNS && se.Name.Local == "li" {
		iri := fmt.Sprintf("%s_%d", rdfNS, *liIndex)
		*liIndex++
		return Term{Kind: TermIRI, Value: iri}
	}
	return Term{Kind: TermIRI, Value: se.Name.Space + se.Name.Local}
}

// isNodeStructuralAttr reports whether name is a structural node-element
// attribute (about, ID, nodeID) that must not generate a property triple.
func isNodeStructuralAttr(name xml.Name) bool {
	if name.Space != rdfNS {
		return false
	}
	switch name.Local {
	case "about", "ID", "nodeID":
		return true
	}
	return false
}

// stripFragment removes the fragment component (from '#' onward) of an IRI.
func stripFragment(iri string) string {
	if i := strings.IndexByte(iri, '#'); i >= 0 {
		return iri[:i]
	}
	return iri
}

// consumeToEnd discards tokens up to and including the matching end element.
func consumeToEnd(d *xml.Decoder, parent xml.Name) error {
	depth := 0
	for {
		tok, err := d.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		switch tok.(type) {
		case xml.StartElement:
			depth++
		case xml.EndElement:
			if depth == 0 {
				return nil
			}
			depth--
		}
	}
}

// consumeRawXML reads element content using Token() and returns a serialised
// XML string, suitable for use as an rdf:XMLLiteral value.
func consumeRawXML(d *xml.Decoder, parent xml.Name) (string, error) {
	var buf strings.Builder
	depth := 0
	for {
		tok, err := d.Token()
		if err == io.EOF {
			return "", fmt.Errorf("unexpected EOF inside rdf:XMLLiteral")
		}
		if err != nil {
			return "", err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			depth++
			buf.WriteByte('<')
			buf.WriteString(t.Name.Local)
			for _, a := range t.Attr {
				buf.WriteByte(' ')
				buf.WriteString(a.Name.Local)
				buf.WriteString(`="`)
				xml.Escape(&buf, []byte(a.Value))
				buf.WriteByte('"')
			}
			buf.WriteByte('>')
		case xml.EndElement:
			if depth == 0 {
				return buf.String(), nil
			}
			depth--
			buf.WriteString("</")
			buf.WriteString(t.Name.Local)
			buf.WriteByte('>')
		case xml.CharData:
			xml.Escape(&buf, []byte(t))
		}
	}
}
