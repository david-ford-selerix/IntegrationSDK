package main

import (
	"strings"
	"testing"
)

func TestFormatXMLIndentsElements(t *testing.T) {
	formatted, err := formatXML(`<root><child>value</child><child2 attr="x" /></root>`)
	if err != nil {
		t.Fatalf("formatXML returned error: %v", err)
	}

	if !strings.Contains(formatted, "\n  <child>value</child>\n") {
		t.Fatalf("expected indented child element, got %q", formatted)
	}

	if !strings.Contains(formatted, "\n  <child2 attr=\"x\"></child2>\n") {
		t.Fatalf("expected second child element on its own line, got %q", formatted)
	}
}

func TestBuildSAMLResponseFormatsPayload(t *testing.T) {
	response := buildSAMLResponse(samlPreviewRequest{
		Version:        "2.0",
		AcsURL:         "https://example.test/saml/acs",
		EmployeeID:     "131193",
		TransmittalXML: `<Transmittal><Type>UploadApplicants</Type><Applicants><Applicant><Relationship>Employee</Relationship></Applicant></Applicants></Transmittal>`,
	})

	if !strings.Contains(response, "\n  <Issuer>Vendor</Issuer>\n") {
		t.Fatalf("expected formatted SAML response, got %q", response)
	}

	if !strings.Contains(response, "&lt;Transmittal&gt;\n  &lt;Type&gt;UploadApplicants&lt;/Type&gt;\n  &lt;Applicants&gt;") {
		t.Fatalf("expected formatted transmittal XML in AttributeValue, got %q", response)
	}
}
